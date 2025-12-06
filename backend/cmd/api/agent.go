package main

import (
	"errors"
	"net/http"

	"github.com/codercollo/property/backend/internal/data"
	"github.com/codercollo/property/backend/internal/validator"
)

// =============================================================================
// A. AGENT ACCOUNT MANAGEMENT
// =============================================================================

// getAgentProfileHandler returns the authenticated agent's profile
func (app *application) getAgentProfileHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	if user.Role != "agent" {
		app.notPermittedResponse(w, r)
		return
	}

	err := app.writeJSON(w, http.StatusOK, envelope{"agent": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// updateAgentProfileHandler updates the authenticated agent's profile
func (app *application) updateAgentProfileHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	if user.Role != "agent" {
		app.notPermittedResponse(w, r)
		return
	}

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
	if input.Name != nil {
		v.Check(*input.Name != "", "name", "must be provided")
		v.Check(len(*input.Name) <= 500, "name", "must not be more than 500 bytes long")
	}
	if input.Email != nil {
		data.ValidateEmail(v, *input.Email)
	}

	if !v.Valid() {
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

	err = app.writeJSON(w, http.StatusOK, envelope{"agent": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// deleteAgentAccountHandler deactivates the authenticated agent's account
func (app *application) deleteAgentAccountHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	if user.Role != "agent" {
		app.notPermittedResponse(w, r)
		return
	}

	user.Activated = false
	err := app.models.Users.Update(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"message": "account successfully deactivated"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// changeAgentPasswordHandler changes the authenticated agent's password
func (app *application) changeAgentPasswordHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	if user.Role != "agent" {
		app.notPermittedResponse(w, r)
		return
	}

	var input struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	match, err := user.Password.Matches(input.CurrentPassword)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	if !match {
		app.invalidCredentialsResponse(w, r)
		return
	}

	v := validator.New()
	data.ValidatePasswordPlaintext(v, input.NewPassword)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

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

	err = app.writeJSON(w, http.StatusOK, envelope{"message": "password successfully changed"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// =============================================================================
// B. AGENT PROPERTY MANAGEMENT
// =============================================================================

// listAgentPropertiesHandler lists all properties belonging to the authenticated agent
func (app *application) listAgentPropertiesHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	if user.Role != "agent" {
		app.notPermittedResponse(w, r)
		return
	}

	var input struct {
		data.Filters
	}

	v := validator.New()
	qs := r.URL.Query()

	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)
	input.Filters.Sort = app.readString(qs, "sort", "id")
	input.Filters.SortSafelist = []string{"id", "title", "year_built", "price", "-id", "-title", "-year_built", "-price"}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	properties, metadata, err := app.models.Properties.GetAllForAgent(user.ID, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"properties": properties, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// getAgentPropertyHandler retrieves a specific property belonging to the authenticated agent
func (app *application) getAgentPropertyHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	if user.Role != "agent" {
		app.notPermittedResponse(w, r)
		return
	}

	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	property, err := app.models.Properties.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrPropertyNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if property.AgentID != user.ID {
		app.notPermittedResponse(w, r)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"property": property}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// getAgentPropertyStatsHandler returns statistics about the agent's properties
func (app *application) getAgentPropertyStatsHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	if user.Role != "agent" {
		app.notPermittedResponse(w, r)
		return
	}

	stats, err := app.models.Properties.GetStatsForAgent(user.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"stats": stats}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// =============================================================================
// C. AGENT REVIEW MANAGEMENT
// =============================================================================

// listAgentReviewsHandler lists all reviews for the agent's properties
func (app *application) listAgentReviewsHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	if user.Role != "agent" {
		app.notPermittedResponse(w, r)
		return
	}

	var input struct {
		data.Filters
	}

	v := validator.New()
	qs := r.URL.Query()

	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)
	input.Filters.Sort = app.readString(qs, "sort", "-created_at")
	input.Filters.SortSafelist = []string{"id", "rating", "created_at", "-id", "-rating", "-created_at"}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	reviews, metadata, err := app.models.Reviews.GetAllForAgent(user.ID, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"reviews": reviews, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// listAgentPendingReviewsHandler lists pending reviews for the agent's properties
func (app *application) listAgentPendingReviewsHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	if user.Role != "agent" {
		app.notPermittedResponse(w, r)
		return
	}

	var input struct {
		data.Filters
	}

	v := validator.New()
	qs := r.URL.Query()

	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)
	input.Filters.Sort = app.readString(qs, "sort", "-created_at")
	input.Filters.SortSafelist = []string{"id", "created_at", "-id", "-created_at"}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	reviews, metadata, err := app.models.Reviews.GetPendingForAgent(user.ID, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"reviews": reviews, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// =============================================================================
// D. AGENT PAYMENTS & FEATURED LISTINGS
// =============================================================================

// createFeaturePaymentHandler processes payment to feature a property
func (app *application) createFeaturePaymentHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	if user.Role != "agent" {
		app.notPermittedResponse(w, r)
		return
	}

	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	property, err := app.models.Properties.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrPropertyNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if property.AgentID != user.ID {
		app.notPermittedResponse(w, r)
		return
	}

	var input struct {
		PaymentMethod string  `json:"payment_method"`
		Amount        float64 `json:"amount"`
	}

	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	v.Check(input.PaymentMethod != "", "payment_method", "must be provided")
	v.Check(input.Amount > 0, "amount", "must be greater than zero")

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	payment, err := app.models.Payments.Create(user.ID, property.ID, input.Amount, input.PaymentMethod)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.models.Properties.Feature(id)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusCreated, envelope{"payment": payment}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// listPaymentHistoryHandler lists the agent's payment history
func (app *application) listPaymentHistoryHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	if user.Role != "agent" {
		app.notPermittedResponse(w, r)
		return
	}

	var input struct {
		data.Filters
	}

	v := validator.New()
	qs := r.URL.Query()

	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)
	input.Filters.Sort = app.readString(qs, "sort", "-created_at")
	input.Filters.SortSafelist = []string{"id", "amount", "created_at", "-id", "-amount", "-created_at"}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	payments, metadata, err := app.models.Payments.GetAllForAgent(user.ID, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"payments": payments, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// getPaymentStatusHandler retrieves the status of a specific payment
func (app *application) getPaymentStatusHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	if user.Role != "agent" {
		app.notPermittedResponse(w, r)
		return
	}

	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	payment, err := app.models.Payments.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrPaymentNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if payment.AgentID != user.ID {
		app.notPermittedResponse(w, r)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"payment": payment}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// =============================================================================
// E. AGENT DASHBOARD
// =============================================================================

// getAgentDashboardStatsHandler returns comprehensive dashboard metrics for the agent
func (app *application) getAgentDashboardStatsHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	if user.Role != "agent" {
		app.notPermittedResponse(w, r)
		return
	}

	stats, err := app.models.Agents.GetDashboardStats(user.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"stats": stats}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
