package main

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/codercollo/property/backend/internal/data"
	"github.com/codercollo/property/backend/internal/validator"
)

// uploadPropertyMediaHandler handles media file uploads for a property
func (app *application) uploadPropertyMediaHandler(w http.ResponseWriter, r *http.Request) {
	// Get property ID from URL
	propertyID, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Verify property exists and user has permission
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

	// Get authenticated user
	user := app.contextGetUser(r)

	// Check if user is the property owner or admin
	if property.AgentID.Valid && property.AgentID.Int64 != user.ID && user.Role != "admin" {
		app.notPermittedResponse(w, r)
		return
	}

	// Parse multipart form (max 50MB)
	err = r.ParseMultipartForm(50 << 20)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Get form values
	mediaType := r.FormValue("media_type")
	caption := r.FormValue("caption")

	// Parse display_order correctly from multipart form
	displayOrder := 0
	if r.MultipartForm != nil {
		if vals, ok := r.MultipartForm.Value["display_order"]; ok && len(vals) > 0 {
			parsedOrder, err := strconv.Atoi(vals[0])
			if err != nil {
				app.badRequestResponse(w, r, fmt.Errorf("invalid display_order"))
				return
			}
			displayOrder = parsedOrder
		}
	}

	// Validate media type
	v := validator.New()
	v.Check(mediaType != "", "media_type", "must be provided")
	v.Check(validator.In(mediaType, "image", "video", "floor_plan"), "media_type", "must be one of: image, video, floor_plan")

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Get uploaded file
	file, header, err := r.FormFile("file")
	if err != nil {
		app.badRequestResponse(w, r, fmt.Errorf("file upload required"))
		return
	}
	defer file.Close()

	// Validate file
	app.validateMediaFile(header, mediaType, v)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Save file to disk
	filePath, err := app.saveMediaFile(file, header, propertyID, mediaType)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Create media record
	media := &data.PropertyMedia{
		PropertyID:   propertyID,
		MediaType:    mediaType,
		FilePath:     filePath,
		FileName:     header.Filename,
		FileSize:     header.Size,
		MimeType:     header.Header.Get("Content-Type"),
		DisplayOrder: displayOrder,
		Caption:      caption,
		IsPrimary:    false,
	}

	// Validate media record
	if data.ValidateMedia(v, media); !v.Valid() {
		// Clean up saved file if validation fails
		os.Remove(filePath)
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Insert into database
	err = app.models.Media.Insert(media)
	if err != nil {
		// Clean up saved file if database insert fails
		os.Remove(filePath)
		app.serverErrorResponse(w, r, err)
		return
	}

	// Return success response
	err = app.writeJSON(w, http.StatusCreated, envelope{"media": media}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// listPropertyMediaHandler retrieves all media for a property
func (app *application) listPropertyMediaHandler(w http.ResponseWriter, r *http.Request) {
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

	// Get all media for the property
	mediaList, err := app.models.Media.GetAllForProperty(propertyID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Return media list
	err = app.writeJSON(w, http.StatusOK, envelope{"media": mediaList}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// deletePropertyMediaHandler removes a media file
func (app *application) deletePropertyMediaHandler(w http.ResponseWriter, r *http.Request) {
	// Get property ID from URL
	propertyID, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Get media ID from query string
	mediaIDStr := r.URL.Query().Get("media_id")
	if mediaIDStr == "" {
		app.badRequestResponse(w, r, fmt.Errorf("media_id is required"))
		return
	}

	var mediaID int64
	_, err = fmt.Sscanf(mediaIDStr, "%d", &mediaID)
	if err != nil || mediaID < 1 {
		app.badRequestResponse(w, r, fmt.Errorf("invalid media_id"))
		return
	}

	// Get the media record
	media, err := app.models.Media.Get(mediaID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrPropertyNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Verify media belongs to the specified property
	if media.PropertyID != propertyID {
		app.notFoundResponse(w, r)
		return
	}

	// Get property
	property, err := app.models.Properties.Get(propertyID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Get authenticated user
	user := app.contextGetUser(r)

	// Admins can always delete
	if user.Role != "admin" {
		// Non-admins must be the property agent
		if !property.AgentID.Valid || property.AgentID.Int64 != user.ID {
			app.notPermittedResponse(w, r)
			return
		}
	}

	// Delete media from database
	err = app.models.Media.Delete(mediaID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Delete file from disk (ignore errors)
	os.Remove(media.FilePath)

	// Return success response
	err = app.writeJSON(w, http.StatusOK, envelope{"message": "media successfully deleted"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// updatePropertyMediaHandler updates media metadata (caption, display order, primary status)
func (app *application) updatePropertyMediaHandler(w http.ResponseWriter, r *http.Request) {
	// Get property ID
	propertyID, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Get media ID from query string
	mediaIDStr := r.URL.Query().Get("media_id")
	if mediaIDStr == "" {
		app.badRequestResponse(w, r, fmt.Errorf("media_id is required"))
		return
	}

	var mediaID int64
	_, err = fmt.Sscanf(mediaIDStr, "%d", &mediaID)
	if err != nil || mediaID < 1 {
		app.badRequestResponse(w, r, fmt.Errorf("invalid media_id"))
		return
	}

	// Get the media record
	media, err := app.models.Media.Get(mediaID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrPropertyNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Verify media belongs to property
	if media.PropertyID != propertyID {
		app.notFoundResponse(w, r)
		return
	}

	// Check permissions
	property, err := app.models.Properties.Get(propertyID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	user := app.contextGetUser(r)
	if property.AgentID.Valid && property.AgentID.Int64 != user.ID && user.Role != "admin" {
		app.notPermittedResponse(w, r)
		return
	}

	// Parse input
	var input struct {
		Caption      *string `json:"caption"`
		DisplayOrder *int    `json:"display_order"`
		IsPrimary    *bool   `json:"is_primary"`
	}

	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Update fields
	if input.Caption != nil {
		media.Caption = *input.Caption
	}
	if input.DisplayOrder != nil {
		media.DisplayOrder = *input.DisplayOrder
	}
	if input.IsPrimary != nil {
		media.IsPrimary = *input.IsPrimary
	}

	// Validate
	v := validator.New()
	data.ValidateMedia(v, media)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Update in database
	if input.IsPrimary != nil && *input.IsPrimary {
		// Use SetPrimary to handle unique constraint
		err = app.models.Media.SetPrimary(mediaID)
	} else {
		err = app.models.Media.Update(media)
	}

	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Return updated media
	err = app.writeJSON(w, http.StatusOK, envelope{"media": media}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// Helper function to validate media files
func (app *application) validateMediaFile(header *multipart.FileHeader, mediaType string, v *validator.Validator) error {
	// Check file size (max 50MB)
	v.Check(header.Size > 0, "file", "must not be empty")
	v.Check(header.Size <= 50*1024*1024, "file", "must not exceed 50MB")

	// Get file extension
	ext := strings.ToLower(filepath.Ext(header.Filename))

	// Validate based on media type
	switch mediaType {
	case "image":
		validExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
		v.Check(validator.In(ext, validExts...), "file", "must be a valid image format (jpg, jpeg, png, gif, webp)")
	case "video":
		validExts := []string{".mp4", ".mov", ".avi", ".wmv", ".webm"}
		v.Check(validator.In(ext, validExts...), "file", "must be a valid video format (mp4, mov, avi, wmv, webm)")
	case "floor_plan":
		validExts := []string{".jpg", ".jpeg", ".png", ".pdf"}
		v.Check(validator.In(ext, validExts...), "file", "must be a valid floor plan format (jpg, jpeg, png, pdf)")
	}

	return nil
}

// Helper function to save media file to disk
func (app *application) saveMediaFile(file multipart.File, header *multipart.FileHeader, propertyID int64, mediaType string) (string, error) {
	// Create upload directory structure: uploads/properties/{propertyID}/{mediaType}/
	uploadDir := filepath.Join("uploads", "properties", fmt.Sprintf("%d", propertyID), mediaType)
	err := os.MkdirAll(uploadDir, 0755)
	if err != nil {
		return "", err
	}

	// Generate unique filename with timestamp
	timestamp := time.Now().Unix()
	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("%d_%s%s", timestamp, sanitizeFilename(header.Filename), ext)
	filePath := filepath.Join(uploadDir, filename)

	// Create destination file
	dst, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	// Copy uploaded file to destination
	_, err = io.Copy(dst, file)
	if err != nil {
		return "", err
	}

	return filePath, nil
}

// Helper function to sanitize filename
func sanitizeFilename(filename string) string {
	// Remove extension
	name := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Replace spaces and special characters
	name = strings.Map(func(r rune) rune {
		if r == ' ' {
			return '_'
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return -1
	}, name)

	// Limit length
	if len(name) > 50 {
		name = name[:50]
	}

	return name
}
