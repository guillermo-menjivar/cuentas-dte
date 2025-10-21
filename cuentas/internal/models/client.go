package models

import (
	"cuentas/internal/codigos"
	"cuentas/internal/tools"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	// Regex patterns
	nitPattern                  = regexp.MustCompile(`^([0-9]{14}|[0-9]{9})$`)
	codigoGeneracionPattern     = regexp.MustCompile(`^[A-F0-9]{8}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{12}$`)
	numeroControlFacturaPattern = regexp.MustCompile(`^DTE-01-[A-Z0-9]{8}-[0-9]{15}$`)
	numeroControlCCFPattern     = regexp.MustCompile(`^DTE-03-[A-Z0-9]{8}-[0-9]{15}$`)
)

// Client represents a client in the system
type Client struct {
	ID                      string    `json:"id"`
	CompanyID               string    `json:"company_id"`
	NCR                     *int64    `json:"-"`
	NCRFormatted            string    `json:"ncr,omitempty"`
	NIT                     *int64    `json:"-"`
	NITFormatted            string    `json:"nit,omitempty"`
	DUI                     *int64    `json:"-"`
	DUIFormatted            string    `json:"dui,omitempty"`
	BusinessName            string    `json:"business_name"`
	LegalBusinessName       string    `json:"legal_business_name"`
	Giro                    string    `json:"giro"`
	TipoContribuyente       string    `json:"tipo_contribuyente"`
	FullAddress             string    `json:"full_address"`
	CountryCode             string    `json:"country_code"`
	DepartmentCode          string    `json:"department_code"`
	MunicipalityCode        string    `json:"municipality_code"`
	TipoPersona             string    `json:"tipo_persona"`
	CodActividad            string    `json:"cod_actividad"`
	CodActividadDescription string    `json:"cod_actividad_description"`
	Active                  bool      `json:"active"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
	Correo                  string    `json:"correo"`
	Telefono                string    `json:"telefono"`
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
	TipoPersona       string `json:"tipo_persona" binding:"required"`
	MunicipalityCode  string `json:"municipality_code" binding:"required"`

	// CCF-specific fields
	CodActividad            *string `json:"cod_actividad"`
	CodActividadDescription string  `json:"cod_actividad_description"`
	Telefono                *string `json:"telefono"`
	Correo                  *string `json:"correo"`
}

// ValidateForCCF validates that a business client (tipo_persona="2") has all required CCF fields
// according to the Hacienda CCF schema v3
func (r *CreateClientRequest) ValidateForCCF() error {
	var errors []string

	// NIT (14 or 9 digits)
	if r.NIT == "" {
		errors = append(errors, "nit is required for CCF clients")
	} else {
		nitDigits := strings.ReplaceAll(r.NIT, "-", "")
		if matched, _ := regexp.MatchString(`^[0-9]{14}$|^[0-9]{9}$`, nitDigits); !matched {
			errors = append(errors, "nit must be 14 or 9 digits")
		}
	}

	// NCR (1-8 digits)
	if r.NCR == "" {
		errors = append(errors, "ncr is required for CCF clients")
	} else {
		ncrDigits := strings.ReplaceAll(r.NCR, "-", "")
		if matched, _ := regexp.MatchString(`^[0-9]{1,8}$`, ncrDigits); !matched {
			errors = append(errors, "ncr must be 1-8 digits")
		}
	}

	// BusinessName (1-250 chars)
	if len(r.BusinessName) == 0 || len(r.BusinessName) > 250 {
		errors = append(errors, "business_name must be between 1 and 250 characters")
	}

	// CodActividad (2-6 digits, must exist in codigos)
	if r.CodActividad == nil || *r.CodActividad == "" {
		errors = append(errors, "cod_actividad is required for CCF clients")
	} else {
		description, exists := codigos.GetEconomicActivityName(*r.CodActividad)
		if !exists {
			errors = append(errors, "cod_actividad is not a valid economic activity code")
		}

		r.CodActividadDescription = description
	}

	// Department code validation
	if r.DepartmentCode == "" {
		errors = append(errors, "department_code is required for CCF clients")
	} else {
		_, exists := codigos.GetDepartmentName(r.DepartmentCode)
		fmt.Println("this is the department you sent", r.DepartmentCode)
		if !exists {
			errors = append(errors, "invalid department code")
		}
	}

	// Municipality code validation
	if r.MunicipalityCode == "" {
		errors = append(errors, "municipality_code is required for CCF clients")
	} else {
		_, exists := codigos.GetMunicipalityName(fmt.Sprintf("%s.%s", r.DepartmentCode, r.MunicipalityCode))
		if !exists {
			errors = append(errors, "invalid municipality code")
		}
		fmt.Println(r.MunicipalityCode, "this is the municipality", exists)
	}

	// Full address required (max 200 chars per schema)
	if r.FullAddress == "" {
		errors = append(errors, "full_address is required for CCF clients")
	} else if len(r.FullAddress) > 200 {
		errors = append(errors, "full_address must not exceed 200 characters")
	}

	// Telefono (optional, but if provided: 8-30 chars)
	if r.Telefono != nil && *r.Telefono != "" {
		if len(*r.Telefono) < 8 || len(*r.Telefono) > 30 {
			errors = append(errors, "telefono must be between 8 and 30 characters")
		}
	}

	// Correo (required, email format, max 100 chars)
	if r.Correo == nil || *r.Correo == "" {
		errors = append(errors, "correo (email) is required for CCF clients")
	} else {
		if len(*r.Correo) > 100 {
			errors = append(errors, "correo must not exceed 100 characters")
		}
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
		if !emailRegex.MatchString(*r.Correo) {
			errors = append(errors, "correo must be a valid email address")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("CCF validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// Validate validates the create client request
func (r *CreateClientRequest) Validate() error {
	hasDUI := r.DUI != ""
	hasNIT := r.NIT != ""
	hasNCR := r.NCR != ""

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

	// Validate tipo persona
	if r.TipoPersona == "" {
		return fmt.Errorf("tipo_persona is required")
	}

	// Validate that tipo_persona is valid
	if !codigos.IsValidPersonType(r.TipoPersona) {
		return fmt.Errorf("invalid tipo_persona: %s (must be 1 for Persona Natural or 2 for Persona Jurídica)", r.TipoPersona)
	}

	// Validate address
	if r.FullAddress == "" {
		return fmt.Errorf("full_address is required")
	}

	// Validate country code (should be 2 characters)
	if len(r.CountryCode) != 2 {
		return fmt.Errorf("country_code must be 2 characters")
	}

	// Validate that country code is valid
	if !codigos.IsValidCountry(r.CountryCode) {
		return fmt.Errorf("invalid country_code: %s", r.CountryCode)
	}

	// Validate department code (should be 2 characters)
	if len(r.DepartmentCode) != 2 {
		return fmt.Errorf("department_code must be 2 characters")
	}

	// Validate that department code is valid
	if !codigos.IsValidDepartment(r.DepartmentCode) {
		return fmt.Errorf("invalid department_code: %s", r.DepartmentCode)
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

	if r.TipoPersona != "" && !codigos.IsValidPersonType(r.TipoPersona) {
		return fmt.Errorf("invalid tipo_persona: must be '1' (Persona Natural) or '2' (Persona Jurídica)")
	}

	return nil
}
