package main

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/codercollo/property/backend/internal/data"
	"github.com/codercollo/property/backend/internal/validator"
	"github.com/pascaldekloe/jwt"
)

// createAuthenticationTokenHandler handles {LOGIN HANDLER} user login and issues an authentication token
// Handles user authentication and returns a JWT token.
func (app *application) createAuthenticationTokenHandler(w http.ResponseWriter, r *http.Request) {
	// Parse request input
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Validate input
	v := validator.New()
	data.ValidateEmail(v, input.Email)
	data.ValidatePasswordPlaintext(v, input.Password)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Retrieve user
	user, err := app.models.Users.GetByEmail(input.Email)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrUserNotFound):
			app.invalidCredentialsResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Check password
	match, err := user.Password.Matches(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	if !match {
		app.invalidCredentialsResponse(w, r)
		return
	}

	// Create JWT claims
	var claims jwt.Claims
	claims.Subject = strconv.FormatInt(user.ID, 10)
	claims.Issued = jwt.NewNumericTime(time.Now())
	claims.NotBefore = jwt.NewNumericTime(time.Now())
	claims.Expires = jwt.NewNumericTime(time.Now().Add(24 * time.Hour))
	claims.Issuer = "propertyown.api"
	claims.Audiences = []string{"propertyown.api"}

	// Sign JWT
	jwtBytes, err := claims.HMACSign(jwt.HS256, []byte(app.config.jwt.secret))
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Return token in response
	err = app.writeJSON(
		w,
		http.StatusCreated,
		envelope{"authentication_token": string(jwtBytes)},
		nil,
	)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// Handles password reset token creation and email delivery.
func (app *application) createPasswordResetTokenHandler(w http.ResponseWriter, r *http.Request) {
	// Parse request input
	var input struct {
		Email string `json:"email"`
	}

	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Validate email address
	v := validator.New()

	if data.ValidateEmail(v, input.Email); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Retrieve user by email
	user, err := app.models.Users.GetByEmail(input.Email)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrUserNotFound):
			v.AddError("email", "no matching email address found")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Check user activation status
	if !user.Activated {
		v.AddError("email", "user account must be activated")
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Generate password reset token
	token, err := app.models.Tokens.New(
		user.ID,
		45*time.Minute,
		data.ScopePasswordReset,
	)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Send password reset email asynchronously
	app.background(func() {
		data := map[string]interface{}{
			"passwordResetToken": token.Plaintext,
		}

		if err := app.mailer.Send(
			user.Email,
			"token_password_reset.tmpl",
			data,
		); err != nil {
			app.logger.PrintError(err, nil)
		}
	})

	// Respond with acceptance message
	env := envelope{
		"message": "an email will be sent to you containing password reset instructions",
	}

	if err := app.writeJSON(w, http.StatusAccepted, env, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// Handles creation of an account activation token and emails it to the user.
func (app *application) createActivationTokenHandler(w http.ResponseWriter, r *http.Request) {
	// Parse request input
	var input struct {
		Email string `json:"email"`
	}

	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Validate email address
	v := validator.New()
	if data.ValidateEmail(v, input.Email); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Retrieve user by email
	user, err := app.models.Users.GetByEmail(input.Email)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrUserNotFound):
			v.AddError("email", "no matching email address found")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Ensure the user is not already activated
	if user.Activated {
		v.AddError("email", "user has already been activated")
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Generate a new activation token
	token, err := app.models.Tokens.New(
		user.ID,
		3*24*time.Hour,
		data.ScopeActivation,
	)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Send activation email asynchronously
	app.background(func() {
		data := map[string]interface{}{
			"activationToken": token.Plaintext,
		}

		if err := app.mailer.Send(user.Email, "token_activation.tmpl", data); err != nil {
			app.logger.PrintError(err, nil)
		}
	})

	// Respond with confirmation message
	env := envelope{
		"message": "an email will be sent to you containing activation instructions",
	}

	if err := app.writeJSON(w, http.StatusAccepted, env, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
