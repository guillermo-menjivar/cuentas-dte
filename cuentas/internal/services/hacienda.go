package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type HaciendaService struct {
	db           *sql.DB
	vaultService *VaultService
	httpClient   *http.Client
}

// HaciendaAuthResponse represents the complete authentication response from Hacienda
type HaciendaAuthResponse struct {
	Status string           `json:"status"`
	Body   HaciendaAuthBody `json:"body"`
}

type HaciendaAuthBody struct {
	User      string   `json:"user"`
	Token     string   `json:"token"`
	Rol       Role     `json:"rol"`
	Roles     []string `json:"roles"`
	TokenType string   `json:"tokenType"`
}

type Role struct {
	Nombre      string  `json:"nombre"`
	Codigo      string  `json:"codigo"`
	Descripcion *string `json:"descripcion"`
}

// NewHaciendaService creates a new Hacienda service instance
func NewHaciendaService(db *sql.DB, vaultService *VaultService) (*HaciendaService, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}
	if vaultService == nil {
		return nil, fmt.Errorf("vault service is required")
	}

	return &HaciendaService{
		db:           db,
		vaultService: vaultService,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// AuthenticateCompany authenticates a company with the Hacienda API
// Returns the complete authentication response
func (h *HaciendaService) AuthenticateCompany(ctx context.Context, companyID string) (*HaciendaAuthResponse, error) {
	// Retrieve company credentials from database
	var username, passwordRef string
	query := `
                SELECT hc_username, hc_password_ref 
                FROM companies 
                WHERE id = $1 AND active = true
        `

	err := h.db.QueryRowContext(ctx, query, companyID).Scan(&username, &passwordRef)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("company not found or inactive")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve company credentials: %v", err)
	}

	// Retrieve password from Vault
	password, err := h.vaultService.GetCompanyPassword(passwordRef)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve password from vault: %v", err)
	}

	// Make authentication request to Hacienda API
	authResponse, err := h.authenticateWithHacienda(username, password)
	if err != nil {
		return nil, fmt.Errorf("hacienda authentication failed: %v", err)
	}

	return authResponse, nil
}

// authenticateWithHacienda makes the actual API call to Hacienda
func (h *HaciendaService) authenticateWithHacienda(username, password string) (*HaciendaAuthResponse, error) {
	// Prepare form data
	formData := url.Values{}
	formData.Set("user", username)
	formData.Set("pwd", password)

	// Create request
	req, err := http.NewRequest(
		"POST",
		"https://apitest.dtes.mh.gob.sv/seguridad/auth",
		strings.NewReader(formData.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "user")

	// Make request
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("authentication failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse JSON response
	var authResponse HaciendaAuthResponse
	if err := json.Unmarshal(body, &authResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	// Verify the response status
	if authResponse.Status != "OK" {
		return nil, fmt.Errorf("authentication failed: status is not OK")
	}

	return &authResponse, nil
}

// UpdateLastActivity updates the last_activity_at timestamp for a company
func (h *HaciendaService) UpdateLastActivity(ctx context.Context, companyID string) error {
	query := `
                UPDATE companies 
                SET last_activity_at = CURRENT_TIMESTAMP 
                WHERE id = $1
        `

	_, err := h.db.ExecContext(ctx, query, companyID)
	if err != nil {
		return fmt.Errorf("failed to update last activity: %v", err)
	}

	return nil
}
