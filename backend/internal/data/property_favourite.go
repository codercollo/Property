package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// Favourite represents a user's saved property
type Favourite struct {
	UserID     int64     `json:"user_id"`
	PropertyID int64     `json:"property_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// FavouriteProperty represents a favourite with full property details
type FavouriteProperty struct {
	UserID       int64     `json:"user_id"`
	PropertyID   int64     `json:"property_id"`
	SavedAt      time.Time `json:"saved_at"`
	Property     *Property `json:"property"`
	IsFavourited bool      `json:"is_favourited"` // Helper field for client
}

// FavouriteStats holds statistics about a user's favourites
type FavouriteStats struct {
	TotalFavourites int     `json:"total_favourites"`
	AveragePrice    float64 `json:"average_price"`
	TotalValue      float64 `json:"total_value"`
}

var (
	ErrFavouriteNotFound      = errors.New("favourite not found")
	ErrFavouriteAlreadyExists = errors.New("property already favourited")
)

// FavouriteModel wraps the database connection for favourite operations
type FavouriteModel struct {
	DB *sql.DB
}

// Add creates a new favourite for a user
func (m FavouriteModel) Add(userID, propertyID int64) error {
	query := `
		INSERT INTO user_favourites (user_id, property_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, property_id) DO NOTHING`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := m.DB.ExecContext(ctx, query, userID, propertyID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	// If no rows affected, the favourite already exists
	if rowsAffected == 0 {
		return ErrFavouriteAlreadyExists
	}

	return nil
}

// Remove deletes a favourite
func (m FavouriteModel) Remove(userID, propertyID int64) error {
	query := `
		DELETE FROM user_favourites 
		WHERE user_id = $1 AND property_id = $2`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := m.DB.ExecContext(ctx, query, userID, propertyID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrFavouriteNotFound
	}

	return nil
}

// IsFavourite checks if a property is favourited by a user
func (m FavouriteModel) IsFavourite(userID, propertyID int64) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM user_favourites 
			WHERE user_id = $1 AND property_id = $2
		)`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var exists bool
	err := m.DB.QueryRowContext(ctx, query, userID, propertyID).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// GetAllForUser retrieves all favourited properties for a user with full property details
func (m FavouriteModel) GetAllForUser(userID int64, filters Filters) ([]*FavouriteProperty, Metadata, error) {
	// Map sort columns to their proper table-qualified names
	sortColumn := filters.sortColumn()
	switch sortColumn {
	case "created_at":
		sortColumn = "uf.created_at"
	case "price":
		sortColumn = "p.price"
	case "title":
		sortColumn = "p.title"
	case "id":
		sortColumn = "p.id"
	}

	query := fmt.Sprintf(`
		SELECT count(*) OVER(),
		       uf.user_id, uf.property_id, uf.created_at,
		       p.id, p.created_at, p.title, p.year_built, p.area, p.bedrooms, 
		       p.bathrooms, p.floor, p.price, p.location, p.property_type, 
		       p.features, p.images, p.featured_at, p.agent_id, p.version
		FROM user_favourites uf
		INNER JOIN properties p ON uf.property_id = p.id
		WHERE uf.user_id = $1
		ORDER BY %s %s
		LIMIT $2 OFFSET $3`, sortColumn, filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []interface{}{userID, filters.limit(), filters.offset()}

	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	favourites := []*FavouriteProperty{}
	totalRecords := 0

	for rows.Next() {
		var fav FavouriteProperty
		fav.Property = &Property{}

		err := rows.Scan(
			&totalRecords,
			&fav.UserID,
			&fav.PropertyID,
			&fav.SavedAt,
			&fav.Property.ID,
			&fav.Property.CreatedAt,
			&fav.Property.Title,
			&fav.Property.YearBuilt,
			&fav.Property.Area,
			&fav.Property.Bedrooms,
			&fav.Property.Bathrooms,
			&fav.Property.Floor,
			&fav.Property.Price,
			&fav.Property.Location,
			&fav.Property.PropertyType,
			pq.Array(&fav.Property.Features),
			pq.Array(&fav.Property.Images),
			&fav.Property.FeaturedAt,
			&fav.Property.AgentID,
			&fav.Property.Version,
		)
		if err != nil {
			return nil, Metadata{}, err
		}
		fav.IsFavourited = true // Always true in this context
		favourites = append(favourites, &fav)
	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)

	return favourites, metadata, nil
}

// GetPropertyIDsForUser retrieves just the property IDs favourited by a user (lightweight)
func (m FavouriteModel) GetPropertyIDsForUser(userID int64) ([]int64, error) {
	query := `
		SELECT property_id 
		FROM user_favourites 
		WHERE user_id = $1 
		ORDER BY created_at DESC`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var propertyIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		propertyIDs = append(propertyIDs, id)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return propertyIDs, nil
}

// GetStatsForUser returns favourite statistics for a user
func (m FavouriteModel) GetStatsForUser(userID int64) (*FavouriteStats, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COALESCE(AVG(p.price), 0) as avg_price,
			COALESCE(SUM(p.price), 0) as total_value
		FROM user_favourites uf
		INNER JOIN properties p ON uf.property_id = p.id
		WHERE uf.user_id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var stats FavouriteStats

	err := m.DB.QueryRowContext(ctx, query, userID).Scan(
		&stats.TotalFavourites,
		&stats.AveragePrice,
		&stats.TotalValue,
	)

	if err != nil {
		return nil, err
	}

	return &stats, nil
}

// GetFavouriteCount returns the total number of users who favourited a property
func (m FavouriteModel) GetFavouriteCount(propertyID int64) (int, error) {
	query := `
		SELECT COUNT(*) 
		FROM user_favourites 
		WHERE property_id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var count int
	err := m.DB.QueryRowContext(ctx, query, propertyID).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// GetMostFavouritedProperties returns properties sorted by favourite count
func (m FavouriteModel) GetMostFavouritedProperties(filters Filters) ([]*Property, Metadata, error) {
	query := `
		SELECT count(*) OVER(),
		       p.id, p.created_at, p.title, p.year_built, p.area, p.bedrooms, 
		       p.bathrooms, p.floor, p.price, p.location, p.property_type, 
		       p.features, p.images, p.featured_at, p.agent_id, p.version,
		       COUNT(uf.user_id) as favourite_count
		FROM properties p
		LEFT JOIN user_favourites uf ON p.id = uf.property_id
		GROUP BY p.id
		ORDER BY favourite_count DESC, p.id DESC
		LIMIT $1 OFFSET $2`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []interface{}{filters.limit(), filters.offset()}

	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	properties := []*Property{}
	totalRecords := 0

	for rows.Next() {
		var property Property
		var favouriteCount int

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
			&favouriteCount,
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

// RemoveAllForProperty removes all favourites for a property (useful when deleting property)
func (m FavouriteModel) RemoveAllForProperty(propertyID int64) error {
	query := `DELETE FROM user_favourites WHERE property_id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, propertyID)
	return err
}

// RemoveAllForUser removes all favourites for a user (useful when deleting user)
func (m FavouriteModel) RemoveAllForUser(userID int64) error {
	query := `DELETE FROM user_favourites WHERE user_id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, userID)
	return err
}
