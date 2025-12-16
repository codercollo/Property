package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/codercollo/property/backend/internal/validator"
	"github.com/google/uuid"
)

const (
	MaxProfilePhotoSize = 5 * 1024 * 1024 // 5MB
	ProfilePhotosDir    = "./uploads/profile_photos"
)

var (
	ErrInvalidFileType = errors.New("invalid file type")
	ErrFileTooLarge    = errors.New("file size exceeds maximum allowed")
)

// AllowedProfilePhotoTypes defines acceptable image MIME types
var AllowedProfilePhotoTypes = map[string]bool{
	"image/jpeg": true,
	"image/jpg":  true,
	"image/png":  true,
	"image/webp": true,
}

// ProfilePhotoUpload represents the profile photo upload request
type ProfilePhotoUpload struct {
	File     multipart.File
	Header   *multipart.FileHeader
	UserID   int64
	PhotoURL string
}

// ValidateProfilePhotoUpload validates the uploaded profile photo
func ValidateProfilePhotoUpload(v *validator.Validator, upload *ProfilePhotoUpload) {
	// Check file exists
	if upload.File == nil || upload.Header == nil {
		v.AddError("photo", "profile photo is required")
		return
	}

	// Check file size
	if upload.Header.Size > MaxProfilePhotoSize {
		v.AddError("photo", fmt.Sprintf("file size must not exceed %d MB", MaxProfilePhotoSize/(1024*1024)))
	}

	// Check file type
	contentType := upload.Header.Header.Get("Content-Type")
	if !AllowedProfilePhotoTypes[contentType] {
		v.AddError("photo", "file must be a valid image (JPEG, PNG, or WebP)")
	}

	// Check filename
	if upload.Header.Filename == "" {
		v.AddError("photo", "filename is required")
	}
}

// SaveProfilePhoto saves the uploaded file to disk and returns the file path
func SaveProfilePhoto(file multipart.File, header *multipart.FileHeader, userID int64) (string, error) {
	// Create uploads directory if it doesn't exist
	if err := os.MkdirAll(ProfilePhotosDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create uploads directory: %w", err)
	}

	// Generate unique filename
	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("%d_%s%s", userID, uuid.New().String(), ext)
	filepath := filepath.Join(ProfilePhotosDir, filename)

	// Create destination file
	dst, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	// Copy uploaded file to destination
	if _, err := io.Copy(dst, file); err != nil {
		// Clean up partial file on error
		os.Remove(filepath)
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	// Return relative URL path
	return fmt.Sprintf("/uploads/profile_photos/%s", filename), nil
}

// DeleteProfilePhoto removes a profile photo from disk
func DeleteProfilePhoto(photoURL string) error {
	if photoURL == "" {
		return nil
	}

	// Extract filename from URL
	parts := strings.Split(photoURL, "/")
	if len(parts) < 2 {
		return nil
	}
	filename := parts[len(parts)-1]
	filepath := filepath.Join(ProfilePhotosDir, filename)

	// Remove file if it exists
	if err := os.Remove(filepath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete photo: %w", err)
	}

	return nil
}

// UpdateProfilePhoto updates the user's profile photo in the database
func (m UserModel) UpdateProfilePhoto(userID int64, photoURL string) error {
	query := `
		UPDATE users
		SET profile_photo = $1, version = version + 1
		WHERE id = $2
		RETURNING version`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var version int
	err := m.DB.QueryRowContext(ctx, query, photoURL, userID).Scan(&version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrUserNotFound
		default:
			return err
		}
	}

	return nil
}

// GetProfilePhoto retrieves the user's profile photo URL
func (m UserModel) GetProfilePhoto(userID int64) (string, error) {
	query := `
		SELECT COALESCE(profile_photo, '') as profile_photo
		FROM users
		WHERE id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var photoURL string
	err := m.DB.QueryRowContext(ctx, query, userID).Scan(&photoURL)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return "", ErrUserNotFound
		default:
			return "", err
		}
	}

	return photoURL, nil
}

// DeleteUserProfilePhoto removes the profile photo URL from the database
func (m UserModel) DeleteUserProfilePhoto(userID int64) error {
	query := `
		UPDATE users
		SET profile_photo = NULL, version = version + 1
		WHERE id = $2`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := m.DB.ExecContext(ctx, query, userID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}
