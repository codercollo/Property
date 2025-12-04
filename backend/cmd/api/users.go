package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/codercollo/property/backend/internal/data"
	"github.com/codercollo/property/backend/internal/validator"
)

// RegisterUserHandler handles user registration with role-based permissions.
func (app *application) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	// Parse user registration input from JSON body
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role,omitempty"` // Optional role field, defaults to "user"
	}
	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Default role assignment for new users
	if input.Role == "" {
		input.Role = "user"
	}

	// Create a new user struct for storage
	user := &data.User{
		Name:      input.Name,
		Email:     input.Email,
		Activated: false,
		Role:      input.Role,
	}

	// Hash and set the user's password
	err = user.Password.Set(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Validate the user fields
	v := validator.New()
	if data.ValidateUser(v, user); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Insert the new user in the database
	if err := app.models.Users.Insert(user); err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddError("email", "a user with this email address already exists")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Assign permissions based on user role
	err = app.grandRolePermissions(user.ID, Role(input.Role))
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Generate a new activation token
	token, err := app.models.Tokens.New(user.ID, 3*24*time.Hour, data.ScopeActivation)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Send welcome email asynchronously
	app.background(func() {
		data := map[string]interface{}{
			"activationToken": token.Plaintext,
			"userID":          user.ID,
		}
		if err := app.mailer.Send(user.Email, "user_welcome.tmpl", data); err != nil {
			app.logger.PrintError(err, nil)
		}
	})

	// Respond with 202 Accepted containing the new user
	err = app.writeJSON(w, http.StatusAccepted, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// activateUserHandler validates the activation token, activates the user account,
// deletes the token and returns the updated user in the response
func (app *application) activateUserHandler(w http.ResponseWriter, r *http.Request) {
	//Parse token from request body
	var input struct {
		TokenPlaintext string `json:"token"`
	}
	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	//validate token format
	v := validator.New()
	if data.ValidateTokenPlaintext(v, input.TokenPlaintext); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	//Get user associated with the token
	user, err := app.models.Users.GetForToken(data.ScopeActivation, input.TokenPlaintext)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrUserNotFound):
			v.AddError("token", "invalid or expired activation token")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	//Activate the user
	user.Activated = true
	if err := app.models.Users.Update(user); err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	//Delete all activation tokens for the user
	if err := app.models.Tokens.DeleteAllForUser(data.ScopeActivation, user.ID); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	//Return updated user data
	if err := app.writeJSON(w, http.StatusOK, envelope{"user": user}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}

}

// updateUserRoleHandler allows admins to update a user's role and permissions
func (app *application) updateUserRoleHandler(w http.ResponseWriter, r *http.Request) {
	//Extract user ID from URL parameter
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	//Parse the new role from request body
	var input struct {
		Role string `json:"role"`
	}
	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	//Validate the role
	v := validator.New()
	data.ValidateRole(v, input.Role)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	//Get the user from database
	user, err := app.models.Users.GetByID(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	//Update the user's role
	user.Role = input.Role
	err = app.models.Users.Update(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	//Remove all existing persmissions for the user
	err = app.models.Permissions.RemoveAllForUser(user.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	//Grant new permissions based on the new role
	err = app.grandRolePermissions(user.ID, Role(input.Role))
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	//Refetch the user
	updatedUser, err := app.models.Users.GetByID(id)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	//Return the updated user
	err = app.writeJSON(w, http.StatusOK, envelope{"user": updatedUser}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
