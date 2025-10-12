package firmador

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"cuentas/internal/models/dte"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/spf13/viper"
)

// Client handles document signing via the firmador service
type Client struct {
	baseURL    string
	httpClient *retryablehttp.Client
}

// SignRequest represents the request payload for the firmador service
type SignRequest struct {
	NIT         string          `json:"nit"`
	Activo      bool            `json:"activo"`
	PasswordPri string          `json:"passwordPri"`
	DteJson     json.RawMessage `json:"dteJson"`
}

// SignResponse represents the response from the firmador service
type SignResponse struct {
	Status string          `json:"status"`
	Body   json.RawMessage `json:"body,omitempty"`
}

// SignResponseError represents an error sign response body
type SignResponseError struct {
	Codigo  string   `json:"codigo"`
	Mensaje []string `json:"mensaje"`
}

// Config holds the firmador client configuration
type Config struct {
	BaseURL      string
	Timeout      time.Duration
	RetryMax     int
	RetryWaitMin time.Duration
	RetryWaitMax time.Duration
	CheckRetry   retryablehttp.CheckRetry
	Backoff      retryablehttp.Backoff
}

// FirmadorError represents a structured error from the firmador service
type FirmadorError struct {
	Type      string // "network", "auth", "validation", "server"
	Code      string // e.g., "ECONNREFUSED", "401", "INVALID_DTE"
	Message   string // Human-readable message
	Timestamp time.Time
}

func (e *FirmadorError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Type, e.Code, e.Message)
}

// NewClient creates a new firmador client
func NewClient(cfg *Config) *Client {
	if cfg == nil {
		cfg = &Config{
			BaseURL:      "http://localhost:8113/firmardocumento",
			Timeout:      30 * time.Second,
			RetryMax:     3,
			RetryWaitMin: 1 * time.Second,
			RetryWaitMax: 5 * time.Second,
		}
	}

	// Set defaults for any missing config
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:8113/firmardocumento"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.RetryMax == 0 {
		cfg.RetryMax = 3
	}
	if cfg.RetryWaitMin == 0 {
		cfg.RetryWaitMin = 1 * time.Second
	}
	if cfg.RetryWaitMax == 0 {
		cfg.RetryWaitMax = 5 * time.Second
	}

	// Create retryable HTTP client
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = cfg.RetryMax
	retryClient.RetryWaitMin = cfg.RetryWaitMin
	retryClient.RetryWaitMax = cfg.RetryWaitMax
	retryClient.HTTPClient.Timeout = cfg.Timeout

	// Custom check retry function
	if cfg.CheckRetry != nil {
		retryClient.CheckRetry = cfg.CheckRetry
	} else {
		retryClient.CheckRetry = customRetryPolicy
	}

	if cfg.Backoff != nil {
		retryClient.Backoff = cfg.Backoff
	}

	// Disable default retry client logging
	retryClient.Logger = nil

	return &Client{
		baseURL:    cfg.BaseURL,
		httpClient: retryClient,
	}
}

// NewClientFromViper creates a firmador client using Viper configuration
func NewClientFromViper() *Client {
	baseURL := viper.GetString("firmador_url")

	// Ensure the URL includes the /firmardocumento path
	if !strings.HasSuffix(baseURL, "/firmardocumento/") {
		if strings.HasSuffix(baseURL, "/firmardocumento") {
			baseURL = baseURL + "/"
		} else {
			baseURL = baseURL + "/firmardocumento/"
		}
	}

	cfg := &Config{
		BaseURL:      baseURL,
		Timeout:      viper.GetDuration("firmador_timeout"),
		RetryMax:     viper.GetInt("firmador_retry_max"),
		RetryWaitMin: viper.GetDuration("firmador_retry_wait_min"),
		RetryWaitMax: viper.GetDuration("firmador_retry_wait_max"),
	}

	return NewClient(cfg)
}

// customRetryPolicy determines whether a request should be retried
func customRetryPolicy(ctx context.Context, resp *http.Response, err error) (bool, error) {
	// Always retry on connection errors
	if err != nil {
		return true, err
	}

	// Don't retry on context cancellation
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	// Retry on 5xx errors (server errors)
	if resp.StatusCode >= 500 {
		return true, nil
	}

	// Retry on 429 (Too Many Requests)
	if resp.StatusCode == 429 {
		return true, nil
	}

	// Don't retry on 4xx errors (client errors) except 429
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return false, nil
	}

	// Retry on any other non-2xx status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return true, nil
	}

	return false, nil
}

