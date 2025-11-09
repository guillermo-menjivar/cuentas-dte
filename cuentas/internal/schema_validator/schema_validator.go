package dte

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/xeipuuv/gojsonschema"
)

//go:embed schemas/*.json
var schemasFS embed.FS

// DTEValidator validates DTEs against official Hacienda JSON schemas
type DTEValidator struct {
	schemas map[string]*gojsonschema.Schema
}

// ValidationError represents a single DTE validation error
type ValidationError struct {
	Field       string      `json:"field"`
	Message     string      `json:"message"`
	Value       interface{} `json:"value,omitempty"`
	Type        string      `json:"type"`
	Description string      `json:"description"`
}

func (e ValidationError) Error() string {
	if e.Value != nil {
		return fmt.Sprintf("%s: %s (got: %v)", e.Field, e.Message, e.Value)
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationResult contains validation results
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors,omitempty"`
}

// NewDTEValidator creates a validator and loads all schemas
func NewDTEValidator() (*DTEValidator, error) {
	validator := &DTEValidator{
		schemas: make(map[string]*gojsonschema.Schema),
	}

	// Map of DTE types to schema filenames
	schemaFiles := map[string]string{
		"01": "schemas/fe-fc-v1.json",
		"03": "schemas/fe-ccf-v3.json",
		"11": "schemas/fe-fex-v1.json",
		"05": "schemas/fe-nc-v3.json",
		"06": "schemas/fe-nd-v3.json",
	}

	// Load and compile each schema
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
		log.Printf("[Validator] Loaded schema for DTE type %s", tipoDte)
	}

	log.Printf("[Validator] Successfully loaded %d schemas", len(validator.schemas))
	return validator, nil
}

// Validate validates a DTE against its type's schema
func (v *DTEValidator) Validate(tipoDte string, dteData interface{}) (*ValidationResult, error) {
	schema, exists := v.schemas[tipoDte]
	if !exists {
		return nil, fmt.Errorf("no schema found for tipo DTE: %s", tipoDte)
	}

	// Convert DTE to JSON bytes
	var dteBytes []byte
	var err error

	switch dte := dteData.(type) {
	case []byte:
		dteBytes = dte
	case string:
		dteBytes = []byte(dte)
	default:
		dteBytes, err = json.Marshal(dteData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal DTE to JSON: %w", err)
		}
	}

	// Create document loader
	documentLoader := gojsonschema.NewBytesLoader(dteBytes)

	// Validate against schema
	result, err := schema.Validate(documentLoader)
	if err != nil {
		return nil, fmt.Errorf("schema validation failed: %w", err)
	}

	// Build result
	validationResult := &ValidationResult{
		Valid:  result.Valid(),
		Errors: []ValidationError{},
	}

	if !result.Valid() {
		for _, err := range result.Errors() {
			valErr := ValidationError{
				Field:   err.Field(),
				Message: err.Description(),
				Value:   err.Value(),
				Type:    err.Type(),
			}

			// Safely extract description if it exists
			if desc, ok := err.Details()["description"].(string); ok {
				valErr.Description = desc
			}

			validationResult.Errors = append(validationResult.Errors, valErr)
		}
	}

	return validationResult, nil
}

// ValidateExportInvoice validates a Type 11 export invoice
func (v *DTEValidator) ValidateExportInvoice(dte *FacturaExportacionDTE) (*ValidationResult, error) {
	return v.Validate("11", dte)
}

// FormatValidationErrors formats validation errors into a readable string
func FormatValidationErrors(errors []ValidationError) string {
	if len(errors) == 0 {
		return ""
	}

	var lines []string
	lines = append(lines, "DTE Schema Validation Failed:")

	for i, err := range errors {
		lines = append(lines, fmt.Sprintf("  [%d] %s", i+1, err.Error()))
	}

	return strings.Join(lines, "\n")
}

// Global validator instance (initialized once)
var globalValidator *DTEValidator

// InitGlobalValidator initializes the global validator
// Call this during application startup
func InitGlobalValidator() error {
	validator, err := NewDTEValidator()
	if err != nil {
		return fmt.Errorf("failed to initialize DTE validator: %w", err)
	}
	globalValidator = validator
	return nil
}

// GetGlobalValidator returns the global validator instance
func GetGlobalValidator() *DTEValidator {
	return globalValidator
}

// MustValidate validates a DTE and panics if validation fails
// Use this during development/testing
func MustValidate(tipoDte string, dte interface{}) {
	if globalValidator == nil {
		log.Printf("WARNING: Global validator not initialized, skipping validation")
		return
	}

	result, err := globalValidator.Validate(tipoDte, dte)
	if err != nil {
		panic(fmt.Sprintf("Validation error: %v", err))
	}

	if !result.Valid {
		panic(FormatValidationErrors(result.Errors))
	}
}

// ValidateBeforeSubmission validates a DTE before submission to Hacienda
// Returns nil if valid, error with details if invalid
func ValidateBeforeSubmission(tipoDte string, dte interface{}) error {
	if globalValidator == nil {
		log.Printf("WARNING: Global validator not initialized, skipping validation")
		return nil
	}

	result, err := globalValidator.Validate(tipoDte, dte)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if !result.Valid {
		return fmt.Errorf("%s", FormatValidationErrors(result.Errors))
	}

	return nil
}
