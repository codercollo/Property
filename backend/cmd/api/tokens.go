package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/codercollo/property/backend/internal/data"
	"github.com/codercollo/property/backend/internal/validator"
)

// createAuthenticationTokenHandler handles {LOGIN HANDLER} user login and issues an authentication token
func (app *application) createAuthenticationTokenHandler(w http.ResponseWriter, r *http.Request) {
	//Parse email and password from request body
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	//Validate email and password format
	v := validator.New()
	data.ValidateEmail(v, input.Email)
	data.ValidatePasswordPlaintext(v, input.Password)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	//Lookup user by email
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

	//Verify password
	match, err := user.Password.Matches(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	if !match {
		app.invalidCredentialsResponse(w, r)
		return
	}

	//Generate new authentication token (24hrs expiry)
	token, err := app.models.Tokens.New(user.ID, 24*time.Hour, data.ScopeAuthentication)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	//Return token in JSON  with 201 Created
	if err := app.writeJSON(w, http.StatusCreated, envelope{"authentication_token": token}, nil); err != nil {
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
