package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/codercollo/property/backend/internal/validator"
	"github.com/julienschmidt/httprouter"
)

// Define an envelope type
type envelope map[string]interface{}

// Extracts and validates the "id" URL parameter from the request

func (app *application) readIDParam(r *http.Request) (int64, error) {
	//Get URL parameters from request context
	params := httprouter.ParamsFromContext(r.Context())

	//Parse "id" parameter as integer and validate
	id, err := strconv.ParseInt(params.ByName("id"), 10, 64)
	if err != nil || id < 1 {
		return 0, errors.New("invalid id parameter")
	}

	//Return the validated ID
	return id, nil
}

// Sends a JSON response with optional headers and a status code use type envelope
func (app *application) writeJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) error {
	//Encode the data to JSON with indentation
	js, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}

	//Append newline for readability in terminal
	js = append(js, '\n')

	//Add any additional header if provided
	for key, value := range headers {
		w.Header()[key] = value

	}

	//Set Content-Type, write status code, and send JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)

	return nil

}

// readJSON decodes the request body into dst and provides detailed JSON error handling
func (app *application) readJSON(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	//Limit request body to 1MB
	maxBytes := 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	//Set up decoder and forbid unknown fields
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	//Decode into destination
	err := dec.Decode(dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalErro *json.InvalidUnmarshalError

		switch {
		//Bad JSON sytnax
		case errors.As(err, &syntaxError):
			return fmt.Errorf("bad JSON (at character %d)", syntaxError.Offset)

			//Another form of syntax error
		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("bad JSON")

			//Wrong type for a field
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("wrong JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("wrong JSON type (at character %d)", unmarshalTypeError.Offset)

			//Empty body
		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")

			//Developer error: dst wasn't a pointer
		case errors.As(err, &invalidUnmarshalErro):
			panic(err)

			//Any other error
		default:
			return err

		}
	}

	//Ensure only one JSON value is sent
	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("body must contain only a single JSON value")
	}

	return nil
}

// readString returns the query string value for a key or default if missing
// eg: ?name=Collins = "Collins" or if missing default "Guest"
func (app *application) readString(qs url.Values, key string, defaultValue string) string {
	s := qs.Get(key)
	if s == "" {
		return defaultValue
	}
	return s
}

// readCSV returns a comma-separated query string as a slice or a default if missing
// eg: ?tags=go,web,api = []string{"go", "web", "api"}
func (app *application) readCSV(qs url.Values, key string, defaultValue []string) []string {
	csv := qs.Get(key)
	if csv == "" {
		return defaultValue
	}
	return strings.Split(csv, ",")
}

// readInt returns the query string value as an int or a default if missing/invalid, recording errors
// eg: ?age25 = 25 or if invalid default 0
func (app *application) readInt(qs url.Values, key string, defaultValue int, v *validator.Validator) int {
	s := qs.Get(key)
	if s == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		v.AddError(key, "must be an integer value")
		return defaultValue
	}
	return i
}

// background runs the given function in a safe background goroutine
func (app *application) background(fn func()) {

	//Increament the WaitGroup counter
	app.wg.Add(1)

	go func() {

		//Lauch the background goroutine
		defer app.wg.Done()

		//Recover and log any panic in the goroutine
		defer func() {
			if err := recover(); err != nil {
				app.logger.PrintError(fmt.Errorf("%s", err), nil)
			}
		}()
		//Execute the provided function
		fn()
	}()
}

// invalidAuthenticationTokenResponse sends a 401 response when the auth token is missing or invalid
func (app *application) invalidAuthenticationTokenResponse(w http.ResponseWriter, r *http.Request) {
	//Tell client authentication must use a Bearer token
	w.Header().Set("WWW-Authenticate", "Bearer")

	//Return standardized 401 Unauthorized error
	message := "invalid or missing authentication token"
	app.errorResponse(w, r, http.StatusUnauthorized, message)

}

// isAgent chacks if a user is an agent
func (app *application) isAgent(r *http.Request) bool {
	user := app.contextGetUser(r)
	return user.Role == "agent"
}

// isAdmin chack if a user is an admin
func (app *application) isAdmin(r *http.Request) bool {
	user := app.contextGetUser(r)
	return user.Role == "admin"
}
