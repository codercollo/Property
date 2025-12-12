package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/codercollo/property/backend/internal/validator"
)

// Schedule represents a property viewing appointment
type Schedule struct {
	ID              int64     `json:"id"`
	PropertyID      int64     `json:"property_id"`
	UserID          int64     `json:"user_id"`
	AgentID         int64     `json:"agent_id"`
	ScheduledAt     time.Time `json:"scheduled_at"`
	DurationMinutes int       `json:"duration_minutes"`
	Status          string    `json:"status"`
	Notes           string    `json:"notes,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	Version         int       `json:"version"`
}

// ScheduleWithDetails includes property and user information
type ScheduleWithDetails struct {
	Schedule
	PropertyTitle string `json:"property_title"`
	PropertyAddr  string `json:"property_address"`
	UserName      string `json:"user_name"`
	UserEmail     string `json:"user_email"`
}

// AgentScheduleStats holds statistics about an agent's schedules
type AgentScheduleStats struct {
	TotalSchedules     int `json:"total_schedules"`
	PendingSchedules   int `json:"pending_schedules"`
	ConfirmedSchedules int `json:"confirmed_schedules"`
	CompletedSchedules int `json:"completed_schedules"`
	CancelledSchedules int `json:"cancelled_schedules"`
}

var (
	ErrScheduleNotFound    = errors.New("schedule not found")
	ErrScheduleConflict    = errors.New("schedule conflict - time slot already booked")
	ErrInvalidScheduleTime = errors.New("invalid schedule time - must be in the future")
	ErrScheduleNotEditable = errors.New("schedule cannot be edited in current status")
)

// ValidateSchedule validates schedule fields
func ValidateSchedule(v *validator.Validator, schedule *Schedule) {
	v.Check(schedule.PropertyID > 0, "property_id", "must be provided")
	v.Check(schedule.UserID > 0, "user_id", "must be provided")
	v.Check(schedule.AgentID > 0, "agent_id", "must be provided")

	v.Check(!schedule.ScheduledAt.IsZero(), "scheduled_at", "must be provided")
	v.Check(schedule.ScheduledAt.After(time.Now()), "scheduled_at", "must be in the future")

	v.Check(schedule.DurationMinutes > 0, "duration_minutes", "must be positive")
	v.Check(schedule.DurationMinutes <= 480, "duration_minutes", "must not exceed 8 hours")

	validStatuses := []string{"pending", "confirmed", "cancelled", "completed"}
	v.Check(validator.In(schedule.Status, validStatuses...), "status", "must be pending, confirmed, cancelled, or completed")

	v.Check(len(schedule.Notes) <= 1000, "notes", "must not exceed 1000 characters")
}

// ScheduleModel wraps database operations for schedules
type ScheduleModel struct {
	DB *sql.DB
}

// Insert creates a new schedule
func (m ScheduleModel) Insert(schedule *Schedule) error {
	// Calculate end time in Go
	endTime := schedule.ScheduledAt.Add(time.Duration(schedule.DurationMinutes) * time.Minute)

	// Check for scheduling conflicts using make_interval
	conflictQuery := `
		SELECT COUNT(*) 
		FROM schedules 
		WHERE agent_id = $1 
		AND status IN ('pending', 'confirmed')
		AND (
			(scheduled_at < $3 AND scheduled_at + make_interval(mins => duration_minutes) > $2)
		)`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var count int
	err := m.DB.QueryRowContext(ctx, conflictQuery,
		schedule.AgentID,
		schedule.ScheduledAt,
		endTime,
	).Scan(&count)

	if err != nil {
		return err
	}

	if count > 0 {
		return ErrScheduleConflict
	}

	// Insert the schedule
	query := `
		INSERT INTO schedules (property_id, user_id, agent_id, scheduled_at, duration_minutes, status, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, version`

	args := []interface{}{
		schedule.PropertyID,
		schedule.UserID,
		schedule.AgentID,
		schedule.ScheduledAt,
		schedule.DurationMinutes,
		schedule.Status,
		schedule.Notes,
	}

	return m.DB.QueryRowContext(ctx, query, args...).Scan(
		&schedule.ID,
		&schedule.CreatedAt,
		&schedule.Version,
	)
}

// Get retrieves a schedule by ID
func (m ScheduleModel) Get(id int64) (*Schedule, error) {
	if id < 1 {
		return nil, ErrScheduleNotFound
	}

	query := `
		SELECT id, property_id, user_id, agent_id, scheduled_at, duration_minutes, 
		       status, notes, created_at, version
		FROM schedules
		WHERE id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var schedule Schedule

	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&schedule.ID,
		&schedule.PropertyID,
		&schedule.UserID,
		&schedule.AgentID,
		&schedule.ScheduledAt,
		&schedule.DurationMinutes,
		&schedule.Status,
		&schedule.Notes,
		&schedule.CreatedAt,
		&schedule.Version,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrScheduleNotFound
		}
		return nil, err
	}

	return &schedule, nil
}

