package data

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
)

// PropertySearchCriteria holds all possible search filters
type PropertySearchCriteria struct {
	Location     string
	PropertyType string
	Status       string // featured, standard, all
	MinPrice     float64
	MaxPrice     float64
	MinBedrooms  int32
	MaxBedrooms  int32
	MinBathrooms int32
	MaxBathrooms int32
	MinArea      int32
	MaxArea      int32
	Features     []string
}

// AvailableFilters represents all available filter options
type AvailableFilters struct {
	PropertyTypes []string  `json:"property_types"`
	Locations     []string  `json:"locations"`
	Features      []string  `json:"features"`
	PriceRange    PriceInfo `json:"price_range"`
	BedroomRange  RangeInfo `json:"bedroom_range"`
	BathroomRange RangeInfo `json:"bathroom_range"`
	AreaRange     RangeInfo `json:"area_range"`
}

type PriceInfo struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

type RangeInfo struct {
	Min int32 `json:"min"`
	Max int32 `json:"max"`
}

// AdvancedSearch performs a comprehensive property search with multiple filters
func (p PropertyModel) AdvancedSearch(criteria PropertySearchCriteria, filters Filters) ([]*Property, Metadata, error) {
	// Build dynamic WHERE clauses
	var whereClauses []string
	var args []interface{}
	argPosition := 1

	// Location filter (case-insensitive partial match)
	if criteria.Location != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("location ILIKE $%d", argPosition))
		args = append(args, "%"+criteria.Location+"%")
		argPosition++
	}

	// Property type filter (case-insensitive partial match)
	if criteria.PropertyType != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("property_type ILIKE $%d", argPosition))
		args = append(args, "%"+criteria.PropertyType+"%")
		argPosition++
	}

	// Status filter (featured vs standard)
	if criteria.Status == "featured" {
		whereClauses = append(whereClauses, "featured_at IS NOT NULL")
	} else if criteria.Status == "standard" {
		whereClauses = append(whereClauses, "featured_at IS NULL")
	}
	// If "all", no status filter is added

	// Price range filters
	if criteria.MinPrice > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("price >= $%d", argPosition))
		args = append(args, criteria.MinPrice)
		argPosition++
	}
	if criteria.MaxPrice > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("price <= $%d", argPosition))
		args = append(args, criteria.MaxPrice)
		argPosition++
	}

	// Bedrooms range filters
	if criteria.MinBedrooms > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("bedrooms >= $%d", argPosition))
		args = append(args, criteria.MinBedrooms)
		argPosition++
	}
	if criteria.MaxBedrooms > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("bedrooms <= $%d", argPosition))
		args = append(args, criteria.MaxBedrooms)
		argPosition++
	}

	// Bathrooms range filters
	if criteria.MinBathrooms > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("bathrooms >= $%d", argPosition))
		args = append(args, criteria.MinBathrooms)
		argPosition++
	}
	if criteria.MaxBathrooms > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("bathrooms <= $%d", argPosition))
		args = append(args, criteria.MaxBathrooms)
		argPosition++
	}

	// Area range filters
	if criteria.MinArea > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("area >= $%d", argPosition))
		args = append(args, criteria.MinArea)
		argPosition++
	}
	if criteria.MaxArea > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("area <= $%d", argPosition))
		args = append(args, criteria.MaxArea)
		argPosition++
	}

	// Features/amenities filter (must have all specified features)
	if len(criteria.Features) > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("features @> $%d", argPosition))
		args = append(args, pq.Array(criteria.Features))
		argPosition++
	}

	// Combine WHERE clauses
	whereSQL := "TRUE" // Default to no filters
	if len(whereClauses) > 0 {
		whereSQL = strings.Join(whereClauses, " AND ")
	}

	// Add pagination arguments
	args = append(args, filters.limit(), filters.offset())

	// Build complete query
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), id, created_at, title, year_built, area, bedrooms, 
		       bathrooms, floor, price, location, property_type, features, images, 
		       featured_at, agent_id, version
		FROM properties
		WHERE %s
		ORDER BY %s %s, id ASC
		LIMIT $%d OFFSET $%d`,
		whereSQL,
		filters.sortColumn(),
		filters.sortDirection(),
		argPosition,
		argPosition+1,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

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

// GetAvailableFilters returns all available filter options from the database
func (p PropertyModel) GetAvailableFilters() (*AvailableFilters, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	filters := &AvailableFilters{}

	// Get distinct property types
	typeQuery := `SELECT DISTINCT property_type FROM properties WHERE property_type != '' ORDER BY property_type`
	typeRows, err := p.DB.QueryContext(ctx, typeQuery)
	if err != nil {
		return nil, err
	}
	defer typeRows.Close()

	for typeRows.Next() {
		var propType string
		if err := typeRows.Scan(&propType); err != nil {
			return nil, err
		}
		filters.PropertyTypes = append(filters.PropertyTypes, propType)
	}

	// Get distinct locations
	locQuery := `SELECT DISTINCT location FROM properties WHERE location != '' ORDER BY location`
	locRows, err := p.DB.QueryContext(ctx, locQuery)
	if err != nil {
		return nil, err
	}
	defer locRows.Close()

	for locRows.Next() {
		var location string
		if err := locRows.Scan(&location); err != nil {
			return nil, err
		}
		filters.Locations = append(filters.Locations, location)
	}

	// Get all unique features
	featureQuery := `SELECT DISTINCT unnest(features) as feature FROM properties ORDER BY feature`
	featureRows, err := p.DB.QueryContext(ctx, featureQuery)
	if err != nil {
		return nil, err
	}
	defer featureRows.Close()

	for featureRows.Next() {
		var feature string
		if err := featureRows.Scan(&feature); err != nil {
			return nil, err
		}
		filters.Features = append(filters.Features, feature)
	}

	// Get price range
	priceQuery := `SELECT MIN(price), MAX(price) FROM properties WHERE price > 0`
	err = p.DB.QueryRowContext(ctx, priceQuery).Scan(&filters.PriceRange.Min, &filters.PriceRange.Max)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Get bedroom range
	bedroomQuery := `SELECT MIN(bedrooms), MAX(bedrooms) FROM properties WHERE bedrooms > 0`
	err = p.DB.QueryRowContext(ctx, bedroomQuery).Scan(&filters.BedroomRange.Min, &filters.BedroomRange.Max)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Get bathroom range
	bathroomQuery := `SELECT MIN(bathrooms), MAX(bathrooms) FROM properties WHERE bathrooms >= 0`
	err = p.DB.QueryRowContext(ctx, bathroomQuery).Scan(&filters.BathroomRange.Min, &filters.BathroomRange.Max)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Get area range
	areaQuery := `SELECT MIN(area), MAX(area) FROM properties WHERE area > 0`
	err = p.DB.QueryRowContext(ctx, areaQuery).Scan(&filters.AreaRange.Min, &filters.AreaRange.Max)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return filters, nil
}
