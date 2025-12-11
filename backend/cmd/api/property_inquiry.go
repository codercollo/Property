package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/codercollo/property/backend/internal/data"
	"github.com/codercollo/property/backend/internal/validator"
)

// =============================================================================
// USER: CREATE INQUIRY
// =============================================================================

// createInquiryHandler allows users to submit an inquiry about a property
func (app *application) createInquiryHandler(w http.ResponseWriter, r *http.Request) {
	// Get property ID from URL
	propertyID, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Verify property exists and get agent ID
	property, err := app.models.Properties.Get(propertyID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrPropertyNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Check if property has an agent
	if !property.AgentID.Valid {
		app.errorResponse(w, r, http.StatusBadRequest,
			"this property does not have an assigned agent")
		return
	}

	// Get authenticated user
	user := app.contextGetUser(r)

	// Parse input
	var input struct {
		Name                   string     `json:"name"`
		Email                  string     `json:"email"`
		Phone                  string     `json:"phone"`
		Message                string     `json:"message"`
		InquiryType            string     `json:"inquiry_type"`
		PreferredContactMethod string     `json:"preferred_contact_method"`
		PreferredViewingDate   *time.Time `json:"preferred_viewing_date"`
	}

	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Create inquiry
	inquiry := &data.Inquiry{
		PropertyID:             propertyID,
		UserID:                 user.ID,
		AgentID:                property.AgentID.Int64,
		Name:                   input.Name,
		Email:                  input.Email,
		Phone:                  input.Phone,
		Message:                input.Message,
		InquiryType:            input.InquiryType,
		PreferredContactMethod: input.PreferredContactMethod,
		PreferredViewingDate:   input.PreferredViewingDate,
		Status:                 "new",
		Priority:               "normal",
	}

	// Auto-escalate priority for viewing requests
	if inquiry.InquiryType == "viewing" || inquiry.InquiryType == "purchase" {
		inquiry.Priority = "high"
	}

	// Validate inquiry
	v := validator.New()
	if data.ValidateInquiry(v, inquiry); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Insert into database
	err = app.models.Inquiries.Insert(inquiry)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Fetch complete inquiry with joined data
	inquiry, err = app.models.Inquiries.Get(inquiry.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Send notification email to agent (async)
	app.background(func() {
		agent, err := app.models.Users.GetByID(property.AgentID.Int64)
		if err != nil {
			app.logger.PrintError(err, nil)
			return
		}

		data := map[string]interface{}{
			"agentName":     agent.Name,
			"inquirerName":  inquiry.Name,
			"propertyTitle": property.Title,
			"inquiryType":   inquiry.InquiryType,
			"message":       inquiry.Message,
			"inquiryID":     inquiry.ID,
		}

		err = app.mailer.Send(agent.Email, "inquiry_notification.tmpl", data)
		if err != nil {
			app.logger.PrintError(err, nil)
		}
	})

	// Return created inquiry
	err = app.writeJSON(w, http.StatusCreated, envelope{"inquiry": inquiry}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// =============================================================================
// AGENT: VIEW AND MANAGE INQUIRIES
// =============================================================================

// listAgentInquiriesHandler retrieves all inquiries for the authenticated agent
func (app *application) listAgentInquiriesHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	if user.Role != "agent" {
		app.notPermittedResponse(w, r)
		return
	}

	// Parse query parameters
	var input struct {
		Status string
		data.Filters
	}

	v := validator.New()
	qs := r.URL.Query()

	input.Status = app.readString(qs, "status", "")
	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)
	input.Filters.Sort = app.readString(qs, "sort", "-created_at")
	input.Filters.SortSafelist = []string{
		"id", "created_at", "priority", "status",
		"-id", "-created_at", "-priority", "-status",
	}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Fetch inquiries
	inquiries, metadata, err := app.models.Inquiries.GetAllForAgent(
		user.ID,
		input.Status,
		input.Filters,
	)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Return response
	err = app.writeJSON(w, http.StatusOK, envelope{
		"inquiries": inquiries,
		"metadata":  metadata,
	}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// getAgentInquiryHandler retrieves a specific inquiry for the agent
func (app *application) getAgentInquiryHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	if user.Role != "agent" {
		app.notPermittedResponse(w, r)
		return
	}

	// Get inquiry ID from URL
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Fetch inquiry
	inquiry, err := app.models.Inquiries.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrPropertyNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Verify inquiry belongs to this agent
	if inquiry.AgentID != user.ID {
		app.notPermittedResponse(w, r)
		return
	}

	// Return inquiry
	err = app.writeJSON(w, http.StatusOK, envelope{"inquiry": inquiry}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// updateInquiryHandler allows agents to update inquiry status and notes
func (app *application) updateInquiryHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	if user.Role != "agent" {
		app.notPermittedResponse(w, r)
		return
	}

	// Get inquiry ID
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Fetch inquiry
	inquiry, err := app.models.Inquiries.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrPropertyNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Verify inquiry belongs to this agent
	if inquiry.AgentID != user.ID {
		app.notPermittedResponse(w, r)
		return
	}

	// Parse input
	var input struct {
		Status     *string `json:"status"`
		Priority   *string `json:"priority"`
		AgentNotes *string `json:"agent_notes"`
	}

	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Update fields if provided
	if input.Status != nil {
		inquiry.Status = *input.Status
		// Auto-set responded_at when status changes from 'new'
		if inquiry.RespondedAt == nil && *input.Status != "new" {
			now := time.Now()
			inquiry.RespondedAt = &now
		}
	}
	if input.Priority != nil {
		inquiry.Priority = *input.Priority
	}
	if input.AgentNotes != nil {
		inquiry.AgentNotes = *input.AgentNotes
	}

	// Validate
	v := validator.New()
	data.ValidateInquiry(v, inquiry)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Update in database
	err = app.models.Inquiries.Update(inquiry)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Fetch updated inquiry
	inquiry, err = app.models.Inquiries.Get(id)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Return updated inquiry
	err = app.writeJSON(w, http.StatusOK, envelope{"inquiry": inquiry}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// getAgentInquiryStatsHandler returns inquiry statistics for the agent
func (app *application) getAgentInquiryStatsHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	if user.Role != "agent" {
		app.notPermittedResponse(w, r)
		return
	}

	stats, err := app.models.Inquiries.GetStatsForAgent(user.ID)
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
// USER: VIEW OWN INQUIRIES
// =============================================================================

// listUserInquiriesHandler retrieves all inquiries made by the authenticated user
func (app *application) listUserInquiriesHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	// Parse query parameters
	var input struct {
		data.Filters
	}

	v := validator.New()
	qs := r.URL.Query()

	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)
	input.Filters.Sort = app.readString(qs, "sort", "-created_at")
	input.Filters.SortSafelist = []string{
		"id", "created_at", "-id", "-created_at",
	}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Fetch inquiries
	inquiries, metadata, err := app.models.Inquiries.GetAllForUser(user.ID, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Return response
	err = app.writeJSON(w, http.StatusOK, envelope{
		"inquiries": inquiries,
		"metadata":  metadata,
	}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// getUserInquiryHandler retrieves a specific inquiry made by the user
func (app *application) getUserInquiryHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	// Get inquiry ID from URL
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Fetch inquiry
	inquiry, err := app.models.Inquiries.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrPropertyNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Verify inquiry belongs to this user (or user is admin)
	if inquiry.UserID != user.ID && user.Role != "admin" {
		app.notPermittedResponse(w, r)
		return
	}

	// Return inquiry
	err = app.writeJSON(w, http.StatusOK, envelope{"inquiry": inquiry}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// deleteInquiryHandler allows users or admins to delete an inquiry
func (app *application) deleteInquiryHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	// Get inquiry ID
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Fetch inquiry
	inquiry, err := app.models.Inquiries.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrPropertyNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Check permissions: user can delete their own, admin can delete any
	if inquiry.UserID != user.ID && user.Role != "admin" {
		app.notPermittedResponse(w, r)
		return
	}

	// Delete inquiry
	err = app.models.Inquiries.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrPropertyNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Return success
	err = app.writeJSON(w, http.StatusOK, envelope{"message": "inquiry successfully deleted"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
