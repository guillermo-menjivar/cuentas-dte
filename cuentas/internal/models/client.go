package models

import (
	"cuentas/internal/codigos"
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
	MunicipalityCode  string `json:"municipality_code" binding:"required"` // Just 2 digits like "23"
}

// Validate validates the create client request
func (r *CreateClientRequest) Validate() error {
	// ... (previous validation code for IDs, business name, etc.)

	// Validate country code (should be 2 characters)
	if len(r.CountryCode) != 2 {
		return fmt.Errorf("country_code must be 2 characters")
	}

	// Validate department code (should be 2 characters)
	if len(r.DepartmentCode) != 2 {
		return fmt.Errorf("department_code must be 2 characters")
	}

	// Validate that department code is valid
	if !codigos.IsValidDepartment(r.DepartmentCode) {
		return fmt.Errorf("invalid department_code: %s", r.DepartmentCode)
	}

	// Validate municipality code (should be 2 characters)
	if len(r.MunicipalityCode) != 2 {
		return fmt.Errorf("municipality_code must be 2 characters")
	}

	// Validate that municipality belongs to the specified department
	departmentName, ok := codigos.GetDepartmentName(r.DepartmentCode)
	if !ok {
		return fmt.Errorf("invalid department_code: %s", r.DepartmentCode)
	}

	municipalities, ok := codigos.GetMunicipalitiesByDepartment(departmentName)
	if !ok {
		return fmt.Errorf("no municipalities found for department: %s", departmentName)
	}

	// Check if the municipality code exists in this department
	validMunicipality := false
	for _, mun := range municipalities {
		if mun.Code == r.MunicipalityCode {
			validMunicipality = true
			break
		}
	}

	if !validMunicipality {
		munName, _ := codigos.GetMunicipalityName(r.MunicipalityCode)
		return fmt.Errorf("municipality %s (%s) does not belong to department %s (%s)",
			r.MunicipalityCode, munName, r.DepartmentCode, departmentName)
	}

	return nil
}
