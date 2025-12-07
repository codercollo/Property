package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/codercollo/property/backend/internal/validator"
	"github.com/lib/pq"
)

// Property represents a real estate listing returned in API responses
// Fields use JSON tags with omitempty to hide zero-value data/ when fields are empty
type Property struct {
	ID           int64      `json:"id"`
	CreatedAt    time.Time  `json:"-"`
	Title        string     `json:"title,"`
	YearBuilt    int32      `json:"year_built,omitempty"`
	Area         Area       `json:"area,omitempty"`
	Bedrooms     Bedrooms   `json:"bedrooms,omitempty"`
	Bathrooms    Bathrooms  `json:"bathrooms,omitempty"`
	Floor        Floor      `json:"floor,omitempty"`
	Price        Price      `json:"price,omitempty"`
	Location     string     `json:"location,"`
	PropertyType string     `json:"property_type,"`
	Features     []string   `json:"features,omitempty"`
	Images       []string   `json:"images,omitempty"`
	FeaturedAt   *time.Time `json:"featured_at,omitempty"`
	AgentID      int64      `json:"agent_id,omitempty"`
	Version      int32      `json:"version"`
}

// PropertyStats holds statistics about an agent's properties
type PropertyStats struct {
	TotalProperties    int `json:"total_properties"`
	FeaturedProperties int `json:"featured_properties"`
	PendingProperties  int `json:"pending_properties"`
}

// ValidateProperty checks that all fields of a Property are valid
func ValidateProperty(v *validator.Validator, property *Property) {
	// Validate title
	v.Check(property.Title != "", "title", "must be provided")
	v.Check(len(property.Title) <= 500, "title", "must not be more than 500 bytes long")

	// Validate year built
	v.Check(property.YearBuilt != 0, "year_built", "must be provided")
	v.Check(property.YearBuilt >= 1800, "year_built", "must be greater than 1800")
	v.Check(property.YearBuilt <= int32(time.Now().Year()), "year_built", "must not be in the future")

	// Validate numeric property details
	v.Check(property.Area > 0, "area", "must be a positive value")
	v.Check(property.Bedrooms > 0, "bedrooms", "must be a positive value")
	v.Check(property.Bathrooms >= 0, "bathrooms", "must be zero or more")
	v.Check(property.Floor >= 0, "floor", "must be zero or more")
	v.Check(property.Price > 0, "price", "must be a positive value")

	// Validate location and type
	v.Check(property.Location != "", "location", "must be provided")
	v.Check(property.PropertyType != "", "property_type", "must be provided")

	// Validate features list
	v.Check(property.Features != nil, "features", "must be provided")
	v.Check(len(property.Features) >= 1, "features", "must contain at least 1 feature")
	v.Check(len(property.Features) <= 10, "features", "must not contain more than 10 features")
	v.Check(validator.Unique(property.Features), "features", "must not contain duplicate values")

	// Validate images list
	v.Check(property.Images != nil, "images", "must be provided")
	v.Check(len(property.Images) >= 1, "images", "must contain at least 1 image")
	v.Check(len(property.Images) <= 10, "images", "must not contain more than 10 images")
	v.Check(validator.Unique(property.Images), "images", "must not contain duplicate values")
}

// PropertyModel wraps a sql.DB connection pool for properties table operations
type PropertyModel struct {
	DB *sql.DB
}

// Insert adds a new property listing
func (p PropertyModel) Insert(property *Property) error {
	//SQL query for inserting a new property and returning system-generated fields.
	query := `
		INSERT INTO properties 
		(title, year_built, area, bedrooms, bathrooms, floor, price, location, property_type, features, images, agent_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, version
                `
	//Create a context with a 3 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	//Arguments for the SQL placeholder
	args := []interface{}{
		property.Title,
		property.YearBuilt,
		property.Area,
		property.Bedrooms,
		property.Bathrooms,
		property.Floor,
		property.Price,
		property.Location,
		property.PropertyType,
		pq.Array(property.Features),
		pq.Array(property.Images),
		property.AgentID,
	}

	//Execute the query and scan the returned values into the property struct
	return p.DB.QueryRowContext(ctx, query, args...).Scan(
		&property.ID,
		&property.CreatedAt,
		&property.Version,
	)
}

// Get retrieves a property by ID
func (p PropertyModel) Get(id int64) (*Property, error) {
	//Shorcut: ID'S less than 1 dont't exist
	if id < 1 {
		return nil, ErrPropertyNotFound
	}

	//SQL query to fetch a property by ID
	query := `
	SELECT id, created_at, title, year_built, area, bedrooms, bathrooms, floor, price, 
	location, property_type, features, images, featured_at, agent_id, version
	FROM properties
	WHERE id = $1`

	//Declare a Property struct to hold query results
	var property Property

	//Context which has 3-sec timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	//Execute the query together with passing the context and scan results
	err := p.DB.QueryRowContext(ctx, query, id).Scan(
		&property.ID,
		&property.CreatedAt,
		&property.Title,
		&property.YearBuilt,
		&property.Area,
		&property.Bedrooms,
		&property.Bathrooms,
		&property.Floor,
		&property.Price,
		&property.Location,
		&property.PropertyType,
		pq.Array(&property.Features),
		pq.Array(&property.Images),
		&property.FeaturedAt,
		&property.AgentID,
		&property.Version,
	)

	//Handle errors
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrPropertyNotFound
		default:
			return nil, err
		}
	}

	//Return the property
	return &property, nil

}

