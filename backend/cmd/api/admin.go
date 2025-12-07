package main

import (
	"errors"
	"net/http"

	"github.com/codercollo/property/backend/internal/data"
	"github.com/codercollo/property/backend/internal/validator"
)

// =============================================================================
// ADMIN SELF-MANAGEMENT
// =============================================================================

// getAdminProfileHandler retrieves the admin's profile
func (app *application) getAdminProfileHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	err := app.writeJSON(w, http.StatusOK, envelope{"admin": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// updateAdminProfileHandler updates admin's name and email
func (app *application) updateAdminProfileHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	var input struct {
		Name  *string `json:"name"`
		Email *string `json:"email"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if input.Name != nil {
		user.Name = *input.Name
	}
	if input.Email != nil {
		user.Email = *input.Email
	}

	v := validator.New()
	if data.ValidateUser(v, user); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Users.Update(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddError("email", "a user with this email address already exists")
			app.failedValidationResponse(w, r, v.Errors)
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"admin": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// changeAdminPasswordHandler allows admin to change their password
func (app *application) changeAdminPasswordHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	var input struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Verify current password
	match, err := user.Password.Matches(input.CurrentPassword)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	if !match {
		app.invalidCredentialsResponse(w, r)
		return
	}

	// Validate new password
	v := validator.New()
	data.ValidatePasswordPlaintext(v, input.NewPassword)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Set new password
	err = user.Password.Set(input.NewPassword)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

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

	err = app.writeJSON(w, http.StatusOK, envelope{"message": "password successfully updated"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// =============================================================================
// ADMIN USER MANAGEMENT
// =============================================================================

// listAllUsersHandler returns all users with pagination
func (app *application) listAllUsersHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Role   string
		Search string
		data.Filters
	}

	v := validator.New()
	qs := r.URL.Query()

	input.Role = app.readString(qs, "role", "")
	input.Search = app.readString(qs, "search", "")
	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)
	input.Filters.Sort = app.readString(qs, "sort", "id")
	input.Filters.SortSafelist = []string{"id", "name", "email", "created_at", "-id", "-name", "-email", "-created_at"}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	users, metadata, err := app.models.Users.GetAll(input.Role, input.Search, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"users": users, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// viewUserHandler retrieves a specific user by ID
func (app *application) viewUserHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	user, err := app.models.Users.GetByID(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrUserNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// updateUserHandler allows admin to update any user
func (app *application) updateUserHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	user, err := app.models.Users.GetByID(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrUserNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	var input struct {
		Name      *string `json:"name"`
		Email     *string `json:"email"`
		Activated *bool   `json:"activated"`
		Role      *string `json:"role"`
	}

	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if input.Name != nil {
		user.Name = *input.Name
	}
	if input.Email != nil {
		user.Email = *input.Email
	}
	if input.Activated != nil {
		user.Activated = *input.Activated
	}
	if input.Role != nil {
		oldRole := user.Role
		user.Role = *input.Role

		// Update permissions if role changed
		if oldRole != user.Role {
			err = app.models.Permissions.RemoveAllForUser(user.ID)
			if err != nil {
				app.serverErrorResponse(w, r, err)
				return
			}
			err = app.grandRolePermissions(user.ID, Role(user.Role))
			if err != nil {
				app.serverErrorResponse(w, r, err)
				return
			}
		}
	}

	v := validator.New()
	if data.ValidateUser(v, user); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Users.Update(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddError("email", "a user with this email address already exists")
			app.failedValidationResponse(w, r, v.Errors)
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// deleteUserHandler allows admin to delete any user
func (app *application) deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	err = app.models.Users.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrUserNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"message": "user successfully deleted"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// =============================================================================
// ADMIN AGENT MANAGEMENT
// =============================================================================

// listAllAgentsHandler returns all agents with filtering
func (app *application) listAllAgentsHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Status string
		Search string
		data.Filters
	}

	v := validator.New()
	qs := r.URL.Query()

	input.Status = app.readString(qs, "status", "")
	input.Search = app.readString(qs, "search", "")
	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)
	input.Filters.Sort = app.readString(qs, "sort", "id")
	input.Filters.SortSafelist = []string{"id", "name", "created_at", "-id", "-name", "-created_at"}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	agents, metadata, err := app.models.Agents.GetAll(input.Status, input.Search, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"agents": agents, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// approveAgentVerificationHandler verifies an agent
func (app *application) approveAgentVerificationHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	err = app.models.Agents.ApproveVerification(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrUserNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	agent, err := app.models.Users.GetByID(id)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"agent": agent}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// suspendAgentHandler suspends an agent account
func (app *application) suspendAgentHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	err = app.models.Agents.Suspend(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrUserNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"message": "agent successfully suspended"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// activateAgentHandler reactivates a suspended agent
func (app *application) activateAgentHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	err = app.models.Agents.Activate(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrUserNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"message": "agent successfully activated"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// =============================================================================
// ADMIN PROPERTY MANAGEMENT
// =============================================================================

// listAllPropertiesHandler returns all properties (admin override)
func (app *application) listAllPropertiesHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		AgentID      int64
		Status       string
		PropertyType string
		data.Filters
	}

	v := validator.New()
	qs := r.URL.Query()

	input.AgentID = int64(app.readInt(qs, "agent_id", 0, v))
	input.Status = app.readString(qs, "status", "")
	input.PropertyType = app.readString(qs, "property_type", "")
	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)
	input.Filters.Sort = app.readString(qs, "sort", "-id")
	input.Filters.SortSafelist = []string{"id", "title", "price", "created_at", "-id", "-title", "-price", "-created_at"}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	properties, metadata, err := app.models.Properties.GetAllAdmin(input.AgentID, input.Status, input.PropertyType, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"properties": properties, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// adminDeletePropertyHandler allows admin to delete any property
func (app *application) adminDeletePropertyHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	err = app.models.Properties.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrPropertyNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"message": "property successfully deleted"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// =============================================================================
// ADMIN PLATFORM STATISTICS
// =============================================================================

// getPlatformStatsHandler returns comprehensive platform statistics
func (app *application) getPlatformStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats, err := app.models.Admin.GetPlatformStats()
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"stats": stats}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// getGrowthMetricsHandler returns growth metrics over time
func (app *application) getGrowthMetricsHandler(w http.ResponseWriter, r *http.Request) {
	qs := r.URL.Query()
	period := app.readString(qs, "period", "30d") // 7d, 30d, 90d, 1y

	metrics, err := app.models.Admin.GetGrowthMetrics(period)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"metrics": metrics}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
