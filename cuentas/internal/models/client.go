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
	NCR               int64     `json:"-"`
	NCRFormatted      string    `json:"ncr"`
	NIT               int64     `json:"-"`
	NITFormatted      string    `json:"nit"`
	DUI               int64     `json:"-"`
	DUIFormatted      string    `json:"dui"`
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
	NCR               string `json:"ncr" binding:"required"`
	NIT               string `json:"nit" binding:"required"`
	DUI               string `json:"dui" binding:"required"`
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
	// Validate NCR format
	if r.NCR == "" {
		return fmt.Errorf("ncr is required")
	}
	if !tools.ValidateNRC(r.NCR) {
		return fmt.Errorf("ncr must be in format XXXXX-X or XXXXXX-X (e.g., 12345-6)")
	}

	// Validate NIT format
	if r.NIT == "" {
		return fmt.Errorf("nit is required")
	}
	if !tools.ValidateNIT(r.NIT) {
		return fmt.Errorf("nit must be in format XXXX-XXXXXX-XXX-X (e.g., 0614-123456-001-2)")
	}

	// Validate DUI format
	if r.DUI == "" {
		return fmt.Errorf("dui is required")
	}
	if !tools.ValidateDUI(r.DUI) {
		return fmt.Errorf("dui must be in format XXXXXXXX-X (e.g., 12345678-9)")
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
