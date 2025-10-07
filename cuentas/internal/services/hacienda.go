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

	"github.com/redis/go-redis/v9"
)

type HaciendaService struct {
	db           *sql.DB
	vaultService *VaultService
	redisClient  *redis.Client
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

const (
	tokenTTL       = 12 * time.Hour
	redisKeyPrefix = "hacienda:token:"
)

// NewHaciendaService creates a new Hacienda service instance
func NewHaciendaService(db *sql.DB, vaultService *VaultService, redisClient *redis.Client) (*HaciendaService, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}
	if vaultService == nil {
		return nil, fmt.Errorf("vault service is required")
	}
	if redisClient == nil {
		return nil, fmt.Errorf("redis client is required")
	}

	return &HaciendaService{
		db:           db,
		vaultService: vaultService,
		redisClient:  redisClient,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// getRedisKey generates the Redis key for a company's token
func (h *HaciendaService) getRedisKey(companyID string) string {
	return fmt.Sprintf("%s%s", redisKeyPrefix, companyID)
}

// GetCachedToken retrieves a token from Redis cache
func (h *HaciendaService) GetCachedToken(ctx context.Context, companyID string) (*HaciendaAuthResponse, error) {
	key := h.getRedisKey(companyID)

	val, err := h.redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil // Cache miss
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get token from cache: %v", err)
	}

	var authResponse HaciendaAuthResponse
	if err := json.Unmarshal([]byte(val), &authResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached token: %v", err)
	}

	return &authResponse, nil
}

// CacheToken stores a token in Redis with 12-hour TTL
func (h *HaciendaService) CacheToken(ctx context.Context, companyID string, authResponse *HaciendaAuthResponse) error {
	key := h.getRedisKey(companyID)

	data, err := json.Marshal(authResponse)
	if err != nil {
		return fmt.Errorf("failed to marshal token for caching: %v", err)
	}

	if err := h.redisClient.Set(ctx, key, data, tokenTTL).Err(); err != nil {
		return fmt.Errorf("failed to cache token: %v", err)
	}

	return nil
}

// InvalidateToken removes a company's token from Redis cache
func (h *HaciendaService) InvalidateToken(ctx context.Context, companyID string) error {
	key := h.getRedisKey(companyID)

	if err := h.redisClient.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to invalidate token: %v", err)
	}

	return nil
}

// AuthenticateCompany authenticates a company with the Hacienda API
// Checks Redis cache first, then authenticates with Hacienda if needed
func (h *HaciendaService) AuthenticateCompany(ctx context.Context, companyID string) (*HaciendaAuthResponse, error) {
	// Check Redis cache first
	cachedToken, err := h.GetCachedToken(ctx, companyID)
	if err != nil {
		// Log error but continue to fetch new token
		fmt.Printf("Warning: failed to get cached token: %v\n", err)
	}
	if cachedToken != nil {
		return cachedToken, nil
	}

	// Cache miss - authenticate with Hacienda
	// Retrieve company credentials from database
	var username, passwordRef string
	query := `
                SELECT hc_username, hc_password_ref
                FROM companies
                WHERE id = $1 AND active = true
        `

	err = h.db.QueryRowContext(ctx, query, companyID).Scan(&username, &passwordRef)
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
	fmt.Println("this is the password", password)

	// Make authentication request to Hacienda API
	authResponse, err := h.authenticateWithHacienda(username, password)
	if err != nil {
		return nil, fmt.Errorf("hacienda authentication failed: %v", err)
	}

	// Cache the token in Redis
	if err := h.CacheToken(ctx, companyID, authResponse); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Warning: failed to cache token: %v\n", err)
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
		fmt.Println(err)
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
		fmt.Println(authResponse.Status)
		fmt.Println(authResponse)
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
