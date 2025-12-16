package main

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/codercollo/property/backend/internal/data"
	"github.com/codercollo/property/backend/internal/validator"
	"github.com/pascaldekloe/jwt"
)

// createAuthenticationTokenHandler handles user login and issues an authentication token
func (app *application) createAuthenticationTokenHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	data.ValidateEmail(v, input.Email)
	data.ValidatePasswordPlaintext(v, input.Password)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

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

	jwtBytes, err := claims.HMACSign(jwt.HS256, []byte(app.config.jwt.secret))
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(
		w,
		http.StatusCreated,
		envelope{
			"authentication_token": string(jwtBytes),
			"user":                 user,
		},
		nil,
	)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// revokeAuthenticationTokenHandler handles token revocation (logout)
func (app *application) revokeAuthenticationTokenHandler(w http.ResponseWriter, r *http.Request) {
	// Get the token from Authorization header
	authorizationHeader := r.Header.Get("Authorization")
	if authorizationHeader == "" {
		app.authenticationRequiredResponse(w, r)
		return
	}

	headerParts := strings.Split(authorizationHeader, " ")
	if len(headerParts) != 2 || headerParts[0] != "Bearer" {
		app.invalidAuthenticationTokenResponse(w, r)
		return
	}
	token := headerParts[1]

	// Verify the token first
	claims, err := jwt.HMACCheck([]byte(token), []byte(app.config.jwt.secret))
	if err != nil {
		app.invalidAuthenticationTokenResponse(w, r)
		return
	}

	if !claims.Valid(time.Now()) {
		app.invalidAuthenticationTokenResponse(w, r)
		return
	}

	// Extract user ID from claims
	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Get the authenticated user from context (for additional verification)
	user := app.contextGetUser(r)
	if user.IsAnonymous() {
		app.authenticationRequiredResponse(w, r)
		return
	}

	// Verify the token belongs to the authenticated user
	if user.ID != userID {
		app.notPermittedResponse(w, r)
		return
	}

	// Revoke the token by adding it to the revoked tokens table
	err = app.models.RevokedTokens.Insert(token, userID, claims.Expires.Time())
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Return success response
	err = app.writeJSON(w, http.StatusOK, envelope{
		"message": "token successfully revoked",
	}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// revokeAllTokensHandler revokes all tokens for the user
func (app *application) revokeAllTokensHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	if user.IsAnonymous() {
		app.authenticationRequiredResponse(w, r)
		return
	}

	// Get token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		app.authenticationRequiredResponse(w, r)
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		app.invalidAuthenticationTokenResponse(w, r)
		return
	}
	token := parts[1]

	// Verify token and extract claims
	claims, err := jwt.HMACCheck([]byte(token), []byte(app.config.jwt.secret))
	if err != nil {
		app.invalidAuthenticationTokenResponse(w, r)
		return
	}

	// Revoke current token
	err = app.models.RevokedTokens.Insert(token, user.ID, claims.Expires.Time())
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Revoke all other tokens for the user
	err = app.models.RevokedTokens.RevokeAllForUser(user.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Respond with success
	err = app.writeJSON(w, http.StatusOK, envelope{
		"message": "all tokens successfully revoked",
	}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// createPasswordResetTokenHandler handles password reset token creation
func (app *application) createPasswordResetTokenHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email string `json:"email"`
	}

	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	if data.ValidateEmail(v, input.Email); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

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

	if !user.Activated {
		v.AddError("email", "user account must be activated")
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	token, err := app.models.Tokens.New(
		user.ID,
		45*time.Minute,
		data.ScopePasswordReset,
	)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

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

	env := envelope{
		"message": "an email will be sent to you containing password reset instructions",
	}

	if err := app.writeJSON(w, http.StatusAccepted, env, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// createActivationTokenHandler handles account activation token creation
func (app *application) createActivationTokenHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email string `json:"email"`
	}

	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	if data.ValidateEmail(v, input.Email); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

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

	if user.Activated {
		v.AddError("email", "user has already been activated")
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	token, err := app.models.Tokens.New(
		user.ID,
		3*24*time.Hour,
		data.ScopeActivation,
	)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	app.background(func() {
		data := map[string]interface{}{
			"activationToken": token.Plaintext,
		}

		if err := app.mailer.Send(user.Email, "token_activation.tmpl", data); err != nil {
			app.logger.PrintError(err, nil)
		}
	})

	env := envelope{
		"message": "an email will be sent to you containing activation instructions",
	}

	if err := app.writeJSON(w, http.StatusAccepted, env, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
