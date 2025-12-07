package data

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// =============================================================================
// ADMIN MODEL EXTENSIONS
// =============================================================================

type AdminModel struct {
	DB *sql.DB
}

// PlatformStats holds comprehensive platform statistics
type PlatformStats struct {
	TotalUsers       int     `json:"total_users"`
	TotalAgents      int     `json:"total_agents"`
	TotalProperties  int     `json:"total_properties"`
	TotalReviews     int     `json:"total_reviews"`
	TotalRevenue     float64 `json:"total_revenue"`
	ActiveAgents     int     `json:"active_agents"`
	FeaturedListings int     `json:"featured_listings"`
	PendingReviews   int     `json:"pending_reviews"`
	VerifiedAgents   int     `json:"verified_agents"`
	SuspendedAgents  int     `json:"suspended_agents"`
}

// GrowthMetrics holds time-series growth data
type GrowthMetrics struct {
	Period         string         `json:"period"`
	UserGrowth     []DataPoint    `json:"user_growth"`
	AgentGrowth    []DataPoint    `json:"agent_growth"`
	PropertyGrowth []DataPoint    `json:"property_growth"`
	RevenueGrowth  []RevenuePoint `json:"revenue_growth"`
}

type DataPoint struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type RevenuePoint struct {
	Date   string  `json:"date"`
	Amount float64 `json:"amount"`
}

