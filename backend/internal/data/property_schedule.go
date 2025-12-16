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
	ID                  int64      `json:"id"`
	PropertyID          int64      `json:"property_id"`
	UserID              int64      `json:"user_id"`
	AgentID             int64      `json:"agent_id"`
	ScheduledAt         time.Time  `json:"scheduled_at"`
	DurationMinutes     int        `json:"duration_minutes"`
	Status              string     `json:"status"`
	Notes               string     `json:"notes,omitempty"`
	RescheduleCount     int        `json:"reschedule_count"`
	OriginalScheduledAt *time.Time `json:"original_scheduled_at,omitempty"`
	LastRescheduledAt   *time.Time `json:"last_rescheduled_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	Version             int        `json:"version"`
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

// Add validation for rescheduling
func ValidateReschedule(v *validator.Validator, schedule *Schedule, newScheduledAt time.Time, newDuration int) {
	// Check reschedule limit
	v.Check(schedule.RescheduleCount < 3, "reschedule", "maximum reschedule limit (3) reached")

	// Check schedule status
	validStatuses := []string{"pending", "confirmed"}
	v.Check(validator.In(schedule.Status, validStatuses...), "status", "can only reschedule pending or confirmed appointments")

	// Check new time is in future
	v.Check(newScheduledAt.After(time.Now()), "scheduled_at", "must be in the future")

	// Check new time is different from current
	v.Check(!newScheduledAt.Equal(schedule.ScheduledAt), "scheduled_at", "new time must be different from current time")

	// Check minimum notice period (at least 2 hours before current scheduled time)
	minNotice := schedule.ScheduledAt.Add(-2 * time.Hour)
	v.Check(time.Now().Before(minNotice), "reschedule", "must reschedule at least 2 hours before appointment")

	// Validate new duration
	v.Check(newDuration > 0, "duration_minutes", "must be positive")
	v.Check(newDuration <= 480, "duration_minutes", "must not exceed 8 hours")
}

// ScheduleModel wraps database operations for schedules
type ScheduleModel struct {
	DB *sql.DB
}

// Insert creates a new schedule
// Update the Insert method to initialize reschedule fields
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

	// Insert the schedule with reschedule tracking fields
	query := `
		INSERT INTO schedules (property_id, user_id, agent_id, scheduled_at, duration_minutes, 
		                       status, notes, reschedule_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 0)
		RETURNING id, created_at, version, reschedule_count`

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
		&schedule.RescheduleCount,
	)
}

// Get retrieves a schedule by ID
// Update Get method to include reschedule fields
func (m ScheduleModel) Get(id int64) (*Schedule, error) {
	if id < 1 {
		return nil, ErrScheduleNotFound
	}

	query := `
		SELECT id, property_id, user_id, agent_id, scheduled_at, duration_minutes, 
		       status, notes, reschedule_count, original_scheduled_at, last_rescheduled_at,
		       created_at, version
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
		&schedule.RescheduleCount,
		&schedule.OriginalScheduledAt,
		&schedule.LastRescheduledAt,
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
// Update GetAllForAgent to include reschedule fields
func (m ScheduleModel) GetAllForAgent(agentID int64, status string, filters Filters) ([]*ScheduleWithDetails, Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), 
		       s.id, s.property_id, s.user_id, s.agent_id, s.scheduled_at, 
		       s.duration_minutes, s.status, s.notes, s.reschedule_count,
		       s.original_scheduled_at, s.last_rescheduled_at, s.created_at, s.version,
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
			&schedule.RescheduleCount,
			&schedule.OriginalScheduledAt,
			&schedule.LastRescheduledAt,
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
// Update GetAllForUser to include reschedule fields
func (m ScheduleModel) GetAllForUser(userID int64, filters Filters) ([]*ScheduleWithDetails, Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), 
		       s.id, s.property_id, s.user_id, s.agent_id, s.scheduled_at, 
		       s.duration_minutes, s.status, s.notes, s.reschedule_count,
		       s.original_scheduled_at, s.last_rescheduled_at, s.created_at, s.version,
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
			&schedule.RescheduleCount,
			&schedule.OriginalScheduledAt,
			&schedule.LastRescheduledAt,
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

// Reschedule updates the scheduled time and duration of an appointment
func (m ScheduleModel) Reschedule(id int64, newScheduledAt time.Time, newDuration int, version int) error {
	// First, get the current schedule
	schedule, err := m.Get(id)
	if err != nil {
		return err
	}

	// Calculate new end time
	newEndTime := newScheduledAt.Add(time.Duration(newDuration) * time.Minute)

	// Check for scheduling conflicts with the new time
	conflictQuery := `
		SELECT COUNT(*) 
		FROM schedules 
		WHERE agent_id = $1 
		AND id != $2
		AND status IN ('pending', 'confirmed')
		AND (
			(scheduled_at < $4 AND scheduled_at + make_interval(mins => duration_minutes) > $3)
		)`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var count int
	err = m.DB.QueryRowContext(ctx, conflictQuery,
		schedule.AgentID,
		id,
		newScheduledAt,
		newEndTime,
	).Scan(&count)

	if err != nil {
		return err
	}

	if count > 0 {
		return ErrScheduleConflict
	}

	// Update the schedule with new time, increment reschedule count
	query := `
		UPDATE schedules
		SET scheduled_at = $1,
		    duration_minutes = $2,
		    reschedule_count = reschedule_count + 1,
		    original_scheduled_at = COALESCE(original_scheduled_at, $3),
		    last_rescheduled_at = NOW(),
		    version = version + 1
		WHERE id = $4 AND version = $5
		RETURNING version, reschedule_count`

	var newVersion, newRescheduleCount int
	err = m.DB.QueryRowContext(ctx, query,
		newScheduledAt,
		newDuration,
		schedule.ScheduledAt, // Set original time if first reschedule
		id,
		version,
	).Scan(&newVersion, &newRescheduleCount)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrEditConflict
		}
		return err
	}

	return nil
}
