package hacienda

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/spf13/viper"
)

// Client handles communication with the Ministerio de Hacienda API
type Client struct {
	baseURL    string
	httpClient *retryablehttp.Client
	apiToken   string // Optional: if they require authentication
}

// Config holds the hacienda client configuration
type Config struct {
	BaseURL      string
	Timeout      time.Duration
	RetryMax     int
	RetryWaitMin time.Duration
	RetryWaitMax time.Duration
	APIToken     string // Optional
}

// ReceptionRequest represents the request to submit a DTE
type ReceptionRequest struct {
	NIT         string `json:"nit"`
	Activo      bool   `json:"activo"`
	PasswordPri string `json:"passwordPri"`
	DteJson     string `json:"dteJson"` // The signed JWT
}

// ReceptionResponse represents the response from Hacienda
type ReceptionResponse struct {
	Version          int      `json:"version"`
	Ambiente         string   `json:"ambiente"`
	VersionApp       int      `json:"versionApp"`
	Estado           string   `json:"estado"` // "PROCESADO", "RECHAZADO", "RECIBIDO"
	CodigoGeneracion string   `json:"codigoGeneracion"`
	SelloRecibido    string   `json:"selloRecibido,omitempty"`
	FhProcesamiento  string   `json:"fhProcesamiento"`
	ClasificacionMsg string   `json:"clasificacionMsg,omitempty"`
	CodigoMsg        string   `json:"codigoMsg,omitempty"`
	DescripcionMsg   string   `json:"descripcionMsg,omitempty"`
	Observaciones    []string `json:"observaciones,omitempty"`
}

// HaciendaError represents an error from the Hacienda service
type HaciendaError struct {
	Type      string // "network", "validation", "rejection", "server"
	Code      string
	Message   string
	Details   interface{}
	Timestamp time.Time
}

func (e *HaciendaError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Type, e.Code, e.Message)
}

// NewClient creates a new Hacienda client
func NewClient(cfg *Config) *Client {
	if cfg == nil {
		cfg = &Config{
			// Test environment
			BaseURL:      "https://apitest.dtes.mh.gob.sv/fesv/recepciondte",
			Timeout:      60 * time.Second,
			RetryMax:     3,
			RetryWaitMin: 2 * time.Second,
			RetryWaitMax: 10 * time.Second,
		}
	}

	// Set defaults
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://apitest.dtes.mh.gob.sv/fesv/recepciondte"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}
	if cfg.RetryMax == 0 {
		cfg.RetryMax = 3
	}

	// Create retryable HTTP client
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = cfg.RetryMax
	retryClient.RetryWaitMin = cfg.RetryWaitMin
	retryClient.RetryWaitMax = cfg.RetryWaitMax
	retryClient.HTTPClient.Timeout = cfg.Timeout
	retryClient.CheckRetry = customRetryPolicy
	retryClient.Logger = nil

	return &Client{
		baseURL:    cfg.BaseURL,
		httpClient: retryClient,
		apiToken:   cfg.APIToken,
	}
}

// NewClientFromViper creates a Hacienda client from Viper config
func NewClientFromViper() *Client {
	cfg := &Config{
		BaseURL:      viper.GetString("hacienda_url"),
		Timeout:      viper.GetDuration("hacienda_timeout"),
		RetryMax:     viper.GetInt("hacienda_retry_max"),
		RetryWaitMin: viper.GetDuration("hacienda_retry_wait_min"),
		RetryWaitMax: viper.GetDuration("hacienda_retry_wait_max"),
		APIToken:     viper.GetString("hacienda_api_token"),
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

	// Retry on 5xx errors
	if resp.StatusCode >= 500 {
		return true, nil
	}

	// Retry on 429
	if resp.StatusCode == 429 {
		return true, nil
	}

	// Don't retry on 4xx errors (client errors)
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return false, nil
	}

	return false, nil
}

// SubmitDTE submits a signed DTE to Hacienda
func (c *Client) SubmitDTE(ctx context.Context, nit, password, signedJWT string) (*ReceptionResponse, error) {
	// Prepare request
	reqPayload := ReceptionRequest{
		NIT:         nit,
		Activo:      true,
		PasswordPri: password,
		DteJson:     signedJWT,
	}

	reqBody, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, &HaciendaError{
			Type:      "validation",
			Code:      "MARSHAL_ERROR",
			Message:   fmt.Sprintf("failed to marshal request: %v", err),
			Timestamp: time.Now(),
		}
	}

	// Create request
	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, &HaciendaError{
			Type:      "network",
			Code:      "REQUEST_CREATE_ERROR",
			Message:   fmt.Sprintf("failed to create request: %v", err),
			Timestamp: time.Now(),
		}
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiToken)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &HaciendaError{
			Type:      "network",
			Code:      "CONNECTION_ERROR",
			Message:   fmt.Sprintf("failed to connect to Hacienda: %v", err),
			Timestamp: time.Now(),
		}
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &HaciendaError{
			Type:      "network",
			Code:      "RESPONSE_READ_ERROR",
			Message:   fmt.Sprintf("failed to read response: %v", err),
			Timestamp: time.Now(),
		}
	}

	// Check status code
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, &HaciendaError{
			Type:      "server",
			Code:      fmt.Sprintf("HTTP_%d", resp.StatusCode),
			Message:   fmt.Sprintf("Hacienda returned status %d: %s", resp.StatusCode, string(respBody)),
			Details:   string(respBody),
			Timestamp: time.Now(),
		}
	}

	// Parse response
	var recepResp ReceptionResponse
	if err := json.Unmarshal(respBody, &recepResp); err != nil {
		return nil, &HaciendaError{
			Type:      "validation",
			Code:      "RESPONSE_PARSE_ERROR",
			Message:   fmt.Sprintf("failed to parse response: %v", err),
			Details:   string(respBody),
			Timestamp: time.Now(),
		}
	}

	// Check if DTE was rejected
	if recepResp.Estado == "RECHAZADO" {
		return &recepResp, &HaciendaError{
			Type:      "rejection",
			Code:      recepResp.CodigoMsg,
			Message:   fmt.Sprintf("DTE rejected: %s", recepResp.DescripcionMsg),
			Details:   recepResp,
			Timestamp: time.Now(),
		}
	}

	return &recepResp, nil
}

// GetBaseURL returns the configured base URL
func (c *Client) GetBaseURL() string {
	return c.baseURL
}
