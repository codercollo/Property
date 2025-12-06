package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/codercollo/property/backend/internal/validator"
)

// Review represents a property review
type Review struct {
	ID         int64      `json:"id"`
	CreatedAt  time.Time  `json:"created_at"`
	PropertyID int64      `json:"property_id"`
	UserID     int64      `json:"user_id"`
	UserName   string     `json:"user_name"`
	Rating     int32      `json:"rating"`
	Comment    string     `json:"comment"`
	Status     string     `json:"status"` // pending, approved, rejected
	ApprovedAt *time.Time `json:"approved_at,omitempty"`
	ApprovedBy *int64     `json:"approved_by,omitempty"`
	Version    int32      `json:"version"`
}

// ValidateReview checks that all fields of a Review are valid
func ValidateReview(v *validator.Validator, review *Review) {
	// Validate property ID
	v.Check(review.PropertyID > 0, "property_id", "must be a positive value")

	// Validate rating
	v.Check(review.Rating > 0, "rating", "must be provided")
	v.Check(review.Rating >= 1, "rating", "must be at least 1")
	v.Check(review.Rating <= 5, "rating", "must not be more than 5")

	// Validate comment
	v.Check(review.Comment != "", "comment", "must be provided")
	v.Check(len(review.Comment) >= 10, "comment", "must be at least 10 characters long")
	v.Check(len(review.Comment) <= 1000, "comment", "must not be more than 1000 characters long")

	// Validate status
	validStatuses := []string{"pending", "approved", "rejected"}
	v.Check(validator.In(review.Status, validStatuses...), "status", "must be one of: pending, approved, rejected")
}

// ReviewModel wraps a sql.DB connection pool for reviews
type ReviewModel struct {
	DB *sql.DB
}

// Insert adds a new review
func (r ReviewModel) Insert(review *Review) error {
	//SQL query to insert a review and return id, created_at, version
	query := `
		INSERT INTO reviews (property_id, user_id, rating, comment, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, version`

	//Create acontext with 3 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	//Arguments fo the SQL query
	args := []interface{}{
		review.PropertyID,
		review.UserID,
		review.Rating,
		review.Comment,
		review.Status,
	}

	//Execute the query and scan returned id, created_at, version into the struct
	return r.DB.QueryRowContext(ctx, query, args...).Scan(
		&review.ID,
		&review.CreatedAt,
		&review.Version,
	)

}

// Get retrives a review by ID, including the user's name
func (r ReviewModel) Get(id int64) (*Review, error) {
	//Validate ID
	if id < 1 {
		return nil, ErrReviewNotFound
	}

	//SQL query to select review and associated user name
	query := `
		SELECT r.id, r.created_at, r.property_id, r.user_id, u.name as user_name,
		       r.rating, r.comment, r.status, r.approved_at, r.approved_by, r.version
		FROM reviews r
		INNER JOIN users u ON r.user_id = u.id
		WHERE r.id = $1`

	var review Review

	//Create context with 3 seconds timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	//Execute query and scan result into review struct
	err := r.DB.QueryRowContext(ctx, query, id).Scan(
		&review.ID,
		&review.CreatedAt,
		&review.PropertyID,
		&review.UserID,
		&review.UserName,
		&review.Rating,
		&review.Comment,
		&review.Status,
		&review.ApprovedAt,
		&review.ApprovedBy,
		&review.Version,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrReviewNotFound
		default:
			return nil, err
		}
	}

	return &review, nil

}

// GetAllForProperty retrieves approved reviews for a specific property with pagination
// and returns metadata
func (r ReviewModel) GetAllForProperty(propertyID int64, filters Filters) ([]*Review, Metadata, error) {
	//SQL query to select approved reviews with total count for metadata
	query := `
		SELECT count(*) OVER(), r.id, r.created_at, r.property_id, r.user_id, u.name as user_name,
		       r.rating, r.comment, r.status, r.approved_at, r.approved_by, r.version
		FROM reviews r
		INNER JOIN users u ON r.user_id = u.id
		WHERE r.property_id = $1 AND r.status = 'approved'
		ORDER BY r.created_at DESC
		LIMIT $2 OFFSET $3`

	//Create context with 3-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []interface{}{
		propertyID,
		filters.limit(),
		filters.offset(),
	}

	//Execute query
	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	reviews := []*Review{}
	totalRecords := 0

	//Iterate throught rows and scan data into reviews struct
	for rows.Next() {
		var review Review
		err := rows.Scan(
			&totalRecords,
			&review.ID,
			&review.CreatedAt,
			&review.PropertyID,
			&review.UserID,
			&review.UserName,
			&review.Rating,
			&review.Comment,
			&review.Status,
			&review.ApprovedAt,
			&review.ApprovedBy,
			&review.Version,
		)
		if err != nil {
			return nil, Metadata{}, err
		}
		reviews = append(reviews, &review)
	}

	//Check for row iteration errors
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	//Calculate pagination metadata
	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)
	return reviews, metadata, nil
}

// GetAllPending retrieves all reviews with 'pending' status and returns pagination metadata
func (r ReviewModel) GetAllPending(filters Filters) ([]*Review, Metadata, error) {
	//SQL query to select pending reviews with total count
	query := `
		SELECT count(*) OVER(), r.id, r.created_at, r.property_id, r.user_id, u.name as user_name,
		       r.rating, r.comment, r.status, r.approved_at, r.approved_by, r.version
		FROM reviews r
		INNER JOIN users u ON r.user_id = u.id
		WHERE r.status = 'pending'
		ORDER BY r.created_at DESC
		LIMIT $1 OFFSET $2`

	//Context with 3-Sec
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []interface{}{
		filters.limit(),
		filters.offset(),
	}

	//Execute query
	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err

	}
	defer rows.Close()

	reviews := []*Review{}
	totalRecords := 0

	// Scan rows into Review structs
	for rows.Next() {
		var review Review
		if err := rows.Scan(
			&totalRecords,
			&review.ID,
			&review.CreatedAt,
			&review.PropertyID,
			&review.UserID,
			&review.UserName,
			&review.Rating,
			&review.Comment,
			&review.Status,
			&review.ApprovedAt,
			&review.ApprovedBy,
			&review.Version,
		); err != nil {
			return nil, Metadata{}, err
		}
		reviews = append(reviews, &review)
	}

	// Check for row iteration errors
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	// Return reviews with pagination metadata
	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)
	return reviews, metadata, nil
}

