package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/codercollo/property/backend/internal/data"
	"github.com/codercollo/property/backend/internal/mpesa"
	"github.com/codercollo/property/backend/internal/validator"
)

// createPaymentHandler handles payment creation for featuring properties
func (app *application) createPaymentHandler(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user
	user := app.contextGetUser(r)

	// Check if user is an agent
	if user.Role != "agent" {
		app.notPermittedResponse(w, r)
		return
	}

	// Parse request body
	var input struct {
		PropertyID       int64   `json:"property_id"`
		Amount           float64 `json:"amount"`
		PaymentMethod    string  `json:"payment_method"`
		PaymentProvider  string  `json:"payment_provider"`
		PhoneNumber      string  `json:"phone_number,omitempty"`
		AccountReference string  `json:"account_reference,omitempty"`
		TransactionDesc  string  `json:"transaction_desc,omitempty"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Verify property exists and belongs to the agent
	property, err := app.models.Properties.Get(input.PropertyID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrPropertyNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Verify ownership
	if !property.AgentID.Valid || property.AgentID.Int64 != user.ID {
		app.notPermittedResponse(w, r)
		return
	}

	// Normalize phone number for M-Pesa (remove spaces, dashes, plus signs)
	if input.PaymentProvider == "mpesa" {
		input.PhoneNumber = normalizePhoneNumber(input.PhoneNumber)
	}

	// Create payment record
	payment := &data.Payment{
		AgentID:          user.ID,
		PropertyID:       input.PropertyID,
		Amount:           input.Amount,
		PaymentMethod:    input.PaymentMethod,
		PaymentProvider:  input.PaymentProvider,
		PhoneNumber:      input.PhoneNumber,
		AccountReference: input.AccountReference,
		TransactionDesc:  input.TransactionDesc,
		Status:           "pending",
	}

	// Validate payment
	v := validator.New()
	if data.ValidatePayment(v, payment); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Handle different payment providers
	switch input.PaymentProvider {
	case "mpesa":
		err = app.processMpesaPayment(payment)
	case "bank":
		err = app.processBankPayment(payment)
	case "card":
		err = app.processCardPayment(payment)
	default:
		app.badRequestResponse(w, r, errors.New("unsupported payment provider"))
		return
	}

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Insert payment record
	err = app.models.Payments.Create(payment)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Return payment response
	err = app.writeJSON(w, http.StatusCreated, envelope{"payment": payment}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// processMpesaPayment initiates M-Pesa STK Push
func (app *application) processMpesaPayment(payment *data.Payment) error {
	// Initialize M-Pesa client
	mpesaClient := mpesa.NewClient(
		app.config.mpesa.consumerKey,
		app.config.mpesa.consumerSecret,
		app.config.mpesa.passkey,
		app.config.mpesa.shortCode,
		app.config.mpesa.environment,
	)

	// Build callback URL
	callbackURL := fmt.Sprintf("%s/v1/payments/mpesa/callback", app.config.baseURL)

	// Initiate STK Push
	response, err := mpesaClient.InitiateSTKPush(
		payment.PhoneNumber,
		payment.Amount,
		fmt.Sprintf("PROP-%d", payment.PropertyID),
		payment.TransactionDesc,
		callbackURL,
	)

	if err != nil {
		return err
	}

	// Update payment with M-Pesa response data
	payment.MerchantRequestID = response.MerchantRequestID
	payment.CheckoutRequestID = response.CheckoutRequestID
	payment.Status = "pending" // Will be updated by callback

	return nil
}

// processBankPayment handles bank payment processing
func (app *application) processBankPayment(payment *data.Payment) error {
	// For bank payments, we just mark as pending
	// The admin will manually verify and update status
	payment.Status = "pending"
	payment.TransactionDesc = fmt.Sprintf("Bank payment for property %d", payment.PropertyID)

	// Generate a reference number
	payment.TransactionID = fmt.Sprintf("BANK-%d-%d", payment.PropertyID, payment.AgentID)

	return nil
}

// processCardPayment handles card payment processing
func (app *application) processCardPayment(payment *data.Payment) error {
	// Placeholder for card payment integration (e.g., Stripe, Paystack)
	// For now, mark as pending
	payment.Status = "pending"
	payment.TransactionDesc = fmt.Sprintf("Card payment for property %d", payment.PropertyID)

	return nil
}

// mpesaCallbackHandler handles M-Pesa payment callbacks
func (app *application) mpesaCallbackHandler(w http.ResponseWriter, r *http.Request) {
	var callback struct {
		Body struct {
			StkCallback struct {
				MerchantRequestID string `json:"MerchantRequestID"`
				CheckoutRequestID string `json:"CheckoutRequestID"`
				ResultCode        int    `json:"ResultCode"`
				ResultDesc        string `json:"ResultDesc"`
				CallbackMetadata  struct {
					Item []struct {
						Name  string      `json:"Name"`
						Value interface{} `json:"Value"`
					} `json:"Item"`
				} `json:"CallbackMetadata"`
			} `json:"stkCallback"`
		} `json:"Body"`
	}

	err := app.readJSON(w, r, &callback)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Find payment by checkout request ID
	payment, err := app.models.Payments.GetByCheckoutRequestID(callback.Body.StkCallback.CheckoutRequestID)
	if err != nil {
		app.logger.PrintError(err, map[string]string{
			"checkout_request_id": callback.Body.StkCallback.CheckoutRequestID,
		})
		app.serverErrorResponse(w, r, err)
		return
	}

	// Extract transaction ID from callback metadata
	var transactionID string
	for _, item := range callback.Body.StkCallback.CallbackMetadata.Item {
		if item.Name == "MpesaReceiptNumber" {
			if val, ok := item.Value.(string); ok {
				transactionID = val
			}
		}
	}

	// Determine payment status based on result code
	status := "failed"
	if callback.Body.StkCallback.ResultCode == 0 {
		status = "completed"
	}

	// Update payment status
	err = app.models.Payments.UpdateStatus(
		payment.ID,
		status,
		transactionID,
		fmt.Sprintf("%d", callback.Body.StkCallback.ResultCode),
		callback.Body.StkCallback.ResultDesc,
		payment.Version,
	)

	if err != nil {
		app.logger.PrintError(err, map[string]string{
			"payment_id": fmt.Sprintf("%d", payment.ID),
		})
		app.serverErrorResponse(w, r, err)
		return
	}

	// If payment is successful, feature the property
	if status == "completed" {
		err = app.models.Properties.Feature(payment.PropertyID)
		if err != nil {
			app.logger.PrintError(err, map[string]string{
				"property_id": fmt.Sprintf("%d", payment.PropertyID),
			})
		}
	}

	// Respond to M-Pesa
	err = app.writeJSON(w, http.StatusOK, envelope{
		"ResultCode": 0,
		"ResultDesc": "Success",
	}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// queryPaymentStatusHandler allows checking payment status
func (app *application) queryPaymentStatusHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	if user.Role != "agent" {
		app.notPermittedResponse(w, r)
		return
	}

	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	payment, err := app.models.Payments.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrPaymentNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Verify ownership
	if payment.AgentID != user.ID {
		app.notPermittedResponse(w, r)
		return
	}

	// If M-Pesa payment is still pending, query status
	if payment.PaymentProvider == "mpesa" && payment.Status == "pending" && payment.CheckoutRequestID != "" {
		mpesaClient := mpesa.NewClient(
			app.config.mpesa.consumerKey,
			app.config.mpesa.consumerSecret,
			app.config.mpesa.passkey,
			app.config.mpesa.shortCode,
			app.config.mpesa.environment,
		)

		status, err := mpesaClient.QuerySTKPushStatus(payment.CheckoutRequestID)
		if err == nil && status.ResultCode == "0" {
			// Update payment if successful
			_ = app.models.Payments.UpdateStatus(
				payment.ID,
				"completed",
				"",
				status.ResultCode,
				status.ResultDesc,
				payment.Version,
			)

			// Feature the property
			_ = app.models.Properties.Feature(payment.PropertyID)

			// Refetch updated payment
			payment, _ = app.models.Payments.Get(id)
		}
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"payment": payment}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// normalizePhoneNumber removes formatting from phone numbers
func normalizePhoneNumber(phone string) string {
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.ReplaceAll(phone, "+", "")

	// Convert to 254 format if starts with 0
	if strings.HasPrefix(phone, "0") {
		phone = "254" + phone[1:]
	}

	return phone
}
