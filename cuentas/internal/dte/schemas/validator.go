package dte

import (
	"embed"
	"encoding/json"
	"fmt"

	"github.com/xeipuuv/gojsonschema"
)

//go:embed schemas/*.json
var schemasFS embed.FS

// DTEValidator validates DTEs against JSON schemas
type DTEValidator struct {
	schemas map[string]*gojsonschema.Schema
}

// NewDTEValidator creates a new DTE validator
func NewDTEValidator() (*DTEValidator, error) {
	validator := &DTEValidator{
		schemas: make(map[string]*gojsonschema.Schema),
	}

	// Load schemas
	schemaFiles := map[string]string{
		"01": "schemas/fe-fc-v1.json",
		"03": "schemas/fe-ccf-v3.json",
		"05": "schemas/fe-nc-v3.json",
		"06": "schemas/fe-nd-v3.json",
		"11": "schemas/fe-fex-v1.json",
	}

	for tipoDte, filename := range schemaFiles {
		schemaBytes, err := schemasFS.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to read schema %s: %w", filename, err)
		}

		schemaLoader := gojsonschema.NewBytesLoader(schemaBytes)
		schema, err := gojsonschema.NewSchema(schemaLoader)
		if err != nil {
			return nil, fmt.Errorf("failed to compile schema %s: %w", filename, err)
		}

		validator.schemas[tipoDte] = schema
	}

	return validator, nil
}

// ValidationError represents a DTE validation error
type ValidationError struct {
	Field       string
	Message     string
	Value       interface{}
	Requirement string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s (got: %v)", e.Field, e.Message, e.Value)
}

// Validate validates a DTE against its schema
func (v *DTEValidator) Validate(tipoDte string, dteJSON interface{}) ([]ValidationError, error) {
	schema, exists := v.schemas[tipoDte]
	if !exists {
		return nil, fmt.Errorf("no schema found for tipo DTE: %s", tipoDte)
	}

	// Convert DTE to JSON
	var dteBytes []byte
	var err error

	switch dte := dteJSON.(type) {
	case []byte:
		dteBytes = dte
	case string:
		dteBytes = []byte(dte)
	default:
		dteBytes, err = json.Marshal(dteJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal DTE: %w", err)
		}
	}

	documentLoader := gojsonschema.NewBytesLoader(dteBytes)

	// Validate
	result, err := schema.Validate(documentLoader)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if result.Valid() {
		return nil, nil
	}

	// Convert errors
	var validationErrors []ValidationError
	for _, err := range result.Errors() {
		validationErrors = append(validationErrors, ValidationError{
			Field:       err.Field(),
			Message:     err.Description(),
			Value:       err.Value(),
			Requirement: err.Type(),
		})
	}

	return validationErrors, nil
}

// ValidateExportInvoice validates an export invoice (Type 11)
func (v *DTEValidator) ValidateExportInvoice(dte *FacturaExportacionDTE) ([]ValidationError, error) {
	return v.Validate("11", dte)
}

// ValidateFactura validates a factura (Type 01)
func (v *DTEValidator) ValidateFactura(dte *FacturaDTE) ([]ValidationError, error) {
	return v.Validate("01", dte)
}

// ValidateCCF validates a CCF (Type 03)
func (v *DTEValidator) ValidateCCF(dte *CreditoFiscalDTE) ([]ValidationError, error) {
	return v.Validate("03", dte)
}
