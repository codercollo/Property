package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/codercollo/property/backend/internal/data"
	"github.com/codercollo/property/backend/internal/validator"
)

// uploadProfilePhotoHandler handles profile photo uploads for authenticated users
func (app *application) uploadProfilePhotoHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	// Parse multipart form (max 10MB in memory)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Get the file from form data
	file, header, err := r.FormFile("photo")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			v := validator.New()
			v.AddError("photo", "profile photo is required")
			app.failedValidationResponse(w, r, v.Errors)
			return
		}
		app.badRequestResponse(w, r, err)
		return
	}
	defer file.Close()

	// Create upload struct
	upload := &data.ProfilePhotoUpload{
		File:   file,
		Header: header,
		UserID: user.ID,
	}

	// Validate upload
	v := validator.New()
	data.ValidateProfilePhotoUpload(v, upload)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Delete old profile photo if exists
	if user.ProfilePhoto != "" {
		if err := data.DeleteProfilePhoto(user.ProfilePhoto); err != nil {
			app.logger.PrintError(err, map[string]string{
				"user_id": fmt.Sprintf("%d", user.ID),
				"action":  "delete_old_photo",
			})
			// Continue even if deletion fails
		}
	}

	// Save new photo to disk
	photoURL, err := data.SaveProfilePhoto(file, header, user.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Update database with new photo URL
	if err := app.models.Users.UpdateProfilePhoto(user.ID, photoURL); err != nil {
		// Clean up uploaded file on database error
		data.DeleteProfilePhoto(photoURL)

		switch {
		case errors.Is(err, data.ErrUserNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Update user object for response
	user.ProfilePhoto = photoURL

	// Return success response
	err = app.writeJSON(w, http.StatusOK, envelope{
		"message":       "profile photo uploaded successfully",
		"profile_photo": photoURL,
		"user":          user,
	}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// getProfilePhotoHandler retrieves the user's current profile photo URL
func (app *application) getProfilePhotoHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	photoURL, err := app.models.Users.GetProfilePhoto(user.ID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrUserNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{
		"profile_photo": photoURL,
	}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// deleteProfilePhotoHandler removes the user's profile photo
func (app *application) deleteProfilePhotoHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	// Get current photo URL
	photoURL, err := app.models.Users.GetProfilePhoto(user.ID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrUserNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Delete photo from disk
	if photoURL != "" {
		if err := data.DeleteProfilePhoto(photoURL); err != nil {
			app.logger.PrintError(err, map[string]string{
				"user_id": fmt.Sprintf("%d", user.ID),
				"action":  "delete_photo",
			})
			// Continue even if deletion fails
		}
	}

	// Remove photo URL from database
	if err := app.models.Users.DeleteUserProfilePhoto(user.ID); err != nil {
		switch {
		case errors.Is(err, data.ErrUserNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{
		"message": "profile photo deleted successfully",
	}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
