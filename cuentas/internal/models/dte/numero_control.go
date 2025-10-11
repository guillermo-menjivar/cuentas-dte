package dte

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// NumeroControlParts represents the parsed components of a numero de control
type NumeroControlParts struct {
	Prefix            string // "DTE"
	TipoDte           string // "01", "03", etc.
	EstablishmentCode string // 8 characters (4 + 4)
	Sequence          int64  // 15 digits
}

// ValidateNumeroControlFormat validates the format of a numero de control
// Format: DTE-{2digits}-{8chars}-{15digits}
// Total length: 31 characters
func ValidateNumeroControlFormat(numeroControl string) error {
	// Check length
	if len(numeroControl) != 31 {
		return fmt.Errorf("numero control must be exactly 31 characters, got %d", len(numeroControl))
	}

	// Check pattern
	pattern := `^DTE-\d{2}-[A-Z0-9]{8}-\d{15}$`
	matched, err := regexp.MatchString(pattern, numeroControl)
	if err != nil {
		return fmt.Errorf("failed to validate pattern: %w", err)
	}
	if !matched {
		return fmt.Errorf("numero control does not match required pattern: DTE-XX-XXXXXXXX-XXXXXXXXXXXXXXX")
	}

	return nil
}

// ParseNumeroControl parses a numero de control into its components
func ParseNumeroControl(numeroControl string) (*NumeroControlParts, error) {
	// Validate format first
	if err := ValidateNumeroControlFormat(numeroControl); err != nil {
		return nil, err
	}

	// Split by dashes
	parts := strings.Split(numeroControl, "-")
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid numero control format: expected 4 parts separated by dashes")
	}

	// Parse sequence number
	sequence, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid sequence number: %w", err)
	}

	return &NumeroControlParts{
		Prefix:            parts[0], // "DTE"
		TipoDte:           parts[1], // "01", "03", etc.
		EstablishmentCode: parts[2], // 8 characters
		Sequence:          sequence, // 15 digits
	}, nil
}

// ValidateNumeroControlStructure validates that the numero control structure matches expected values
func ValidateNumeroControlStructure(
	numeroControl string,
	expectedTipoDte string,
	expectedCodEstableMH string,
	expectedCodPuntoVentaMH string,
	expectedSequence int64,
) error {
	// Parse the numero control
	parts, err := ParseNumeroControl(numeroControl)
	if err != nil {
		return fmt.Errorf("failed to parse numero control: %w", err)
	}

	// Validate tipo DTE
	if parts.TipoDte != expectedTipoDte {
		return fmt.Errorf("tipo DTE mismatch: expected %s, got %s", expectedTipoDte, parts.TipoDte)
	}

	// Validate establishment codes
	expectedEstablishmentCode := fmt.Sprintf("%s%s", expectedCodEstableMH, expectedCodPuntoVentaMH)
	if parts.EstablishmentCode != expectedEstablishmentCode {
		return fmt.Errorf("establishment code mismatch: expected %s, got %s", expectedEstablishmentCode, parts.EstablishmentCode)
	}

	// Validate sequence
	if parts.Sequence != expectedSequence {
		return fmt.Errorf("sequence mismatch: expected %d, got %d", expectedSequence, parts.Sequence)
	}

	return nil
}

// BuildNumeroControl constructs a numero de control from its components
// This ensures the numero control is always built correctly
func BuildNumeroControl(
	tipoDte string,
	codEstableMH string,
	codPuntoVentaMH string,
	sequence int64,
) (string, error) {
	// Validate components before building
	if len(tipoDte) != 2 {
		return "", fmt.Errorf("tipoDte must be exactly 2 characters, got %d", len(tipoDte))
	}

	if len(codEstableMH) != 4 {
		return "", fmt.Errorf("codEstableMH must be exactly 4 characters, got %d", len(codEstableMH))
	}

	if len(codPuntoVentaMH) != 4 {
		return "", fmt.Errorf("codPuntoVentaMH must be exactly 4 characters, got %d", len(codPuntoVentaMH))
	}

	if sequence < 0 {
		return "", fmt.Errorf("sequence must be non-negative, got %d", sequence)
	}

	if sequence > 999999999999999 { // Max 15 digits
		return "", fmt.Errorf("sequence exceeds maximum 15 digits: %d", sequence)
	}

	// Build the numero control
	numeroControl := fmt.Sprintf("DTE-%s-%s%s-%015d", tipoDte, codEstableMH, codPuntoVentaMH, sequence)

	// Validate the constructed numero control
	if err := ValidateNumeroControlFormat(numeroControl); err != nil {
		return "", fmt.Errorf("constructed numero control is invalid: %w", err)
	}

	return numeroControl, nil
}
