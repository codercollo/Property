package data

import (
	"context"
	"database/sql"
	"time"
)

type AgentModel struct {
	DB *sql.DB
}

// DashboardStats holds comprehensive statistics for an agent's dashboard
type DashboardStats struct {
	PropertiesCount int     `json:"properties_count"`
	FeaturedCount   int     `json:"featured_count"`
	ReviewsCount    int     `json:"reviews_count"`
	AverageRating   float64 `json:"average_rating"`
	TotalRevenue    float64 `json:"total_revenue"`
	PendingReviews  int     `json:"pending_reviews"`
}

// GetDashboardStats retrieves comprehensive dashboard metrics for an agent
func (m AgentModel) GetDashboardStats(agentID int64) (*DashboardStats, error) {
	query := `
		SELECT 
			COUNT(DISTINCT p.id) as properties_count,
			COUNT(DISTINCT CASE WHEN p.featured_at IS NOT NULL THEN p.id END) as featured_count,
			COUNT(DISTINCT r.id) as reviews_count,
			COALESCE(AVG(r.rating), 0) as average_rating,
			COALESCE(SUM(pay.amount), 0) as total_revenue,
			COUNT(DISTINCT CASE WHEN r.status = 'pending' THEN r.id END) as pending_reviews
		FROM properties p
		LEFT JOIN reviews r ON p.id = r.property_id AND r.status = 'approved'
		LEFT JOIN payments pay ON p.agent_id = pay.agent_id
		WHERE p.agent_id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var stats DashboardStats

	err := m.DB.QueryRowContext(ctx, query, agentID).Scan(
		&stats.PropertiesCount,
		&stats.FeaturedCount,
		&stats.ReviewsCount,
		&stats.AverageRating,
		&stats.TotalRevenue,
		&stats.PendingReviews,
	)

	if err != nil {
		return nil, err
	}

	return &stats, nil
}
