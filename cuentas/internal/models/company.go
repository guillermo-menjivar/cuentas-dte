package models

import (
	"cuentas/internal/tools"
	"fmt"
	"regexp"
	"time"
)

// Company represents a company in the system
type Company struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	CodActividad        string    `json:"cod_actividad" binding:"required"` // NEW
	NombreComercial     *string   `json:"nombre_comercial"`                 // NEW: Optional
	DTEAmbiente         string    `json:"dte_ambiente" binding:"required"`
	NIT                 int64     `json:"-"`   // Store as int, don't expose directly
	NITFormatted        string    `json:"nit"` // Expose formatted version
	NCR                 int64     `json:"-"`   // Store as int, don't expose directly
	NCRFormatted        string    `json:"ncr"` // Expose formatted version
	HCUsername          string    `json:"hc_username"`
	DescActividad       string    `json:"desc_actividad"`
	HCPasswordRef       string    `json:"-"` // Never expose in JSON
	LastActivityAt      time.Time `json:"last_activity_at"`
	FirmadorUsername    string    `json:"firmador_username"` // NEW
	FirmadorPasswordRef string    `json:"-"`
	Email               string    `json:"email"`
	Active              bool      `json:"active"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// CreateCompanyRequest represents the request to create a company
type CreateCompanyRequest struct {
	Name             string  `json:"name" binding:"required"`
	NIT              string  `json:"nit" binding:"required"` // Accept as string with dashes
	NCR              string  `json:"ncr" binding:"required"` // Accept as string with dashes
	HCUsername       string  `json:"hc_username" binding:"required"`
	HCPassword       string  `json:"hc_password" binding:"required"`
	Email            string  `json:"email" binding:"required"`
	CodActividad     string  `json:"cod_actividad" binding:"required"`
	NombreComercial  *string `json:"nombre_comercial"`
	DTEAmbiente      string  `json:"dte_ambiente" binding:"required"`
	FirmadorUsername string  `json:"firmador_username" binding:"required"` // NEW
	FirmadorPassword string  `json:"firmador_password" binding:"required"`
}

// Validate validates the create company request
func (r *CreateCompanyRequest) Validate() error {
	// Validate name
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}

	// Validate NIT format
	if r.NIT == "" {
		return fmt.Errorf("nit is required")
	}
	if !tools.ValidateNIT(r.NIT) {
		return fmt.Errorf("nit must be in format XXXX-XXXXXX-XXX-X (e.g., 0614-123456-001-2)")
	}

	// Validate NCR format
	if r.NCR == "" {
		return fmt.Errorf("ncr is required")
	}
	if !tools.ValidateNRC(r.NCR) {
		return fmt.Errorf("ncr must be in format XXXXX-X or XXXXXX-X (e.g., 12345-6)")
	}

	// Validate email format
	if r.Email == "" {
		return fmt.Errorf("email is required")
	}
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(r.Email) {
		return fmt.Errorf("email format is invalid")
	}

	// Validate username
	if r.HCUsername == "" {
		return fmt.Errorf("hc_username is required")
	}

	// Validate password
	if r.HCPassword == "" {
		return fmt.Errorf("hc_password is required")
	}

	return nil
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}
