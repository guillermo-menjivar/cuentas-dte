package schemavalidator

import (
	"embed"
	"fmt"
	"log"
	"strings"

	"github.com/xeipuuv/gojsonschema"
)

//go:embed schemas/*.json
var schemasFS embed.FS

// Validator validates JSON against Hacienda DTE schemas
type Validator struct {
	schemas map[string]*gojsonschema.Schema
}

// ValidationError represents a single validation error
type ValidationError struct {
	Field   string      `json:"field"`
	Message string      `json:"message"`
	Value   interface{} `json:"value,omitempty"`
	Type    string      `json:"type"`
}

func (e ValidationError) Error() string {
	if e.Value != nil {
		return fmt.Sprintf("%s: %s (got: %v)", e.Field, e.Message, e.Value)
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// NewValidator creates a validator and loads all schemas
func NewValidator() (*Validator, error) {
	validator := &Validator{
		schemas: make(map[string]*gojsonschema.Schema),
	}

	// Map DTE types to schema files
	schemaFiles := map[string]string{
		"01": "schemas/fe-fc-v1.json",  // Factura
		"03": "schemas/fe-ccf-v3.json", // Crédito Fiscal
		"05": "schemas/fe-nc-v3.json",  // Nota de Crédito
		"06": "schemas/fe-nd-v3.json",  // Nota de Débito
		"11": "schemas/fe-fex-v1.json", // Factura Exportación
	}

	// Load and compile schemas
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
		log.Printf("[SchemaValidator] Loaded schema for DTE type %s", tipoDte)
	}

	log.Printf("[SchemaValidator] Successfully loaded %d schemas", len(validator.schemas))
	return validator, nil
}

// ValidateJSON validates JSON bytes against the schema for the given DTE type
// This is the ONLY public method - pure validation, no side effects
func (v *Validator) ValidateJSON(tipoDte string, jsonBytes []byte) error {
	// Check if schema exists
	schema, exists := v.schemas[tipoDte]
	if !exists {
		return fmt.Errorf("no schema found for DTE type: %s", tipoDte)
	}

	// Load JSON document
	documentLoader := gojsonschema.NewBytesLoader(jsonBytes)

	// Validate
	result, err := schema.Validate(documentLoader)
	if err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	// If valid, return nil
	if result.Valid() {
		return nil
	}

	// Build error list
	var errors []ValidationError
	for _, err := range result.Errors() {
		errors = append(errors, ValidationError{
			Field:   err.Field(),
			Message: err.Description(),
			Value:   err.Value(),
			Type:    err.Type(),
		})
	}

	// Format and return error
	return formatValidationErrors(errors)
}

// formatValidationErrors formats errors into a readable error message
func formatValidationErrors(errors []ValidationError) error {
	if len(errors) == 0 {
		return nil
	}

	var lines []string
	lines = append(lines, "DTE Schema Validation Failed:")

	for i, err := range errors {
		lines = append(lines, fmt.Sprintf("  [%d] %s", i+1, err.Error()))
	}

	return fmt.Errorf("%s", strings.Join(lines, "\n"))
}

// Global validator instance
var globalValidator *Validator

// Init initializes the global validator (call once at startup)
// Validate validates JSON bytes using the global validator
// Returns nil if valid, error with details if invalid
func Validate(tipoDte string, jsonBytes []byte) error {
	if globalValidator == nil {
		return fmt.Errorf("schema validator not initialized - call schemavalidator.Init() first")
	}
	return globalValidator.ValidateJSON(tipoDte, jsonBytes)
}