// GetAll retrieves property listings with optional filtering, sorting, and pagination.
// Returns a slice of Property pointers and pagination Metadata.
func (p PropertyModel) GetAll(title, location, propertyType string, features []string, filters Filters) ([]*Property, Metadata, error) {
	// SQL query with filtering, sorting, pagination, and total count using a window function
	query := fmt.Sprintf(`
	SELECT count(*) OVER(), id, created_at, title, year_built, area, bedrooms, bathrooms,
	       floor, price, location, property_type, features, images, featured_at, agent_id, version
	FROM properties
	WHERE (to_tsvector('simple', title) @@ plainto_tsquery('simple', $1) OR $1 = '')
	AND (features @> $2 OR $2 = '{}')
	AND (location ILIKE '%%' || $3 || '%%' OR $3 = '')
	AND (property_type ILIKE '%%' || $4 || '%%' OR $4 = '')
	ORDER BY %s %s, id ASC
	LIMIT $5 OFFSET $6`, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if features == nil {
		features = []string{}
	}

	// Arguments for placeholders
	args := []interface{}{title, pq.Array(features), location, propertyType, filters.limit(), filters.offset()}

	// Execute query
	rows, err := p.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	properties := []*Property{}
	totalListings := 0

	// Scan rows into Property structs and capture total count
	for rows.Next() {
		var property Property
		err := rows.Scan(
			&totalListings,
			&property.ID,
			&property.CreatedAt,
			&property.Title,
			&property.YearBuilt,
			&property.Area,
			&property.Bedrooms,
			&property.Bathrooms,
			&property.Floor,
			&property.Price,
			&property.Location,
			&property.PropertyType,
			pq.Array(&property.Features),
			pq.Array(&property.Images),
			&property.FeaturedAt,
			&property.AgentID,
			&property.Version,
		)
		if err != nil {
			return nil, Metadata{}, err
		}
		properties = append(properties, &property)
	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	// Generate pagination metadata
	metadata := calculateMetadata(totalListings, filters.Page, filters.PageSize)

	// Include the metadata struct when returning.
	return properties, metadata, nil
}

// Update modifies an existing movie record
func (p PropertyModel) Update(property *Property) error {
	//SQL quesry to update a propertu and increment its version
	query := `
UPDATE properties
SET 
    title = $1,
    year_built = $2,
    area = $3,
    bedrooms = $4,
    bathrooms = $5,
    floor = $6,
    price = $7,
    location = $8,
    property_type = $9,
    features = $10,
    images = $11,
    agent_id = $12,
    version = version + 1
WHERE id = $13 AND version = $14
RETURNING version
`
	//Create a context with a 3-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Parameters for the query including version for optimistic locking
	args := []interface{}{
		property.Title,
		property.YearBuilt,
		property.Area,
		property.Bedrooms,
		property.Bathrooms,
		property.Floor,
		property.Price,
		property.Location,
		property.PropertyType,
		pq.Array(property.Features),
		pq.Array(property.Images),
		property.AgentID,
		property.ID,
		property.Version,
	}

	//Execute the update and scan the new version
	err := p.DB.QueryRowContext(ctx, query, args...).Scan(&property.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}

	//Update succeeded
	return nil

}

// Delete removes a property by ID
func (p PropertyModel) Delete(id int64) error {
	//Return ErrPropertyNotFound if the ID is invalid
	if id < 1 {
		return ErrPropertyNotFound

	}

	//SQL query to delete a property by ID
	query := `
        DELETE FROM properties
        WHERE id = $1
    `
	//Execute the query
	results, err := p.DB.Exec(query, id)
	if err != nil {
		return err
	}

	//check how many rows were affected
	rowsAffected, err := results.RowsAffected()
	if err != nil {
		return nil
	}

	//Return ErrPropertyNotFound if no row was deleted
	if rowsAffected == 0 {
		return ErrPropertyNotFound
	}

	return nil

}

// Feature marks a property as featured by setting FeaturedAt to now
func (p PropertyModel) Feature(id int64) error {
	//invalid ID
	if id < 1 {
		return ErrPropertyNotFound
	}

	//SQL to update FeaturedAt and increment version
	query := `
		UPDATE properties
		SET featured_at = NOW(), version = version + 1
		WHERE id = $1
		RETURNING version
	`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var newVersion int32

	err := p.DB.QueryRowContext(ctx, query, id).Scan(&newVersion)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPropertyNotFound
		}
		return err
	}

	return nil
}

