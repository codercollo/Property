package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// =============================================================================
// PAYMENT MODEL
// =============================================================================

type Payment struct {
	ID            int64     `json:"id"`
	AgentID       int64     `json:"agent_id"`
	PropertyID    int64     `json:"property_id"`
	Amount        float64   `json:"amount"`
	PaymentMethod string    `json:"payment_method"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Version       int32     `json:"version"`
}

type PaymentModel struct {
	DB *sql.DB
}

// Create inserts a new payment record
func (m PaymentModel) Create(agentID, propertyID int64, amount float64, paymentMethod string) (*Payment, error) {
	query := `
		INSERT INTO payments (agent_id, property_id, amount, payment_method, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at, version`

	payment := &Payment{
		AgentID:       agentID,
		PropertyID:    propertyID,
		Amount:        amount,
		PaymentMethod: paymentMethod,
		Status:        "completed",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query,
		agentID, propertyID, amount, paymentMethod, payment.Status,
	).Scan(
		&payment.ID,
		&payment.CreatedAt,
		&payment.UpdatedAt,
		&payment.Version,
	)

	if err != nil {
		return nil, err
	}

	return payment, nil
}

// Get retrieves a payment by ID
func (m PaymentModel) Get(id int64) (*Payment, error) {
	if id < 1 {
		return nil, ErrPaymentNotFound
	}

	query := `
		SELECT id, agent_id, property_id, amount, payment_method, status, 
		       created_at, updated_at, version
		FROM payments
		WHERE id = $1`

	var payment Payment

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&payment.ID,
		&payment.AgentID,
		&payment.PropertyID,
		&payment.Amount,
		&payment.PaymentMethod,
		&payment.Status,
		&payment.CreatedAt,
		&payment.UpdatedAt,
		&payment.Version,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrPaymentNotFound
		default:
			return nil, err
		}
	}

	return &payment, nil
}

// GetAllForAgent retrieves all payments for a specific agent with pagination
func (m PaymentModel) GetAllForAgent(agentID int64, filters Filters) ([]*Payment, Metadata, error) {
	query := `
		SELECT count(*) OVER(), id, agent_id, property_id, amount, payment_method, 
		       status, created_at, updated_at, version
		FROM payments
		WHERE agent_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []interface{}{agentID, filters.limit(), filters.offset()}

	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	var payments []*Payment
	totalRecords := 0

	for rows.Next() {
		var payment Payment
		err := rows.Scan(
			&totalRecords,
			&payment.ID,
			&payment.AgentID,
			&payment.PropertyID,
			&payment.Amount,
			&payment.PaymentMethod,
			&payment.Status,
			&payment.CreatedAt,
			&payment.UpdatedAt,
			&payment.Version,
		)

		if err != nil {
			return nil, Metadata{}, err
		}

		payments = append(payments, &payment)
	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)

	return payments, metadata, nil
}

// GetAllForAdmin retrieves all payments with pagination for admin view
func (m PaymentModel) GetAllForAdmin(filters Filters) ([]*Payment, Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), id, agent_id, property_id, amount, payment_method, 
		       status, created_at, updated_at, version
		FROM payments
		ORDER BY %s %s, id DESC
		LIMIT $1 OFFSET $2`, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []interface{}{filters.limit(), filters.offset()}

	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	payments := []*Payment{}
	totalRecords := 0

	for rows.Next() {
		var payment Payment
		err := rows.Scan(
			&totalRecords,
			&payment.ID,
			&payment.AgentID,
			&payment.PropertyID,
			&payment.Amount,
			&payment.PaymentMethod,
			&payment.Status,
			&payment.CreatedAt,
			&payment.UpdatedAt,
			&payment.Version,
		)
		if err != nil {
			return nil, Metadata{}, err
		}
		payments = append(payments, &payment)
	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)

	return payments, metadata, nil
}