// GetPlatformStats retrieves comprehensive platform statistics
func (m AdminModel) GetPlatformStats() (*PlatformStats, error) {
	query := `
		SELECT 
			(SELECT COUNT(*) FROM users) as total_users,
			(SELECT COUNT(*) FROM users WHERE role = 'agent') as total_agents,
			(SELECT COUNT(*) FROM properties) as total_properties,
			(SELECT COUNT(*) FROM reviews WHERE status = 'approved') as total_reviews,
			(SELECT COALESCE(SUM(amount), 0) FROM payments) as total_revenue,
			(SELECT COUNT(*) FROM users WHERE role = 'agent' AND activated = true) as active_agents,
			(SELECT COUNT(*) FROM properties WHERE featured_at IS NOT NULL) as featured_listings,
			(SELECT COUNT(*) FROM reviews WHERE status = 'pending') as pending_reviews,
			(SELECT COUNT(*) FROM agent_profiles WHERE verified = true) as verified_agents,
			(SELECT COUNT(*) FROM agent_profiles WHERE status = 'suspended') as suspended_agents
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var stats PlatformStats

	err := m.DB.QueryRowContext(ctx, query).Scan(
		&stats.TotalUsers,
		&stats.TotalAgents,
		&stats.TotalProperties,
		&stats.TotalReviews,
		&stats.TotalRevenue,
		&stats.ActiveAgents,
		&stats.FeaturedListings,
		&stats.PendingReviews,
		&stats.VerifiedAgents,
		&stats.SuspendedAgents,
	)

	if err != nil {
		return nil, err
	}

	return &stats, nil
}

// GetGrowthMetrics retrieves time-series growth data
func (m AdminModel) GetGrowthMetrics(period string) (*GrowthMetrics, error) {
	var days int
	switch period {
	case "7d":
		days = 7
	case "30d":
		days = 30
	case "90d":
		days = 90
	case "1y":
		days = 365
	default:
		days = 30
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	metrics := &GrowthMetrics{Period: period}

	// Get user growth
	userQuery := `
		SELECT DATE(created_at) as date, COUNT(*) as count
		FROM users
		WHERE created_at >= NOW() - INTERVAL '%d days'
		GROUP BY DATE(created_at)
		ORDER BY date
	`
	userGrowth, err := m.getDataPoints(ctx, fmt.Sprintf(userQuery, days))
	if err != nil {
		return nil, err
	}
	metrics.UserGrowth = userGrowth

	// Get agent growth
	agentQuery := `
		SELECT DATE(created_at) as date, COUNT(*) as count
		FROM users
		WHERE role = 'agent' AND created_at >= NOW() - INTERVAL '%d days'
		GROUP BY DATE(created_at)
		ORDER BY date
	`
	agentGrowth, err := m.getDataPoints(ctx, fmt.Sprintf(agentQuery, days))
	if err != nil {
		return nil, err
	}
	metrics.AgentGrowth = agentGrowth

	// Get property growth
	propertyQuery := `
		SELECT DATE(created_at) as date, COUNT(*) as count
		FROM properties
		WHERE created_at >= NOW() - INTERVAL '%d days'
		GROUP BY DATE(created_at)
		ORDER BY date
	`
	propertyGrowth, err := m.getDataPoints(ctx, fmt.Sprintf(propertyQuery, days))
	if err != nil {
		return nil, err
	}
	metrics.PropertyGrowth = propertyGrowth

	// Get revenue growth
	revenueQuery := `
		SELECT DATE(created_at) as date, SUM(amount) as amount
		FROM payments
		WHERE created_at >= NOW() - INTERVAL '%d days'
		GROUP BY DATE(created_at)
		ORDER BY date
	`
	revenueGrowth, err := m.getRevenuePoints(ctx, fmt.Sprintf(revenueQuery, days))
	if err != nil {
		return nil, err
	}
	metrics.RevenueGrowth = revenueGrowth

	return metrics, nil
}

// Helper function to fetch data points
func (m AdminModel) getDataPoints(ctx context.Context, query string) ([]DataPoint, error) {
	rows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []DataPoint
	for rows.Next() {
		var point DataPoint
		var date time.Time
		err := rows.Scan(&date, &point.Count)
		if err != nil {
			return nil, err
		}
		point.Date = date.Format("2006-01-02")
		points = append(points, point)
	}

	return points, rows.Err()
}

// Helper function to fetch revenue points
func (m AdminModel) getRevenuePoints(ctx context.Context, query string) ([]RevenuePoint, error) {
	rows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []RevenuePoint
	for rows.Next() {
		var point RevenuePoint
		var date time.Time
		err := rows.Scan(&date, &point.Amount)
		if err != nil {
			return nil, err
		}
		point.Date = date.Format("2006-01-02")
		points = append(points, point)
	}

	return points, rows.Err()
}

// =============================================================================
// EXTENDED USER MODEL FOR ADMIN
// =============================================================================

// GetAll retrieves all users with filtering and pagination
func (m UserModel) GetAll(role, search string, filters Filters) ([]*User, Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), id, created_at, name, email, activated, role, version
		FROM users
		WHERE (role = $1 OR $1 = '')
		AND (name ILIKE '%%' || $2 || '%%' OR email ILIKE '%%' || $2 || '%%' OR $2 = '')
		ORDER BY %s %s, id ASC
		LIMIT $3 OFFSET $4
	`, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []interface{}{role, search, filters.limit(), filters.offset()}

	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	users := []*User{}
	totalRecords := 0

	for rows.Next() {
		var user User
		err := rows.Scan(
			&totalRecords,
			&user.ID,
			&user.CreatedAt,
			&user.Name,
			&user.Email,
			&user.Activated,
			&user.Role,
			&user.Version,
		)
		if err != nil {
			return nil, Metadata{}, err
		}
		users = append(users, &user)
	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)

	return users, metadata, nil
}

// Delete removes a user from the database
func (m UserModel) Delete(id int64) error {
	if id < 1 {
		return ErrUserNotFound
	}

	query := `DELETE FROM users WHERE id = $1`

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
		return ErrUserNotFound
	}

	return nil
}

// =============================================================================
// EXTENDED AGENT MODEL FOR ADMIN
// =============================================================================