// Approve marks a pending review as approved and updates its version.
func (r ReviewModel) Approve(id, approverID int64) error {
	// Validate review ID
	if id < 1 {
		return ErrReviewNotFound
	}

	// SQL query to approve the review and increment version
	query := `
		UPDATE reviews
		SET status = 'approved', approved_at = NOW(), approved_by = $1, version = version + 1
		WHERE id = $2 AND status = 'pending'
		RETURNING version`

	// Context with 3-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var newVersion int32

	// Execute query and scan new version
	err := r.DB.QueryRowContext(ctx, query, approverID, id).Scan(&newVersion)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrReviewNotFound
		}
		return err
	}

	return nil
}

// Reject marks a pending review as rejected and updates its version.
func (r ReviewModel) Reject(id, approverID int64) error {
	// Validate review ID
	if id < 1 {
		return ErrReviewNotFound
	}

	// SQL query to reject the review and increment version
	query := `
		UPDATE reviews
		SET status = 'rejected', approved_by = $1, version = version + 1
		WHERE id = $2 AND status = 'pending'
		RETURNING version`

	// Context with 3-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var newVersion int32

	// Execute query and scan new version
	err := r.DB.QueryRowContext(ctx, query, approverID, id).Scan(&newVersion)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrReviewNotFound
		}
		return err
	}

	return nil
}

// Delete removes a review by its ID.
func (r ReviewModel) Delete(id int64) error {
	// Validate review ID
	if id < 1 {
		return ErrReviewNotFound
	}

	// SQL query to delete the review
	query := `DELETE FROM reviews WHERE id = $1`

	// Context with 3-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Execute deletion
	result, err := r.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	// Check if any row was deleted
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrReviewNotFound
	}

	return nil
}

// GetAverageRating calculates the average rating and total count of approved reviews for a property.
func (r ReviewModel) GetAverageRating(propertyID int64) (float64, int, error) {
	// SQL query to calculate average rating and count
	query := `
		SELECT COALESCE(AVG(rating), 0), COUNT(*)
		FROM reviews
		WHERE property_id = $1 AND status = 'approved'`

	// Context with 3-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var avgRating float64
	var count int

	// Execute query and scan results
	err := r.DB.QueryRowContext(ctx, query, propertyID).Scan(&avgRating, &count)
	if err != nil {
		return 0, 0, err
	}

	return avgRating, count, nil
}

// GetAllForAgent retrieves all reviews for properties belonging to a specific agent
func (m ReviewModel) GetAllForAgent(agentID int64, filters Filters) ([]*Review, Metadata, error) {
	query := `
		SELECT count(*) OVER(), r.id, r.created_at, r.property_id, r.user_id, u.name as user_name,
		       r.rating, r.comment, r.status, r.approved_at, r.approved_by, r.version
		FROM reviews r
		INNER JOIN users u ON r.user_id = u.id
		INNER JOIN properties p ON r.property_id = p.id
		WHERE p.agent_id = $1
		ORDER BY r.created_at DESC
		LIMIT $2 OFFSET $3`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []interface{}{agentID, filters.limit(), filters.offset()}

	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	reviews := []*Review{}
	totalRecords := 0

	for rows.Next() {
		var review Review
		err := rows.Scan(
			&totalRecords,
			&review.ID,
			&review.CreatedAt,
			&review.PropertyID,
			&review.UserID,
			&review.UserName,
			&review.Rating,
			&review.Comment,
			&review.Status,
			&review.ApprovedAt,
			&review.ApprovedBy,
			&review.Version,
		)
		if err != nil {
			return nil, Metadata{}, err
		}
		reviews = append(reviews, &review)
	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)

	return reviews, metadata, nil
}

// GetPendingForAgent retrieves pending reviews for properties belonging to a specific agent
func (m ReviewModel) GetPendingForAgent(agentID int64, filters Filters) ([]*Review, Metadata, error) {
	query := `
		SELECT count(*) OVER(), r.id, r.created_at, r.property_id, r.user_id, u.name as user_name,
		       r.rating, r.comment, r.status, r.approved_at, r.approved_by, r.version
		FROM reviews r
		INNER JOIN users u ON r.user_id = u.id
		INNER JOIN properties p ON r.property_id = p.id
		WHERE p.agent_id = $1 AND r.status = 'pending'
		ORDER BY r.created_at DESC
		LIMIT $2 OFFSET $3`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []interface{}{agentID, filters.limit(), filters.offset()}

	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	reviews := []*Review{}
	totalRecords := 0

	for rows.Next() {
		var review Review
		err := rows.Scan(
			&totalRecords,
			&review.ID,
			&review.CreatedAt,
			&review.PropertyID,
			&review.UserID,
			&review.UserName,
			&review.Rating,
			&review.Comment,
			&review.Status,
			&review.ApprovedAt,
			&review.ApprovedBy,
			&review.Version,
		)
		if err != nil {
			return nil, Metadata{}, err
		}
		reviews = append(reviews, &review)
	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)

	return reviews, metadata, nil
}