// GetAllForAgent retrieves all schedules for an agent with optional filtering
func (m ScheduleModel) GetAllForAgent(agentID int64, status string, filters Filters) ([]*ScheduleWithDetails, Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), 
		       s.id, s.property_id, s.user_id, s.agent_id, s.scheduled_at, 
		       s.duration_minutes, s.status, s.notes, s.created_at, s.version,
		       p.title, p.location, u.name, u.email
		FROM schedules s
		INNER JOIN properties p ON s.property_id = p.id
		INNER JOIN users u ON s.user_id = u.id
		WHERE s.agent_id = $1
		AND (s.status = $2 OR $2 = '')
		ORDER BY %s %s, s.id ASC
		LIMIT $3 OFFSET $4`, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []interface{}{agentID, status, filters.limit(), filters.offset()}

	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	schedules := []*ScheduleWithDetails{}
	totalRecords := 0

	for rows.Next() {
		var schedule ScheduleWithDetails
		err := rows.Scan(
			&totalRecords,
			&schedule.ID,
			&schedule.PropertyID,
			&schedule.UserID,
			&schedule.AgentID,
			&schedule.ScheduledAt,
			&schedule.DurationMinutes,
			&schedule.Status,
			&schedule.Notes,
			&schedule.CreatedAt,
			&schedule.Version,
			&schedule.PropertyTitle,
			&schedule.PropertyAddr,
			&schedule.UserName,
			&schedule.UserEmail,
		)
		if err != nil {
			return nil, Metadata{}, err
		}
		schedules = append(schedules, &schedule)
	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)

	return schedules, metadata, nil
}

// GetAllForUser retrieves all schedules for a user
func (m ScheduleModel) GetAllForUser(userID int64, filters Filters) ([]*ScheduleWithDetails, Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), 
		       s.id, s.property_id, s.user_id, s.agent_id, s.scheduled_at, 
		       s.duration_minutes, s.status, s.notes, s.created_at, s.version,
		       p.title, p.location, u.name, u.email
		FROM schedules s
		INNER JOIN properties p ON s.property_id = p.id
		INNER JOIN users u ON s.agent_id = u.id
		WHERE s.user_id = $1
		ORDER BY %s %s, s.id ASC
		LIMIT $2 OFFSET $3`, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []interface{}{userID, filters.limit(), filters.offset()}

	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	schedules := []*ScheduleWithDetails{}
	totalRecords := 0

	for rows.Next() {
		var schedule ScheduleWithDetails
		err := rows.Scan(
			&totalRecords,
			&schedule.ID,
			&schedule.PropertyID,
			&schedule.UserID,
			&schedule.AgentID,
			&schedule.ScheduledAt,
			&schedule.DurationMinutes,
			&schedule.Status,
			&schedule.Notes,
			&schedule.CreatedAt,
			&schedule.Version,
			&schedule.PropertyTitle,
			&schedule.PropertyAddr,
			&schedule.UserName,
			&schedule.UserEmail,
		)
		if err != nil {
			return nil, Metadata{}, err
		}
		schedules = append(schedules, &schedule)
	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)

	return schedules, metadata, nil
}

// UpdateStatus updates the status of a schedule
func (m ScheduleModel) UpdateStatus(id int64, status string, version int) error {
	query := `
		UPDATE schedules
		SET status = $1, version = version + 1
		WHERE id = $2 AND version = $3
		RETURNING version`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var newVersion int
	err := m.DB.QueryRowContext(ctx, query, status, id, version).Scan(&newVersion)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrEditConflict
		}
		return err
	}

	return nil
}

// Delete removes a schedule
func (m ScheduleModel) Delete(id int64) error {
	if id < 1 {
		return ErrScheduleNotFound
	}

	query := `DELETE FROM schedules WHERE id = $1`

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
		return ErrScheduleNotFound
	}

	return nil
}

// GetStatsForAgent returns schedule statistics for an agent
func (m ScheduleModel) GetStatsForAgent(agentID int64) (*AgentScheduleStats, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending,
			COUNT(CASE WHEN status = 'confirmed' THEN 1 END) as confirmed,
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed,
			COUNT(CASE WHEN status = 'cancelled' THEN 1 END) as cancelled
		FROM schedules
		WHERE agent_id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var stats AgentScheduleStats

	err := m.DB.QueryRowContext(ctx, query, agentID).Scan(
		&stats.TotalSchedules,
		&stats.PendingSchedules,
		&stats.ConfirmedSchedules,
		&stats.CompletedSchedules,
		&stats.CancelledSchedules,
	)

	if err != nil {
		return nil, err
	}

	return &stats, nil
}
