package firmador

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

// DefaultConfig returns default configuration for the firmador client
func DefaultConfig() *Config {
	return &Config{
		BaseURL:      viper.GetString("firmador_url"),
		Timeout:      30 * time.Second,
		RetryMax:     3,
		RetryWaitMin: 1 * time.Second,
		RetryWaitMax: 5 * time.Second,
		CheckRetry:   retryablehttp.DefaultRetryPolicy,
		Backoff:      retryablehttp.DefaultBackoff,
	}
}

// NewClient creates a new firmador client
func NewClient(cfg *Config) *Client {
	if cfg == nil {
		cfg = DefaultConfig()
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
	cfg := &Config{
		BaseURL:      viper.GetString("firmador_url"),
		Timeout:      viper.GetDuration("firmador.timeout"),
		RetryMax:     viper.GetInt("firmador.retry_max"),
		RetryWaitMin: viper.GetDuration("firmador.retry_wait_min"),
		RetryWaitMax: viper.GetDuration("firmador.retry_wait_max"),
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

// SignDTE is a type-safe wrapper that requires the DTE interface
// This validates the DTE before signing
func (c *Client) SignDTE(ctx context.Context, nit string, password string, document dte.DTE) (string, error) {
	// Validate before signing
	if err := document.Validate(); err != nil {
		return "", fmt.Errorf("DTE validation failed: %w", err)
	}

	return c.SignDocument(ctx, nit, password, document)
}

// SignDocument signs a DTE document using the firmador service
// Accepts any type that can be marshaled to JSON - provides maximum flexibility
func (c *Client) SignDocument(ctx context.Context, nit string, password string, document interface{}) (string, error) {
	// Marshal the DTE to JSON first
	dteJSON, err := json.Marshal(document)
	if err != nil {
		return "", fmt.Errorf("failed to marshal DTE: %w", err)
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
		return "", fmt.Errorf("failed to marshal sign request: %w", err)
	}

	// Create retryable HTTP request
	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Execute the request with retries
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call firmador service after retries: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("firmador service returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var signResp SignResponse
	if err := json.Unmarshal(respBody, &signResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Check response status
	if signResp.Status != "OK" {
		// Try to parse as error response
		var errResp SignResponseError
		if err := json.Unmarshal(signResp.Body, &errResp); err == nil && errResp.Codigo != "" {
			return "", &FirmadorError{
				Code:     errResp.Codigo,
				Messages: errResp.Mensaje,
			}
		}
		return "", fmt.Errorf("firmador returned error status: %s", signResp.Status)
	}

	// Extract the signed document (it's a string in the body field)
	var signedDoc string
	if err := json.Unmarshal(signResp.Body, &signedDoc); err != nil {
		return "", fmt.Errorf("failed to extract signed document: %w", err)
	}

	if signedDoc == "" {
		return "", fmt.Errorf("firmador returned empty signed document")
	}

	return signedDoc, nil
}

// FirmadorError represents an error from the firmador service
type FirmadorError struct {
	Code     string
	Messages []string
}

func (e *FirmadorError) Error() string {
	if len(e.Messages) > 0 {
		return fmt.Sprintf("firmador error [%s]: %v", e.Code, e.Messages)
	}
	return fmt.Errorf("firmador error [%s]", e.Code)
}

// IsRetryableError checks if an error is retryable
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// FirmadorError (business logic errors) are not retryable
	if _, ok := err.(*FirmadorError); ok {
		return false
	}

	// Other errors might be retryable (network issues, etc.)
	return true
}

// GetBaseURL returns the configured base URL
func (c *Client) GetBaseURL() string {
	return c.baseURL
}

// SetBaseURL updates the base URL (useful for testing)
func (c *Client) SetBaseURL(url string) {
	c.baseURL = url
}