// Sign signs a document using the provided credentials
func (c *Client) Sign(ctx context.Context, nit, password string, document interface{}) (string, error) {
	// Marshal the DTE to JSON first
	dteJSON, err := json.Marshal(document)
	if err != nil {
		return "", &FirmadorError{
			Type:      "validation",
			Code:      "MARSHAL_ERROR",
			Message:   fmt.Sprintf("failed to marshal DTE: %v", err),
			Timestamp: time.Now(),
		}
	}

	// Prepare the request payload
	signReq := SignRequest{
		NIT:         nit,
		Activo:      true,
		PasswordPri: password,
		DteJson:     dteJSON,
	}

	// Marshal the request
	reqBody, err := json.Marshal(signReq)
	if err != nil {
		return "", &FirmadorError{
			Type:      "validation",
			Code:      "REQUEST_MARSHAL_ERROR",
			Message:   fmt.Sprintf("failed to marshal sign request: %v", err),
			Timestamp: time.Now(),
		}
	}

	// Create retryable HTTP request
	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", &FirmadorError{
			Type:      "network",
			Code:      "REQUEST_CREATE_ERROR",
			Message:   fmt.Sprintf("failed to create HTTP request: %v", err),
			Timestamp: time.Now(),
		}
	}

	req.Header.Set("Content-Type", "application/json")

	// Execute the request with retries
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", &FirmadorError{
			Type:      "network",
			Code:      "CONNECTION_ERROR",
			Message:   fmt.Sprintf("failed to call firmador service after retries: %v", err),
			Timestamp: time.Now(),
		}
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", &FirmadorError{
			Type:      "network",
			Code:      "RESPONSE_READ_ERROR",
			Message:   fmt.Sprintf("failed to read response body: %v", err),
			Timestamp: time.Now(),
		}
	}

	// Check HTTP status code
	if resp.StatusCode == http.StatusUnauthorized {
		return "", &FirmadorError{
			Type:      "auth",
			Code:      "UNAUTHORIZED",
			Message:   "invalid credentials",
			Timestamp: time.Now(),
		}
	}

	if resp.StatusCode >= 500 {
		fmt.Println("the server returned", resp.StatusCode)
		return "", &FirmadorError{
			Type:      "server",
			Code:      fmt.Sprintf("HTTP_%d", resp.StatusCode),
			Message:   fmt.Sprintf("server error: %s", string(respBody)),
			Timestamp: time.Now(),
		}
	}

	if resp.StatusCode != http.StatusOK {
		return "", &FirmadorError{
			Type:      "server",
			Code:      fmt.Sprintf("HTTP_%d", resp.StatusCode),
			Message:   fmt.Sprintf("firmador service returned status %d: %s", resp.StatusCode, string(respBody)),
			Timestamp: time.Now(),
		}
	}

	// Parse response
	var signResp SignResponse
	if err := json.Unmarshal(respBody, &signResp); err != nil {
		return "", &FirmadorError{
			Type:      "validation",
			Code:      "RESPONSE_PARSE_ERROR",
			Message:   fmt.Sprintf("failed to parse response: %v", err),
			Timestamp: time.Now(),
		}
	}

	// Check response status
	if signResp.Status != "OK" {
		// Try to parse as error response
		var errResp SignResponseError
		if err := json.Unmarshal(signResp.Body, &errResp); err == nil && errResp.Codigo != "" {
			return "", &FirmadorError{
				Type:      "validation",
				Code:      errResp.Codigo,
				Message:   fmt.Sprintf("%v", errResp.Mensaje),
				Timestamp: time.Now(),
			}
		}
		return "", &FirmadorError{
			Type:      "server",
			Code:      "ERROR_STATUS",
			Message:   fmt.Sprintf("firmador returned error status: %s", signResp.Status),
			Timestamp: time.Now(),
		}
	}

	// Extract the signed document (it's a string in the body field)
	var signedDoc string
	if err := json.Unmarshal(signResp.Body, &signedDoc); err != nil {
		return "", &FirmadorError{
			Type:      "validation",
			Code:      "SIGNED_DOC_PARSE_ERROR",
			Message:   fmt.Sprintf("failed to extract signed document: %v", err),
			Timestamp: time.Now(),
		}
	}

	if signedDoc == "" {
		return "", &FirmadorError{
			Type:      "validation",
			Code:      "EMPTY_SIGNED_DOC",
			Message:   "firmador returned empty signed document",
			Timestamp: time.Now(),
		}
	}

	return signedDoc, nil
}

// SignDTE is a type-safe wrapper that validates the DTE before signing
func (c *Client) SignDTE(ctx context.Context, nit, password string, document dte.DTE) (string, error) {
	// Validate before signing
	if err := document.Validate(); err != nil {
		return "", &FirmadorError{
			Type:      "validation",
			Code:      "DTE_VALIDATION_ERROR",
			Message:   fmt.Sprintf("DTE validation failed: %v", err),
			Timestamp: time.Now(),
		}
	}

	return c.Sign(ctx, nit, password, document)
}

// IsRetryableError checks if an error is retryable
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	firmErr, ok := err.(*FirmadorError)
	if !ok {
		return true // Unknown errors might be retryable
	}

	// Only network and server errors are retryable
	// Auth and validation errors are not retryable
	return firmErr.Type == "network" || firmErr.Type == "server"
}

// GetBaseURL returns the configured base URL
func (c *Client) GetBaseURL() string {
	return c.baseURL
}

// SetBaseURL updates the base URL (useful for testing)
func (c *Client) SetBaseURL(url string) {
	c.baseURL = url
}
