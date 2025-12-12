package main

import (
	"errors"
	"net/http"

	"github.com/codercollo/property/backend/internal/data"
	"github.com/codercollo/property/backend/internal/validator"
)

// =============================================================================
// USER FAVOURITES MANAGEMENT
// =============================================================================

// addFavouriteHandler allows users to add a property to their favourites
func (app *application) addFavouriteHandler(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user
	user := app.contextGetUser(r)

	// Get property ID from URL
	propertyID, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Verify property exists
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

	// Add to favourites
	err = app.models.Favourites.Add(user.ID, propertyID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrFavouriteAlreadyExists):
			app.errorResponse(w, r, http.StatusConflict, "property already in favourites")
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Return success response with property details
	err = app.writeJSON(w, http.StatusCreated, envelope{
		"message":  "property added to favourites",
		"property": property,
	}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// removeFavouriteHandler allows users to remove a property from their favourites
func (app *application) removeFavouriteHandler(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user
	user := app.contextGetUser(r)

	// Get property ID from URL
	propertyID, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Remove from favourites
	err = app.models.Favourites.Remove(user.ID, propertyID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrFavouriteNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Return success response
	err = app.writeJSON(w, http.StatusOK, envelope{
		"message": "property removed from favourites",
	}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// listUserFavouritesHandler retrieves all favourited properties for the authenticated user
func (app *application) listUserFavouritesHandler(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user
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
		"id", "title", "price", "created_at",
		"-id", "-title", "-price", "-created_at",
	}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Fetch favourites
	favourites, metadata, err := app.models.Favourites.GetAllForUser(user.ID, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Return response
	err = app.writeJSON(w, http.StatusOK, envelope{
		"favourites": favourites,
		"metadata":   metadata,
	}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// getUserFavouriteStatsHandler returns statistics about user's favourites
func (app *application) getUserFavouriteStatsHandler(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user
	user := app.contextGetUser(r)

	// Get stats
	stats, err := app.models.Favourites.GetStatsForUser(user.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Return response
	err = app.writeJSON(w, http.StatusOK, envelope{"stats": stats}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// checkFavouriteStatusHandler checks if a property is favourited by the user
func (app *application) checkFavouriteStatusHandler(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user
	user := app.contextGetUser(r)

	// Get property ID from URL
	propertyID, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Check if favourited
	isFavourited, err := app.models.Favourites.IsFavourite(user.ID, propertyID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Return response
	err = app.writeJSON(w, http.StatusOK, envelope{
		"property_id":   propertyID,
		"is_favourited": isFavourited,
	}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// =============================================================================
// PUBLIC ENDPOINTS (Optional - for showing popular properties)
// =============================================================================

// listMostFavouritedPropertiesHandler returns the most favourited properties
func (app *application) listMostFavouritedPropertiesHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	var input struct {
		data.Filters
	}

	v := validator.New()
	qs := r.URL.Query()

	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Fetch most favourited properties
	properties, metadata, err := app.models.Favourites.GetMostFavouritedProperties(input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Return response
	err = app.writeJSON(w, http.StatusOK, envelope{
		"properties": properties,
		"metadata":   metadata,
	}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// getPropertyFavouriteCountHandler returns the number of users who favourited a property
func (app *application) getPropertyFavouriteCountHandler(w http.ResponseWriter, r *http.Request) {
	// Get property ID from URL
	propertyID, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Verify property exists
	_, err = app.models.Properties.Get(propertyID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrPropertyNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Get favourite count
	count, err := app.models.Favourites.GetFavouriteCount(propertyID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Return response
	err = app.writeJSON(w, http.StatusOK, envelope{
		"property_id":     propertyID,
		"favourite_count": count,
	}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
