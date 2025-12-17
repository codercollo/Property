package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/codercollo/property/backend/internal/validator"
)

// Payment represents a payment transaction
type Payment struct {
	ID                int64     `json:"id"`
	AgentID           int64     `json:"agent_id"`
	PropertyID        int64     `json:"property_id"`
	Amount            float64   `json:"amount"`
	PaymentMethod     string    `json:"payment_method"`
	PaymentProvider   string    `json:"payment_provider"`
	TransactionID     string    `json:"transaction_id,omitempty"`
	PhoneNumber       string    `json:"phone_number,omitempty"`
	AccountReference  string    `json:"account_reference,omitempty"`
	TransactionDesc   string    `json:"transaction_desc,omitempty"`
	MerchantRequestID string    `json:"merchant_request_id,omitempty"`
	CheckoutRequestID string    `json:"checkout_request_id,omitempty"`
	ResultCode        string    `json:"result_code,omitempty"`
	ResultDesc        string    `json:"result_desc,omitempty"`
	Status            string    `json:"status"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	Version           int32     `json:"version"`
}

type PaymentModel struct {
	DB *sql.DB
}

// ValidatePayment validates payment fields
func ValidatePayment(v *validator.Validator, payment *Payment) {
	v.Check(payment.AgentID > 0, "agent_id", "must be provided")
	v.Check(payment.PropertyID > 0, "property_id", "must be provided")
	v.Check(payment.Amount > 0, "amount", "must be greater than zero")
	v.Check(payment.PaymentMethod != "", "payment_method", "must be provided")
	v.Check(payment.PaymentProvider != "", "payment_provider", "must be provided")

	validProviders := []string{"mpesa", "bank", "card"}
	v.Check(validator.In(payment.PaymentProvider, validProviders...), "payment_provider", "must be mpesa, bank, or card")

	// Provider-specific validations
	if payment.PaymentProvider == "mpesa" {
		v.Check(payment.PhoneNumber != "", "phone_number", "must be provided for M-Pesa payments")
		v.Check(len(payment.PhoneNumber) >= 10, "phone_number", "must be a valid phone number")
	}

	if payment.PaymentProvider == "bank" {
		v.Check(payment.AccountReference != "", "account_reference", "must be provided for bank payments")
	}
}

// Create inserts a new payment record
func (m PaymentModel) Create(payment *Payment) error {
	query := `
		INSERT INTO payments (
			agent_id, property_id, amount, payment_method, payment_provider,
			transaction_id, phone_number, account_reference, transaction_desc,
			merchant_request_id, checkout_request_id, status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at, version`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []interface{}{
		payment.AgentID,
		payment.PropertyID,
		payment.Amount,
		payment.PaymentMethod,
		payment.PaymentProvider,
		nullString(payment.TransactionID),
		nullString(payment.PhoneNumber),
		nullString(payment.AccountReference),
		nullString(payment.TransactionDesc),
		nullString(payment.MerchantRequestID),
		nullString(payment.CheckoutRequestID),
		payment.Status,
	}

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(
		&payment.ID,
		&payment.CreatedAt,
		&payment.UpdatedAt,
		&payment.Version,
	)

	if err != nil {
		return err
	}

	return nil
}

// Get retrieves a payment by ID
func (m PaymentModel) Get(id int64) (*Payment, error) {
	if id < 1 {
		return nil, ErrPaymentNotFound
	}

	query := `
		SELECT id, agent_id, property_id, amount, payment_method, payment_provider,
		       COALESCE(transaction_id, '') as transaction_id, 
		       COALESCE(phone_number, '') as phone_number, 
		       COALESCE(account_reference, '') as account_reference, 
		       COALESCE(transaction_desc, '') as transaction_desc,
		       COALESCE(merchant_request_id, '') as merchant_request_id, 
		       COALESCE(checkout_request_id, '') as checkout_request_id, 
		       COALESCE(result_code, '') as result_code, 
		       COALESCE(result_desc, '') as result_desc,
		       status, created_at, updated_at, version
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
		&payment.PaymentProvider,
		&payment.TransactionID,
		&payment.PhoneNumber,
		&payment.AccountReference,
		&payment.TransactionDesc,
		&payment.MerchantRequestID,
		&payment.CheckoutRequestID,
		&payment.ResultCode,
		&payment.ResultDesc,
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

// GetByCheckoutRequestID retrieves a payment by M-Pesa checkout request ID
func (m PaymentModel) GetByCheckoutRequestID(checkoutRequestID string) (*Payment, error) {
	query := `
		SELECT id, agent_id, property_id, amount, payment_method, payment_provider,
		       COALESCE(transaction_id, '') as transaction_id, 
		       COALESCE(phone_number, '') as phone_number, 
		       COALESCE(account_reference, '') as account_reference, 
		       COALESCE(transaction_desc, '') as transaction_desc,
		       COALESCE(merchant_request_id, '') as merchant_request_id, 
		       COALESCE(checkout_request_id, '') as checkout_request_id, 
		       COALESCE(result_code, '') as result_code, 
		       COALESCE(result_desc, '') as result_desc,
		       status, created_at, updated_at, version
		FROM payments
		WHERE checkout_request_id = $1`

	var payment Payment

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, checkoutRequestID).Scan(
		&payment.ID,
		&payment.AgentID,
		&payment.PropertyID,
		&payment.Amount,
		&payment.PaymentMethod,
		&payment.PaymentProvider,
		&payment.TransactionID,
		&payment.PhoneNumber,
		&payment.AccountReference,
		&payment.TransactionDesc,
		&payment.MerchantRequestID,
		&payment.CheckoutRequestID,
		&payment.ResultCode,
		&payment.ResultDesc,
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

// UpdateStatus updates payment status and related fields
func (m PaymentModel) UpdateStatus(id int64, status, transactionID, resultCode, resultDesc string, version int32) error {
	query := `
		UPDATE payments
		SET status = $1,
		    transaction_id = COALESCE(NULLIF($2, ''), transaction_id),
		    result_code = $3,
		    result_desc = $4,
		    updated_at = NOW(),
		    version = version + 1
		WHERE id = $5 AND version = $6
		RETURNING version`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var newVersion int32
	err := m.DB.QueryRowContext(ctx, query, status, transactionID, resultCode, resultDesc, id, version).Scan(&newVersion)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrEditConflict
		}
		return err
	}

	return nil
}

// GetAllForAgent retrieves all payments for a specific agent with pagination
func (m PaymentModel) GetAllForAgent(agentID int64, filters Filters) ([]*Payment, Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), 
		       id, agent_id, property_id, amount, payment_method, 
		       payment_provider, 
		       COALESCE(transaction_id, '') as transaction_id, 
		       COALESCE(phone_number, '') as phone_number, 
		       COALESCE(account_reference, '') as account_reference,
		       COALESCE(transaction_desc, '') as transaction_desc,
		       COALESCE(merchant_request_id, '') as merchant_request_id,
		       COALESCE(checkout_request_id, '') as checkout_request_id,
		       COALESCE(result_code, '') as result_code,
		       COALESCE(result_desc, '') as result_desc,
		       status, created_at, updated_at, version
		FROM payments
		WHERE agent_id = $1
		ORDER BY %s %s, id DESC
		LIMIT $2 OFFSET $3`, filters.sortColumn(), filters.sortDirection())

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
			&payment.PaymentProvider,
			&payment.TransactionID,
			&payment.PhoneNumber,
			&payment.AccountReference,
			&payment.TransactionDesc,
			&payment.MerchantRequestID,
			&payment.CheckoutRequestID,
			&payment.ResultCode,
			&payment.ResultDesc,
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

// Helper function to convert empty strings to NULL
func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
