package models

import (
	"cuentas/internal/codigos"
	"cuentas/internal/tools"
	"fmt"
	"regexp"
	"time"
)

// Company represents a company in the system
type Company struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	CodActividad         string    `json:"cod_actividad" binding:"required"`     // NEW
	NombreComercial      *string   `json:"nombre_comercial" binding: "required"` // NEW: Optional
	DTEAmbiente          string    `json:"dte_ambiente" binding:"required"`
	NIT                  int64     `json:"-"`   // Store as int, don't expose directly
	NITFormatted         string    `json:"nit"` // Expose formatted version
	NCR                  int64     `json:"-"`   // Store as int, don't expose directly
	NCRFormatted         string    `json:"ncr"` // Expose formatted version
	HCUsername           string    `json:"hc_username"`
	DescActividad        string    `json:"desc_actividad"`
	HCPasswordRef        string    `json:"-"` // Never expose in JSON
	LastActivityAt       time.Time `json:"last_activity_at"`
	FirmadorUsername     string    `json:"firmador_username"` // NEW
	FirmadorPasswordRef  string    `json:"-"`
	Email                string    `json:"email"`
	Active               bool      `json:"active"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
	Departamento         string    `json:"departamento"`
	Municipio            string    `json:"municipio"`
	ComplementoDireccion string    `json:"complemento_direccion"`
	Telefono             string    `json:"telefono"`
}

// CreateCompanyRequest represents the request to create a company
type CreateCompanyRequest struct {
	Name                 string  `json:"name" binding:"required"`
	NIT                  string  `json:"nit" binding:"required"` // Accept as string with dashes
	NCR                  string  `json:"ncr" binding:"required"` // Accept as string with dashes
	HCUsername           string  `json:"hc_username" binding:"required"`
	HCPassword           string  `json:"hc_password" binding:"required"`
	Email                string  `json:"email" binding:"required"`
	CodActividad         string  `json:"cod_actividad" binding:"required"`
	NombreComercial      *string `json:"nombre_comercial" binding:"required"`
	DTEAmbiente          string  `json:"dte_ambiente" binding:"required"`
	FirmadorUsername     string  `json:"firmador_username" binding:"required"` // NEW
	FirmadorPassword     string  `json:"firmador_password" binding:"required"`
	Departamento         string  `json:"departamento" binding:"required"`
	Municipio            string  `json:"municipio" binding:"required"`
	ComplementoDireccion string  `json:"complemento_direccion" binding:"required"`
	Telefono             string  `json:"telefono" binding:"required"`
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

	// Validate departamento
	if r.Departamento == "" {
		return fmt.Errorf("departamento is required")
	}
	if !codigos.IsValidDepartment(r.Departamento) {
		return fmt.Errorf("invalid departamento code: %s", r.Departamento)
	}

	// Validate municipio
	if r.Municipio == "" {
		return fmt.Errorf("municipio is required")
	}

	if !codigos.IsValidMunicipalityInDepartment(r.Departamento, r.Municipio) {
		return fmt.Errorf("invalid municipio code '%s' for departamento '%s'", r.Municipio, r.Departamento)
	}
	// Validate complemento_direccion

	if r.ComplementoDireccion == "" {
		return fmt.Errorf("complemento_direccion is required")
	}

	if len(r.ComplementoDireccion) < 5 {
		return fmt.Errorf("complemento_direccion must be at least 5 characters")
	}
	if len(r.ComplementoDireccion) > 200 {

		return fmt.Errorf("complemento_direccion must not exceed 200 characters")
	}

	// Validate telefono
	if r.Telefono == "" {
		return fmt.Errorf("telefono is required")
	}

	if len(r.Telefono) < 8 {
		return fmt.Errorf("telefono must be at least 8 characters")
	}

	return nil
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}