// AgentProfile represents an agent with additional profile information
type AgentProfile struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	Email           string    `json:"email"`
	Activated       bool      `json:"activated"`
	CreatedAt       time.Time `json:"created_at"`
	Verified        bool      `json:"verified"`
	Status          string    `json:"status"`
	PropertiesCount int       `json:"properties_count"`
	TotalRevenue    float64   `json:"total_revenue"`
}

// GetAll retrieves all agents with filtering
func (m AgentModel) GetAll(status, search string, filters Filters) ([]*AgentProfile, Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), 
		       u.id, u.name, u.email, u.activated, u.created_at,
		       COALESCE(ap.verified, false) as verified,
		       COALESCE(ap.status, 'active') as status,
		       COUNT(DISTINCT p.id) as properties_count,
		       COALESCE(SUM(pay.amount), 0) as total_revenue
		FROM users u
		LEFT JOIN agent_profiles ap ON u.id = ap.user_id
		LEFT JOIN properties p ON u.id = p.agent_id
		LEFT JOIN payments pay ON u.id = pay.agent_id
		WHERE u.role = 'agent'
		AND (ap.status = $1 OR $1 = '')
		AND (u.name ILIKE '%%' || $2 || '%%' OR u.email ILIKE '%%' || $2 || '%%' OR $2 = '')
		GROUP BY u.id, u.name, u.email, u.activated, u.created_at, ap.verified, ap.status
		ORDER BY %s %s, u.id ASC
		LIMIT $3 OFFSET $4
	`, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []interface{}{status, search, filters.limit(), filters.offset()}

	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	agents := []*AgentProfile{}
	totalRecords := 0

	for rows.Next() {
		var agent AgentProfile
		err := rows.Scan(
			&totalRecords,
			&agent.ID,
			&agent.Name,
			&agent.Email,
			&agent.Activated,
			&agent.CreatedAt,
			&agent.Verified,
			&agent.Status,
			&agent.PropertiesCount,
			&agent.TotalRevenue,
		)
		if err != nil {
			return nil, Metadata{}, err
		}
		agents = append(agents, &agent)
	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)

	return agents, metadata, nil
}

// ApproveVerification marks an agent as verified
func (m AgentModel) ApproveVerification(userID int64) error {
	query := `
		INSERT INTO agent_profiles (user_id, verified, status)
		VALUES ($1, true, 'active')
		ON CONFLICT (user_id) 
		DO UPDATE SET verified = true, status = 'active'
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, userID)
	return err
}

// Suspend marks an agent as suspended
func (m AgentModel) Suspend(userID int64) error {
	query := `
		INSERT INTO agent_profiles (user_id, status)
		VALUES ($1, 'suspended')
		ON CONFLICT (user_id) 
		DO UPDATE SET status = 'suspended'
	`

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

// Activate reactivates a suspended agent
func (m AgentModel) Activate(userID int64) error {
	query := `
		INSERT INTO agent_profiles (user_id, status)
		VALUES ($1, 'active')
		ON CONFLICT (user_id) 
		DO UPDATE SET status = 'active'
	`

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

// =============================================================================
// EXTENDED PROPERTY MODEL FOR ADMIN
// =============================================================================

// GetAllAdmin retrieves all properties with admin filters
func (p PropertyModel) GetAllAdmin(agentID int64, status, propertyType string, filters Filters) ([]*Property, Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), id, created_at, title, year_built, area, bedrooms, 
		       bathrooms, floor, price, location, property_type, features, images, 
		       featured_at, COALESCE(agent_id, 0) as agent_id, version
		FROM properties
		WHERE (agent_id = $1 OR $1 = 0)
		AND (property_type ILIKE '%%' || $2 || '%%' OR $2 = '')
		AND (
			($3 = 'featured' AND featured_at IS NOT NULL) OR
			($3 = 'standard' AND featured_at IS NULL) OR
			$3 = ''
		)
		ORDER BY %s %s, id ASC
		LIMIT $4 OFFSET $5
	`, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []interface{}{agentID, propertyType, status, filters.limit(), filters.offset()}

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
