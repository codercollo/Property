package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/codercollo/property/backend/internal/validator"
)

// Inquiry represents a property inquiry from a potential buyer/renter
type Inquiry struct {
	ID                     int64      `json:"id"`
	PropertyID             int64      `json:"property_id"`
	UserID                 int64      `json:"user_id"`
	AgentID                int64      `json:"agent_id"`
	Name                   string     `json:"name"`
	Email                  string     `json:"email"`
	Phone                  string     `json:"phone,omitempty"`
	Message                string     `json:"message"`
	InquiryType            string     `json:"inquiry_type"`
	PreferredContactMethod string     `json:"preferred_contact_method"`
	PreferredViewingDate   *time.Time `json:"preferred_viewing_date,omitempty"`
	Status                 string     `json:"status"`
	Priority               string     `json:"priority"`
	AgentNotes             string     `json:"agent_notes,omitempty"` // Changed from sql.NullString
	RespondedAt            *time.Time `json:"responded_at,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
	Version                int32      `json:"version"`
	// Joined fields
	PropertyTitle string `json:"property_title,omitempty"`
	UserName      string `json:"user_name,omitempty"`
}

// InquiryStats holds statistics about inquiries
type InquiryStats struct {
	TotalInquiries      int     `json:"total_inquiries"`
	NewInquiries        int     `json:"new_inquiries"`
	ContactedCount      int     `json:"contacted_count"`
	ScheduledCount      int     `json:"scheduled_count"`
	ClosedCount         int     `json:"closed_count"`
	ResponseRate        float64 `json:"response_rate"`
	AverageResponseTime string  `json:"average_response_time"`
}

// ValidateInquiry checks that all fields of an Inquiry are valid
func ValidateInquiry(v *validator.Validator, inquiry *Inquiry) {
	// Validate required fields
	v.Check(inquiry.PropertyID > 0, "property_id", "must be a positive integer")
	v.Check(inquiry.Name != "", "name", "must be provided")
	v.Check(len(inquiry.Name) <= 255, "name", "must not exceed 255 characters")

	// Validate email
	ValidateEmail(v, inquiry.Email)

	// Validate phone if provided
	if inquiry.Phone != "" {
		v.Check(len(inquiry.Phone) <= 50, "phone", "must not exceed 50 characters")
	}

	// Validate message
	v.Check(inquiry.Message != "", "message", "must be provided")
	v.Check(len(inquiry.Message) >= 10, "message", "must be at least 10 characters")
	v.Check(len(inquiry.Message) <= 2000, "message", "must not exceed 2000 characters")

	// Validate inquiry type
	validTypes := []string{"general", "viewing", "purchase", "rent", "more_info"}
	v.Check(validator.In(inquiry.InquiryType, validTypes...), "inquiry_type",
		"must be one of: general, viewing, purchase, rent, more_info")

	// Validate contact method
	validMethods := []string{"email", "phone", "any"}
	v.Check(validator.In(inquiry.PreferredContactMethod, validMethods...),
		"preferred_contact_method", "must be one of: email, phone, any")

	// Validate status
	validStatuses := []string{"new", "contacted", "scheduled", "closed", "spam"}
	v.Check(validator.In(inquiry.Status, validStatuses...), "status",
		"must be one of: new, contacted, scheduled, closed, spam")

	// Validate priority
	validPriorities := []string{"low", "normal", "high", "urgent"}
	v.Check(validator.In(inquiry.Priority, validPriorities...), "priority",
		"must be one of: low, normal, high, urgent")

	// Validate viewing date if provided
	if inquiry.PreferredViewingDate != nil {
		v.Check(inquiry.PreferredViewingDate.After(time.Now()),
			"preferred_viewing_date", "must be in the future")
	}
}

// InquiryModel wraps the database connection for inquiry operations
type InquiryModel struct {
	DB *sql.DB
}

// Insert creates a new inquiry
func (m InquiryModel) Insert(inquiry *Inquiry) error {
	query := `
		INSERT INTO inquiries 
		(property_id, user_id, agent_id, name, email, phone, message, 
		 inquiry_type, preferred_contact_method, preferred_viewing_date, status, priority)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at, version`

	args := []interface{}{
		inquiry.PropertyID,
		inquiry.UserID,
		inquiry.AgentID,
		inquiry.Name,
		inquiry.Email,
		inquiry.Phone,
		inquiry.Message,
		inquiry.InquiryType,
		inquiry.PreferredContactMethod,
		inquiry.PreferredViewingDate,
		inquiry.Status,
		inquiry.Priority,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return m.DB.QueryRowContext(ctx, query, args...).Scan(
		&inquiry.ID,
		&inquiry.CreatedAt,
		&inquiry.UpdatedAt,
		&inquiry.Version,
	)
}

// Get retrieves a specific inquiry by ID with joined property and user info
func (m InquiryModel) Get(id int64) (*Inquiry, error) {
	if id < 1 {
		return nil, ErrPropertyNotFound
	}

	query := `
		SELECT i.id, i.property_id, i.user_id, i.agent_id, i.name, i.email, i.phone,
		       i.message, i.inquiry_type, i.preferred_contact_method, 
		       i.preferred_viewing_date, i.status, i.priority, 
		       COALESCE(i.agent_notes, '') as agent_notes,
		       i.responded_at, i.created_at, i.updated_at, i.version,
		       p.title as property_title, u.name as user_name
		FROM inquiries i
		INNER JOIN properties p ON i.property_id = p.id
		INNER JOIN users u ON i.user_id = u.id
		WHERE i.id = $1`

	var inquiry Inquiry

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&inquiry.ID,
		&inquiry.PropertyID,
		&inquiry.UserID,
		&inquiry.AgentID,
		&inquiry.Name,
		&inquiry.Email,
		&inquiry.Phone,
		&inquiry.Message,
		&inquiry.InquiryType,
		&inquiry.PreferredContactMethod,
		&inquiry.PreferredViewingDate,
		&inquiry.Status,
		&inquiry.Priority,
		&inquiry.AgentNotes,
		&inquiry.RespondedAt,
		&inquiry.CreatedAt,
		&inquiry.UpdatedAt,
		&inquiry.Version,
		&inquiry.PropertyTitle,
		&inquiry.UserName,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrPropertyNotFound
		default:
			return nil, err
		}
	}

	return &inquiry, nil
}

// GetAllForAgent retrieves all inquiries for a specific agent
func (m InquiryModel) GetAllForAgent(agentID int64, status string, filters Filters) ([]*Inquiry, Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), 
		       i.id, i.property_id, i.user_id, i.agent_id, i.name, i.email, i.phone,
		       i.message, i.inquiry_type, i.preferred_contact_method, 
		       i.preferred_viewing_date, i.status, i.priority, 
		       COALESCE(i.agent_notes, '') as agent_notes,
		       i.responded_at, i.created_at, i.updated_at, i.version,
		       p.title as property_title, u.name as user_name
		FROM inquiries i
		INNER JOIN properties p ON i.property_id = p.id
		INNER JOIN users u ON i.user_id = u.id
		WHERE i.agent_id = $1
		AND (i.status = $2 OR $2 = '')
		ORDER BY %s %s, i.id DESC
		LIMIT $3 OFFSET $4`, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []interface{}{agentID, status, filters.limit(), filters.offset()}

	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	inquiries := []*Inquiry{}
	totalRecords := 0

	for rows.Next() {
		var inquiry Inquiry
		err := rows.Scan(
			&totalRecords,
			&inquiry.ID,
			&inquiry.PropertyID,
			&inquiry.UserID,
			&inquiry.AgentID,
			&inquiry.Name,
			&inquiry.Email,
			&inquiry.Phone,
			&inquiry.Message,
			&inquiry.InquiryType,
			&inquiry.PreferredContactMethod,
			&inquiry.PreferredViewingDate,
			&inquiry.Status,
			&inquiry.Priority,
			&inquiry.AgentNotes,
			&inquiry.RespondedAt,
			&inquiry.CreatedAt,
			&inquiry.UpdatedAt,
			&inquiry.Version,
			&inquiry.PropertyTitle,
			&inquiry.UserName,
		)
		if err != nil {
			return nil, Metadata{}, err
		}
		inquiries = append(inquiries, &inquiry)
	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)

	return inquiries, metadata, nil
}

