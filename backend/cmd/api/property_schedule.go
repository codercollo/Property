package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/codercollo/property/backend/internal/data"
	"github.com/codercollo/property/backend/internal/validator"
)

// =============================================================================
// USER SCHEDULE ENDPOINTS
// =============================================================================

// createScheduleHandler allows users to schedule a property viewing
func (app *application) createScheduleHandler(w http.ResponseWriter, r *http.Request) {
	// Get property ID from URL
	propertyID, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Get authenticated user
	user := app.contextGetUser(r)

	// Get the property to extract agent_id
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
		app.badRequestResponse(w, r, errors.New("property does not have an assigned agent"))
		return
	}

	// Parse request body
	var input struct {
		ScheduledAt     time.Time `json:"scheduled_at"`
		DurationMinutes int       `json:"duration_minutes"`
		Notes           string    `json:"notes"`
	}

	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Set default duration if not provided
	if input.DurationMinutes == 0 {
		input.DurationMinutes = 60 // Default 1 hour
	}

	// Create schedule
	schedule := &data.Schedule{
		PropertyID:      propertyID,
		UserID:          user.ID,
		AgentID:         property.AgentID.Int64,
		ScheduledAt:     input.ScheduledAt,
		DurationMinutes: input.DurationMinutes,
		Status:          "pending",
		Notes:           input.Notes,
	}

	// Validate
	v := validator.New()
	if data.ValidateSchedule(v, schedule); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Insert schedule
	err = app.models.Schedules.Insert(schedule)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrScheduleConflict):
			v.AddError("scheduled_at", "this time slot is already booked")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Send notification to agent (background task)
	app.background(func() {
		// Here you could send an email or push notification to the agent
		app.logger.PrintInfo("Schedule created", map[string]string{
			"schedule_id": strconv.FormatInt(schedule.ID, 10),
			"property_id": strconv.FormatInt(propertyID, 10),
			"agent_id":    strconv.FormatInt(property.AgentID.Int64, 10),
		})
	})

	// Return created schedule
	err = app.writeJSON(w, http.StatusCreated, envelope{"schedule": schedule}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// listUserSchedulesHandler lists all schedules for the authenticated user
func (app *application) listUserSchedulesHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	var input struct {
		data.Filters
	}

	v := validator.New()
	qs := r.URL.Query()

	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)
	input.Filters.Sort = app.readString(qs, "sort", "-scheduled_at")
	input.Filters.SortSafelist = []string{"id", "scheduled_at", "-id", "-scheduled_at"}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	schedules, metadata, err := app.models.Schedules.GetAllForUser(user.ID, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"schedules": schedules, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// getUserScheduleHandler retrieves a specific schedule for the user
func (app *application) getUserScheduleHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	schedule, err := app.models.Schedules.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrScheduleNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Verify ownership
	if schedule.UserID != user.ID {
		app.notPermittedResponse(w, r)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"schedule": schedule}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// cancelUserScheduleHandler allows users to cancel their schedules
func (app *application) cancelUserScheduleHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	schedule, err := app.models.Schedules.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrScheduleNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Verify ownership
	if schedule.UserID != user.ID {
		app.notPermittedResponse(w, r)
		return
	}

	// Check if already cancelled or completed
	if schedule.Status == "cancelled" || schedule.Status == "completed" {
		app.badRequestResponse(w, r, errors.New("schedule cannot be cancelled"))
		return
	}

	// Update status to cancelled
	err = app.models.Schedules.UpdateStatus(id, "cancelled", schedule.Version)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"message": "schedule successfully cancelled"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// =============================================================================
// AGENT SCHEDULE ENDPOINTS
// =============================================================================

// listAgentSchedulesHandler lists all schedules for the agent
func (app *application) listAgentSchedulesHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	// Check if user is an agent (removed requireAgentRole middleware)
	if user.Role != "agent" {
		app.notPermittedResponse(w, r)
		return
	}

	var input struct {
		Status string
		data.Filters
	}

	v := validator.New()
	qs := r.URL.Query()

	input.Status = app.readString(qs, "status", "")
	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)
	input.Filters.Sort = app.readString(qs, "sort", "scheduled_at")
	input.Filters.SortSafelist = []string{"id", "scheduled_at", "-id", "-scheduled_at"}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	schedules, metadata, err := app.models.Schedules.GetAllForAgent(user.ID, input.Status, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"schedules": schedules, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// getAgentScheduleHandler retrieves a specific schedule for the agent
func (app *application) getAgentScheduleHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	// Check if user is an agent
	if user.Role != "agent" {
		app.notPermittedResponse(w, r)
		return
	}

	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	schedule, err := app.models.Schedules.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrScheduleNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Verify this is the agent's schedule
	if schedule.AgentID != user.ID {
		app.notPermittedResponse(w, r)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"schedule": schedule}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// updateAgentScheduleStatusHandler allows agents to update schedule status
func (app *application) updateAgentScheduleStatusHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	// Check if user is an agent
	if user.Role != "agent" {
		app.notPermittedResponse(w, r)
		return
	}

	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	schedule, err := app.models.Schedules.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrScheduleNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Verify this is the agent's schedule
	if schedule.AgentID != user.ID {
		app.notPermittedResponse(w, r)
		return
	}

	var input struct {
		Status string `json:"status"`
	}

	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Validate status
	v := validator.New()
	validStatuses := []string{"confirmed", "cancelled", "completed"}
	v.Check(validator.In(input.Status, validStatuses...), "status", "must be confirmed, cancelled, or completed")

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Update status
	err = app.models.Schedules.UpdateStatus(id, input.Status, schedule.Version)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Fetch updated schedule
	updatedSchedule, err := app.models.Schedules.Get(id)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"schedule": updatedSchedule}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// getAgentScheduleStatsHandler returns schedule statistics for the agent
func (app *application) getAgentScheduleStatsHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	// Check if user is an agent
	if user.Role != "agent" {
		app.notPermittedResponse(w, r)
		return
	}

	stats, err := app.models.Schedules.GetStatsForAgent(user.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"stats": stats}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// rescheduleUserScheduleHandler allows users to reschedule their viewing appointments
func (app *application) rescheduleUserScheduleHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	// Get schedule ID from URL
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Fetch the schedule
	schedule, err := app.models.Schedules.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrScheduleNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Verify ownership
	if schedule.UserID != user.ID {
		app.notPermittedResponse(w, r)
		return
	}

	// Parse request body
	var input struct {
		ScheduledAt     time.Time `json:"scheduled_at"`
		DurationMinutes *int      `json:"duration_minutes,omitempty"`
		Notes           *string   `json:"notes,omitempty"`
	}

	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Use existing duration if not provided
	newDuration := schedule.DurationMinutes
	if input.DurationMinutes != nil {
		newDuration = *input.DurationMinutes
	}

	// Validate reschedule request
	v := validator.New()
	data.ValidateReschedule(v, schedule, input.ScheduledAt, newDuration)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Perform the reschedule
	err = app.models.Schedules.Reschedule(id, input.ScheduledAt, newDuration, schedule.Version)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrScheduleConflict):
			v.AddError("scheduled_at", "this time slot is already booked")
			app.failedValidationResponse(w, r, v.Errors)
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Update notes if provided
	if input.Notes != nil {
		updatedSchedule, err := app.models.Schedules.Get(id)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}
		updatedSchedule.Notes = *input.Notes
		// Note: You might want to add an UpdateNotes method to avoid version conflict issues
	}

	// Fetch the updated schedule
	updatedSchedule, err := app.models.Schedules.Get(id)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Get agent information for notification
	agent, err := app.models.Users.GetByID(schedule.AgentID)
	if err != nil {
		app.logger.PrintError(err, map[string]string{
			"context": "fetching agent for reschedule notification",
		})
	} else {
		// Send notification to agent (background task)
		app.background(func() {
			property, err := app.models.Properties.Get(schedule.PropertyID)
			if err != nil {
				app.logger.PrintError(err, nil)
				return
			}

			emailData := map[string]interface{}{
				"agentName":       agent.Name,
				"userName":        user.Name,
				"userEmail":       user.Email,
				"propertyTitle":   property.Title,
				"oldScheduledAt":  schedule.ScheduledAt.Format("Monday, January 2, 2006 at 3:04 PM"),
				"newScheduledAt":  input.ScheduledAt.Format("Monday, January 2, 2006 at 3:04 PM"),
				"duration":        newDuration,
				"rescheduleCount": updatedSchedule.RescheduleCount,
				"scheduleID":      id,
			}

			err = app.mailer.Send(agent.Email, "schedule_rescheduled.tmpl", emailData)
			if err != nil {
				app.logger.PrintError(err, map[string]string{
					"context": "sending reschedule notification email",
				})
			}
		})
	}

	// Log the reschedule action
	app.logger.PrintInfo("schedule rescheduled", map[string]string{
		"schedule_id":      fmt.Sprintf("%d", id),
		"user_id":          fmt.Sprintf("%d", user.ID),
		"agent_id":         fmt.Sprintf("%d", schedule.AgentID),
		"old_time":         schedule.ScheduledAt.Format(time.RFC3339),
		"new_time":         input.ScheduledAt.Format(time.RFC3339),
		"reschedule_count": fmt.Sprintf("%d", updatedSchedule.RescheduleCount),
	})

	// Return success response with updated schedule
	err = app.writeJSON(w, http.StatusOK, envelope{
		"message":  "schedule successfully rescheduled",
		"schedule": updatedSchedule,
	}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
