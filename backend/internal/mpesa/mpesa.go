package mpesa

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

var (
	ErrAuthenticationFailed = errors.New("mpesa authentication failed")
	ErrSTKPushFailed        = errors.New("stk push failed")
	ErrInvalidResponse      = errors.New("invalid response from mpesa")
)

// Client represents an M-Pesa Daraja API client
type Client struct {
	ConsumerKey       string
	ConsumerSecret    string
	Passkey           string
	BusinessShortCode string
	Environment       string // "sandbox" or "production"
	httpClient        *http.Client
}

// AuthResponse represents the OAuth token response
type AuthResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in"`
}

// ErrorResponse represents an error response from M-Pesa
type ErrorResponse struct {
	RequestID    string `json:"requestId"`
	ErrorCode    string `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
}

// STKPushRequest represents the STK Push request payload
type STKPushRequest struct {
	BusinessShortCode string `json:"BusinessShortCode"`
	Password          string `json:"Password"`
	Timestamp         string `json:"Timestamp"`
	TransactionType   string `json:"TransactionType"`
	Amount            string `json:"Amount"`
	PartyA            string `json:"PartyA"`
	PartyB            string `json:"PartyB"`
	PhoneNumber       string `json:"PhoneNumber"`
	CallBackURL       string `json:"CallBackURL"`
	AccountReference  string `json:"AccountReference"`
	TransactionDesc   string `json:"TransactionDesc"`
}

// STKPushResponse represents the STK Push response
type STKPushResponse struct {
	MerchantRequestID   string `json:"MerchantRequestID"`
	CheckoutRequestID   string `json:"CheckoutRequestID"`
	ResponseCode        string `json:"ResponseCode"`
	ResponseDescription string `json:"ResponseDescription"`
	CustomerMessage     string `json:"CustomerMessage"`
}

// STKQueryRequest represents the STK Query request payload
type STKQueryRequest struct {
	BusinessShortCode string `json:"BusinessShortCode"`
	Password          string `json:"Password"`
	Timestamp         string `json:"Timestamp"`
	CheckoutRequestID string `json:"CheckoutRequestID"`
}

// STKQueryResponse represents the STK Query response
type STKQueryResponse struct {
	ResponseCode        string `json:"ResponseCode"`
	ResponseDescription string `json:"ResponseDescription"`
	MerchantRequestID   string `json:"MerchantRequestID"`
	CheckoutRequestID   string `json:"CheckoutRequestID"`
	ResultCode          string `json:"ResultCode"`
	ResultDesc          string `json:"ResultDesc"`
}

// NewClient creates a new M-Pesa client
func NewClient(consumerKey, consumerSecret, passkey, businessShortCode, environment string) *Client {
	return &Client{
		ConsumerKey:       consumerKey,
		ConsumerSecret:    consumerSecret,
		Passkey:           passkey,
		BusinessShortCode: businessShortCode,
		Environment:       environment,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// getBaseURL returns the base URL based on environment
func (c *Client) getBaseURL() string {
	if c.Environment == "production" {
		return "https://api.safaricom.co.ke"
	}
	return "https://sandbox.safaricom.co.ke"
}

// Authenticate gets an OAuth token from M-Pesa
func (c *Client) Authenticate() (string, error) {
	url := fmt.Sprintf("%s/oauth/v1/generate?grant_type=client_credentials", c.getBaseURL())

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	auth := base64.StdEncoding.EncodeToString([]byte(c.ConsumerKey + ":" + c.ConsumerSecret))
	req.Header.Set("Authorization", "Basic "+auth)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("authentication request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Try to parse error response
		var errResp ErrorResponse
		if json.Unmarshal(body, &errResp) == nil {
			return "", fmt.Errorf("%w: [%s] %s", ErrAuthenticationFailed, errResp.ErrorCode, errResp.ErrorMessage)
		}
		return "", fmt.Errorf("%w: status %d, body: %s", ErrAuthenticationFailed, resp.StatusCode, string(body))
	}

	var authResp AuthResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		return "", fmt.Errorf("failed to parse auth response: %w", err)
	}

	if authResp.AccessToken == "" {
		return "", fmt.Errorf("%w: empty access token received", ErrAuthenticationFailed)
	}

	return authResp.AccessToken, nil
}

// generatePassword generates the M-Pesa password
func (c *Client) generatePassword(timestamp string) string {
	str := c.BusinessShortCode + c.Passkey + timestamp
	return base64.StdEncoding.EncodeToString([]byte(str))
}

// InitiateSTKPush initiates an M-Pesa STK Push transaction
func (c *Client) InitiateSTKPush(phoneNumber string, amount float64, accountReference, description, callbackURL string) (*STKPushResponse, error) {
	// Validate inputs
	if c.ConsumerKey == "" || c.ConsumerSecret == "" {
		return nil, fmt.Errorf("consumer key and secret are required")
	}
	if c.Passkey == "" || c.BusinessShortCode == "" {
		return nil, fmt.Errorf("passkey and business shortcode are required")
	}

	// Get access token
	token, err := c.Authenticate()
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Generate timestamp and password
	timestamp := time.Now().Format("20060102150405")
	password := c.generatePassword(timestamp)

	// Prepare request payload
	payload := STKPushRequest{
		BusinessShortCode: c.BusinessShortCode,
		Password:          password,
		Timestamp:         timestamp,
		TransactionType:   "CustomerPayBillOnline",
		Amount:            fmt.Sprintf("%.0f", amount),
		PartyA:            phoneNumber,
		PartyB:            c.BusinessShortCode,
		PhoneNumber:       phoneNumber,
		CallBackURL:       callbackURL,
		AccountReference:  accountReference,
		TransactionDesc:   description,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make API request
	url := fmt.Sprintf("%s/mpesa/stkpush/v1/processrequest", c.getBaseURL())
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stk push request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d, body: %s", ErrSTKPushFailed, resp.StatusCode, string(body))
	}

	var stkResp STKPushResponse
	if err := json.Unmarshal(body, &stkResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check response code
	if stkResp.ResponseCode != "0" {
		return nil, fmt.Errorf("%w: %s", ErrSTKPushFailed, stkResp.ResponseDescription)
	}

	return &stkResp, nil
}

// QuerySTKPushStatus queries the status of an STK Push transaction
func (c *Client) QuerySTKPushStatus(checkoutRequestID string) (*STKQueryResponse, error) {
	// Get access token
	token, err := c.Authenticate()
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Generate timestamp and password
	timestamp := time.Now().Format("20060102150405")
	password := c.generatePassword(timestamp)

	// Prepare request payload
	payload := STKQueryRequest{
		BusinessShortCode: c.BusinessShortCode,
		Password:          password,
		Timestamp:         timestamp,
		CheckoutRequestID: checkoutRequestID,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make API request
	url := fmt.Sprintf("%s/mpesa/stkpushquery/v1/query", c.getBaseURL())
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("query request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var queryResp STKQueryResponse
	if err := json.Unmarshal(body, &queryResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &queryResp, nil
}