// GetAllForUser retrieves all inquiries made by a specific user
func (m InquiryModel) GetAllForUser(userID int64, filters Filters) ([]*Inquiry, Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), 
		       i.id, i.property_id, i.user_id, i.agent_id, i.name, i.email, i.phone,
		       i.message, i.inquiry_type, i.preferred_contact_method, 
		       i.preferred_viewing_date, i.status, i.priority, 
		       COALESCE(i.agent_notes, '') as agent_notes,
		       i.responded_at, i.created_at, i.updated_at, i.version,
		       p.title as property_title, u.name as user_name
		FROM inquiries i
		INNER JOIN properties p ON i.property_id = p.id
		INNER JOIN users u ON i.user_id = u.id
		WHERE i.user_id = $1
		ORDER BY %s %s, i.id DESC
		LIMIT $2 OFFSET $3`, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []interface{}{userID, filters.limit(), filters.offset()}

	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	inquiries := []*Inquiry{}
	totalRecords := 0

	for rows.Next() {
		var inquiry Inquiry
		err := rows.Scan(
			&totalRecords,
			&inquiry.ID,
			&inquiry.PropertyID,
			&inquiry.UserID,
			&inquiry.AgentID,
			&inquiry.Name,
			&inquiry.Email,
			&inquiry.Phone,
			&inquiry.Message,
			&inquiry.InquiryType,
			&inquiry.PreferredContactMethod,
			&inquiry.PreferredViewingDate,
			&inquiry.Status,
			&inquiry.Priority,
			&inquiry.AgentNotes,
			&inquiry.RespondedAt,
			&inquiry.CreatedAt,
			&inquiry.UpdatedAt,
			&inquiry.Version,
			&inquiry.PropertyTitle,
			&inquiry.UserName,
		)
		if err != nil {
			return nil, Metadata{}, err
		}
		inquiries = append(inquiries, &inquiry)
	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)

	return inquiries, metadata, nil
}

// Update modifies an existing inquiry
func (m InquiryModel) Update(inquiry *Inquiry) error {
	query := `
		UPDATE inquiries
		SET status = $1, priority = $2, agent_notes = NULLIF($3, ''), responded_at = $4, 
		    version = version + 1
		WHERE id = $5 AND version = $6
		RETURNING version, updated_at`

	args := []interface{}{
		inquiry.Status,
		inquiry.Priority,
		inquiry.AgentNotes,
		inquiry.RespondedAt,
		inquiry.ID,
		inquiry.Version,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&inquiry.Version, &inquiry.UpdatedAt)
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

// Delete removes an inquiry
func (m InquiryModel) Delete(id int64) error {
	if id < 1 {
		return ErrPropertyNotFound
	}

	query := `DELETE FROM inquiries WHERE id = $1`

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

// GetStatsForAgent returns inquiry statistics for a specific agent
func (m InquiryModel) GetStatsForAgent(agentID int64) (*InquiryStats, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'new' THEN 1 END) as new,
			COUNT(CASE WHEN status = 'contacted' THEN 1 END) as contacted,
			COUNT(CASE WHEN status = 'scheduled' THEN 1 END) as scheduled,
			COUNT(CASE WHEN status = 'closed' THEN 1 END) as closed,
			CASE 
				WHEN COUNT(*) > 0 THEN 
					ROUND((COUNT(CASE WHEN responded_at IS NOT NULL THEN 1 END)::numeric / COUNT(*)::numeric) * 100, 2)
				ELSE 0 
			END as response_rate,
			COALESCE(
				EXTRACT(EPOCH FROM AVG(responded_at - created_at)) / 3600, 
				0
			) as avg_response_hours
		FROM inquiries
		WHERE agent_id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var stats InquiryStats
	var avgResponseHours float64

	err := m.DB.QueryRowContext(ctx, query, agentID).Scan(
		&stats.TotalInquiries,
		&stats.NewInquiries,
		&stats.ContactedCount,
		&stats.ScheduledCount,
		&stats.ClosedCount,
		&stats.ResponseRate,
		&avgResponseHours,
	)

	if err != nil {
		return nil, err
	}

	// Format average response time
	if avgResponseHours > 0 {
		stats.AverageResponseTime = fmt.Sprintf("%.1f hours", avgResponseHours)
	} else {
		stats.AverageResponseTime = "N/A"
	}

	return &stats, nil
}

// MarkAsResponded updates the inquiry to mark it as responded
func (m InquiryModel) MarkAsResponded(id int64) error {
	query := `
		UPDATE inquiries
		SET responded_at = NOW(), status = 'contacted', version = version + 1
		WHERE id = $1 AND responded_at IS NULL
		RETURNING version`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var newVersion int32
	err := m.DB.QueryRowContext(ctx, query, id).Scan(&newVersion)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPropertyNotFound
		}
		return err
	}

	return nil
}
