package models

import (
	"cuentas/internal/tools"
	"fmt"
	"time"
)

// Client represents a client in the system
type Client struct {
	ID                string    `json:"id"`
	CompanyID         string    `json:"company_id"`
	NCR               *int64    `json:"-"`
	NCRFormatted      string    `json:"ncr,omitempty"`
	NIT               *int64    `json:"-"`
	NITFormatted      string    `json:"nit,omitempty"`
	DUI               *int64    `json:"-"`
	DUIFormatted      string    `json:"dui,omitempty"`
	BusinessName      string    `json:"business_name"`
	LegalBusinessName string    `json:"legal_business_name"`
	Giro              string    `json:"giro"`
	TipoContribuyente string    `json:"tipo_contribuyente"`
	FullAddress       string    `json:"full_address"`
	CountryCode       string    `json:"country_code"`
	DepartmentCode    string    `json:"department_code"`
	MunicipalityCode  string    `json:"municipality_code"`
	Active            bool      `json:"active"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// CreateClientRequest represents the request to create a client
type CreateClientRequest struct {
	NCR               string `json:"ncr"`
	NIT               string `json:"nit"`
	DUI               string `json:"dui"`
	BusinessName      string `json:"business_name" binding:"required"`
	LegalBusinessName string `json:"legal_business_name" binding:"required"`
	Giro              string `json:"giro" binding:"required"`
	TipoContribuyente string `json:"tipo_contribuyente" binding:"required"`
	FullAddress       string `json:"full_address" binding:"required"`
	CountryCode       string `json:"country_code" binding:"required"`
	DepartmentCode    string `json:"department_code" binding:"required"`
	MunicipalityCode  string `json:"municipality_code" binding:"required"`
}

// Validate validates the create client request
func (r *CreateClientRequest) Validate() error {
	// Check if at least one identification method is provided
	hasDUI := r.DUI != ""
	hasNIT := r.NIT != ""
	hasNCR := r.NCR != ""

	// Business rules:
	// 1. If DUI is provided, NIT and NCR are optional
	// 2. If NIT is provided, NCR must also be provided (no DUI required)
	// 3. At least one identification type must be provided

	if !hasDUI && !hasNIT && !hasNCR {
		return fmt.Errorf("at least one of dui, nit, or ncr must be provided")
	}

	// If NIT is provided, NCR must also be provided
	if hasNIT && !hasNCR {
		return fmt.Errorf("ncr is required when nit is provided")
	}

	// If NCR is provided without NIT, that's an error
	if hasNCR && !hasNIT {
		return fmt.Errorf("nit is required when ncr is provided")
	}

	// Validate NCR format if provided
	if hasNCR {
		if !tools.ValidateNRC(r.NCR) {
			return fmt.Errorf("ncr must be in format XXXXX-X or XXXXXX-X (e.g., 12345-6)")
		}
	}

	// Validate NIT format if provided
	if hasNIT {
		if !tools.ValidateNIT(r.NIT) {
			return fmt.Errorf("nit must be in format XXXX-XXXXXX-XXX-X (e.g., 0614-123456-001-2)")
		}
	}

	// Validate DUI format if provided
	if hasDUI {
		if !tools.ValidateDUI(r.DUI) {
			return fmt.Errorf("dui must be in format XXXXXXXX-X (e.g., 12345678-9)")
		}
	}

	// Validate business name
	if r.BusinessName == "" {
		return fmt.Errorf("business_name is required")
	}

	// Validate legal business name
	if r.LegalBusinessName == "" {
		return fmt.Errorf("legal_business_name is required")
	}

	// Validate giro
	if r.Giro == "" {
		return fmt.Errorf("giro is required")
	}

	// Validate tipo contribuyente
	if r.TipoContribuyente == "" {
		return fmt.Errorf("tipo_contribuyente is required")
	}

	// Validate address
	if r.FullAddress == "" {
		return fmt.Errorf("full_address is required")
	}

	// Validate country code (should be 2 characters)
	if len(r.CountryCode) != 2 {
		return fmt.Errorf("country_code must be 2 characters")
	}

	// Validate department code (should be 2 characters)
	if len(r.DepartmentCode) != 2 {
		return fmt.Errorf("department_code must be 2 characters")
	}

	// Validate municipality code (should be 4 characters)
	if len(r.MunicipalityCode) != 4 {
		return fmt.Errorf("municipality_code must be 4 characters")
	}

	return nil
}
