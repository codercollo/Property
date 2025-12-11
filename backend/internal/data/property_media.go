package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/codercollo/property/backend/internal/validator"
)

// PropertyMedia represents a media file associated with a property
type PropertyMedia struct {
	ID           int64     `json:"id"`
	PropertyID   int64     `json:"property_id"`
	MediaType    string    `json:"media_type"`
	FilePath     string    `json:"file_path"`
	FileName     string    `json:"file_name"`
	FileSize     int64     `json:"file_size"`
	MimeType     string    `json:"mime_type"`
	DisplayOrder int       `json:"display_order"`
	Caption      string    `json:"caption,omitempty"`
	IsPrimary    bool      `json:"is_primary"`
	CreatedAt    time.Time `json:"created_at"`
	Version      int32     `json:"version"`
}

// MediaModel wraps the database connection for property media operations
type MediaModel struct {
	DB *sql.DB
}

// ValidateMedia checks that all fields of a PropertyMedia are valid
func ValidateMedia(v *validator.Validator, media *PropertyMedia) {
	v.Check(media.PropertyID > 0, "property_id", "must be a positive integer")
	v.Check(media.MediaType != "", "media_type", "must be provided")
	v.Check(validator.In(media.MediaType, "image", "video", "floor_plan"), "media_type", "must be one of: image, video, floor_plan")
	v.Check(media.FilePath != "", "file_path", "must be provided")
	v.Check(media.FileName != "", "file_name", "must be provided")
	v.Check(media.FileSize > 0, "file_size", "must be a positive integer")
	v.Check(media.FileSize <= 50*1024*1024, "file_size", "must not exceed 50MB")
	v.Check(media.MimeType != "", "mime_type", "must be provided")
	v.Check(media.DisplayOrder >= 0, "display_order", "must be zero or positive")
	v.Check(len(media.Caption) <= 500, "caption", "must not exceed 500 characters")
}

// Insert adds a new media file to the database
func (m MediaModel) Insert(media *PropertyMedia) error {
	query := `
		INSERT INTO property_media 
		(property_id, media_type, file_path, file_name, file_size, mime_type, display_order, caption, is_primary)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, version`

	args := []interface{}{
		media.PropertyID,
		media.MediaType,
		media.FilePath,
		media.FileName,
		media.FileSize,
		media.MimeType,
		media.DisplayOrder,
		media.Caption,
		media.IsPrimary,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return m.DB.QueryRowContext(ctx, query, args...).Scan(
		&media.ID,
		&media.CreatedAt,
		&media.Version,
	)
}

// GetAllForProperty retrieves all media files for a specific property
func (m MediaModel) GetAllForProperty(propertyID int64) ([]*PropertyMedia, error) {
	query := `
		SELECT id, property_id, media_type, file_path, file_name, file_size, 
		       mime_type, display_order, caption, is_primary, created_at, version
		FROM property_media
		WHERE property_id = $1
		ORDER BY display_order ASC, created_at ASC`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, propertyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mediaList []*PropertyMedia

	for rows.Next() {
		var media PropertyMedia
		err := rows.Scan(
			&media.ID,
			&media.PropertyID,
			&media.MediaType,
			&media.FilePath,
			&media.FileName,
			&media.FileSize,
			&media.MimeType,
			&media.DisplayOrder,
			&media.Caption,
			&media.IsPrimary,
			&media.CreatedAt,
			&media.Version,
		)
		if err != nil {
			return nil, err
		}
		mediaList = append(mediaList, &media)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return mediaList, nil
}

// Get retrieves a specific media file by ID
func (m MediaModel) Get(id int64) (*PropertyMedia, error) {
	if id < 1 {
		return nil, ErrPropertyNotFound
	}

	query := `
		SELECT id, property_id, media_type, file_path, file_name, file_size, 
		       mime_type, display_order, caption, is_primary, created_at, version
		FROM property_media
		WHERE id = $1`

	var media PropertyMedia

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&media.ID,
		&media.PropertyID,
		&media.MediaType,
		&media.FilePath,
		&media.FileName,
		&media.FileSize,
		&media.MimeType,
		&media.DisplayOrder,
		&media.Caption,
		&media.IsPrimary,
		&media.CreatedAt,
		&media.Version,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrPropertyNotFound
		default:
			return nil, err
		}
	}

	return &media, nil
}

// Delete removes a media file from the database
func (m MediaModel) Delete(id int64) error {
	if id < 1 {
		return ErrPropertyNotFound
	}

	query := `DELETE FROM property_media WHERE id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := m.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrPropertyNotFound
	}

	return nil
}

// Update modifies an existing media file's metadata
func (m MediaModel) Update(media *PropertyMedia) error {
	query := `
		UPDATE property_media
		SET display_order = $1,
		    caption = $2,
		    is_primary = $3,
		    version = version + 1
		WHERE id = $4 AND version = $5
		RETURNING version`

	args := []interface{}{
		media.DisplayOrder,
		media.Caption,
		media.IsPrimary,
		media.ID,
		media.Version,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&media.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}

	return nil
}

// DeleteAllForProperty removes all media files for a specific property
func (m MediaModel) DeleteAllForProperty(propertyID int64) error {
	query := `DELETE FROM property_media WHERE property_id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, propertyID)
	return err
}

// SetPrimary sets a media file as the primary media for its type
func (m MediaModel) SetPrimary(id int64) error {
	// First, get the media to know its property_id and media_type
	media, err := m.Get(id)
	if err != nil {
		return err
	}

	// Start a transaction
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	tx, err := m.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Unset all primary flags for this property and media type
	query1 := `
		UPDATE property_media
		SET is_primary = false
		WHERE property_id = $1 AND media_type = $2`

	_, err = tx.ExecContext(ctx, query1, media.PropertyID, media.MediaType)
	if err != nil {
		return err
	}

	// Set this media as primary
	query2 := `
		UPDATE property_media
		SET is_primary = true, version = version + 1
		WHERE id = $1
		RETURNING version`

	err = tx.QueryRowContext(ctx, query2, id).Scan(&media.Version)
	if err != nil {
		return err
	}

	return tx.Commit()
}
