package main

import (
	"errors"
	"net/http"

	"github.com/codercollo/property/backend/internal/data"
	"github.com/codercollo/property/backend/internal/validator"
)

// createReviewHandler handles submitting a new review for a property
func (app *application) createReviewHandler(w http.ResponseWriter, r *http.Request) {
	//Get the property ID from the URL
	propertyID, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	//Verify property exists
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

	//Parse input
	var input struct {
		Rating  int32  `json:"rating"`
		Comment string `json:"comment"`
	}

	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	//Get authenticated user form contex
	user := app.contextGetUser(r)

	//create review struct
	review := &data.Review{
		PropertyID: propertyID,
		UserID:     user.ID,
		Rating:     input.Rating,
		Comment:    input.Comment,
		Status:     "pending", //default
	}

	//validate review
	v := validator.New()

	if data.ValidateReview(v, review); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	//Insert review into the database
	err = app.models.Reviews.Insert(review)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	//Fetch the complete review with user name
	review, err = app.models.Reviews.Get(review.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	//Return response
	err = app.writeJSON(w, http.StatusCreated, envelope{"review": review}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// listReviewsForPropertyHandler returns all approved reviews for a specific property
func (app *application) listReviewsForPropertyHandler(w http.ResponseWriter, r *http.Request) {
	// Get property ID from URL
	propertyID, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Parse query parameters
	var input struct {
		data.Filters
	}

	v := validator.New()
	qs := r.URL.Query()

	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)
	input.Filters.Sort = app.readString(qs, "sort", "-created_at")
	input.Filters.SortSafelist = []string{"created_at", "-created_at", "rating", "-rating"}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Fetch reviews
	reviews, metadata, err := app.models.Reviews.GetAllForProperty(propertyID, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Get average rating
	avgRating, totalReviews, err := app.models.Reviews.GetAverageRating(propertyID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Return response with reviews, metadata, and rating stats
	err = app.writeJSON(w, http.StatusOK, envelope{
		"reviews":        reviews,
		"metadata":       metadata,
		"average_rating": avgRating,
		"total_reviews":  totalReviews,
	}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// listPendingReviewsHandler returns all pending reviews (admin only)
func (app *application) listPendingReviewsHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	var input struct {
		data.Filters
	}

	v := validator.New()
	qs := r.URL.Query()

	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)
	input.Filters.Sort = app.readString(qs, "sort", "-created_at")
	input.Filters.SortSafelist = []string{"created_at", "-created_at"}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Fetch pending reviews
	reviews, metadata, err := app.models.Reviews.GetAllPending(input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Return response
	err = app.writeJSON(w, http.StatusOK, envelope{
		"reviews":  reviews,
		"metadata": metadata,
	}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// approveReviewHandler approves a pending review (admin only)
func (app *application) approveReviewHandler(w http.ResponseWriter, r *http.Request) {
	// Get review ID from URL
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Get authenticated user (admin)
	user := app.contextGetUser(r)

	// Approve the review
	err = app.models.Reviews.Approve(id, user.ID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrReviewNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Fetch updated review
	review, err := app.models.Reviews.Get(id)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Return response
	err = app.writeJSON(w, http.StatusOK, envelope{"review": review}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// rejectReviewHandler rejects a pending review (admin only)
func (app *application) rejectReviewHandler(w http.ResponseWriter, r *http.Request) {
	// Get review ID from URL
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Get authenticated user (admin)
	user := app.contextGetUser(r)

	// Reject the review
	err = app.models.Reviews.Reject(id, user.ID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrReviewNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Fetch updated review
	review, err := app.models.Reviews.Get(id)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Return response
	err = app.writeJSON(w, http.StatusOK, envelope{"review": review}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// deleteReviewHandler deletes a review (admin or review owner)
func (app *application) deleteReviewHandler(w http.ResponseWriter, r *http.Request) {
	// Get review ID from URL
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Get authenticated user
	user := app.contextGetUser(r)

	// Fetch review to check ownership
	review, err := app.models.Reviews.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrReviewNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Check if user is owner or admin
	if review.UserID != user.ID && user.Role != "admin" {
		app.notPermittedResponse(w, r)
		return
	}

	// Delete the review
	err = app.models.Reviews.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrReviewNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Return success response
	err = app.writeJSON(w, http.StatusOK, envelope{"message": "review successfully deleted"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