// Unfeature clears FeaturedAt to mark a property as not featured
func (p PropertyModel) Unfeature(id int64) error {
	//invalid property
	if id < 1 {
		return ErrPropertyNotFound
	}

	//SQL to clear FeaturedAt and increment version
	query := `
		UPDATE properties
		SET featured_at = NULL, version = version + 1
		WHERE id = $1
		RETURNING version
	`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var newVersion int32

	err := p.DB.QueryRowContext(ctx, query, id).Scan(&newVersion)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPropertyNotFound
		}
		return err
	}

	return nil
}

// Replace the GetAllForAgent method in your property.go model file

// GetAllForAgent retrieves all properties belonging to a specific agent
func (p PropertyModel) GetAllForAgent(agentID int64, filters Filters) ([]*Property, Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), id, created_at, title, year_built, area, bedrooms, 
		       bathrooms, floor, price, location, property_type, features, images, 
		       featured_at, agent_id, version
		FROM properties
		WHERE agent_id = $1
		ORDER BY %s %s, id ASC
		LIMIT $2 OFFSET $3`, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []interface{}{agentID, filters.limit(), filters.offset()}

	rows, err := p.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	properties := []*Property{}
	totalRecords := 0

	for rows.Next() {
		var property Property
		err := rows.Scan(
			&totalRecords,
			&property.ID,
			&property.CreatedAt,
			&property.Title,
			&property.YearBuilt,
			&property.Area,
			&property.Bedrooms,
			&property.Bathrooms,
			&property.Floor,
			&property.Price,
			&property.Location,
			&property.PropertyType,
			pq.Array(&property.Features),
			pq.Array(&property.Images),
			&property.FeaturedAt,
			&property.AgentID,
			&property.Version,
		)
		if err != nil {
			return nil, Metadata{}, err
		}
		properties = append(properties, &property)
	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)

	return properties, metadata, nil
}

// GetStatsForAgent returns property statistics for a specific agent
func (p PropertyModel) GetStatsForAgent(agentID int64) (*PropertyStats, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN featured_at IS NOT NULL THEN 1 END) as featured,
			0 as pending
		FROM properties
		WHERE agent_id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var stats PropertyStats

	err := p.DB.QueryRowContext(ctx, query, agentID).Scan(
		&stats.TotalProperties,
		&stats.FeaturedProperties,
		&stats.PendingProperties,
	)

	if err != nil {
		return nil, err
	}

	return &stats, nil
}

// GetAllForAdmin retrieves all properties with filtering for admin view
func (p PropertyModel) GetAllForAdmin(status string, filters Filters) ([]*Property, Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), id, created_at, title, year_built, area, bedrooms, 
		       bathrooms, floor, price, location, property_type, features, images, 
		       featured_at, agent_id, version
		FROM properties
		WHERE (status = $1 OR $1 = '')
		ORDER BY %s %s, id DESC
		LIMIT $2 OFFSET $3`, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []interface{}{status, filters.limit(), filters.offset()}

	rows, err := p.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	properties := []*Property{}
	totalRecords := 0

	for rows.Next() {
		var property Property
		err := rows.Scan(
			&totalRecords,
			&property.ID,
			&property.CreatedAt,
			&property.Title,
			&property.YearBuilt,
			&property.Area,
			&property.Bedrooms,
			&property.Bathrooms,
			&property.Floor,
			&property.Price,
			&property.Location,
			&property.PropertyType,
			pq.Array(&property.Features),
			pq.Array(&property.Images),
			&property.FeaturedAt,
			&property.AgentID,
			&property.Version,
		)
		if err != nil {
			return nil, Metadata{}, err
		}
		properties = append(properties, &property)
	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)

	return properties, metadata, nil
}

// ApproveProperty approves a property listing
func (p PropertyModel) ApproveProperty(id, adminID int64) error {
	query := `
		UPDATE properties
		SET status = 'approved', 
		    moderated_by = $1, 
		    moderated_at = NOW(),
		    rejection_reason = NULL,
		    version = version + 1
		WHERE id = $2
		RETURNING version`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var newVersion int32

	err := p.DB.QueryRowContext(ctx, query, adminID, id).Scan(&newVersion)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPropertyNotFound
		}
		return err
	}

	return nil
}

// RejectProperty rejects a property listing with a reason
func (p PropertyModel) RejectProperty(id, adminID int64, reason string) error {
	query := `
		UPDATE properties
		SET status = 'rejected', 
		    moderated_by = $1, 
		    moderated_at = NOW(),
		    rejection_reason = $2,
		    version = version + 1
		WHERE id = $3
		RETURNING version`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var newVersion int32

	err := p.DB.QueryRowContext(ctx, query, adminID, reason, id).Scan(&newVersion)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPropertyNotFound
		}
		return err
	}

	return nil
}
