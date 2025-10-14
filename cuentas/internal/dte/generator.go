package dte

import (
	"crypto/rand"
	"fmt"
	"strings"
)

// Generator generates unique identifiers for DTEs
type Generator struct{}

// NewGenerator creates a new generator instance
func NewGenerator() *Generator {
	return &Generator{}
}

// GenerateUUID creates a UUID v4 with uppercase hex digits (A-F, 0-9)
// This is required by Hacienda for codigoGeneracion
//
// Format: XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX
// Pattern: ^[A-F0-9]{8}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{12}$
//
// Example: B8AF117B-D8B1-0E60-E6D0-1B0CA6645D68
func (g *Generator) GenerateUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		panic(fmt.Sprintf("failed to generate UUID: %v", err))
	}

	// Set version (4) and variant bits
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant 10

	uuid := fmt.Sprintf("%X-%X-%X-%X-%X",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])

	return strings.ToUpper(uuid)
}

// GenerateNumeroControl creates a properly formatted numero de control
//
// Format: DTE-{tipoDte}-M{codEstable}P{codPuntoVenta}-{sequence}
// Pattern: ^DTE-01-[A-Z0-9]{8}-[0-9]{15}$
//
// Where:
//   - tipoDte: "01" for Factura
//   - M{codEstable}: "M" + 4-digit establishment code (e.g., "M0001")
//   - P{codPuntoVenta}: "P" + 4-digit point of sale code (e.g., "P0001")
//   - sequence: 15-digit sequential number (zero-padded)
//
// Example: DTE-01-M0001P0001-000000000000001
// Total length: 31 characters
func (g *Generator) GenerateNumeroControl(tipoDte, codEstable, codPuntoVenta string, sequence int64) string {
	return fmt.Sprintf(
		"DTE-%s-M%sP%s-%015d",
		tipoDte,
		codEstable,    // Already 4 digits: "0001"
		codPuntoVenta, // Already 4 digits: "0001"
		sequence,      // Zero-padded to 15 digits
	)
}

// ValidateUUID checks if a UUID matches Hacienda's required format
func (g *Generator) ValidateUUID(uuid string) bool {
	if len(uuid) != 36 {
		return false
	}

	// Check format: XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX
	parts := strings.Split(uuid, "-")
	if len(parts) != 5 {
		return false
	}

	if len(parts[0]) != 8 || len(parts[1]) != 4 || len(parts[2]) != 4 || len(parts[3]) != 4 || len(parts[4]) != 12 {
		return false
	}

	// Check that all characters are uppercase hex (A-F, 0-9)
	validChars := "0123456789ABCDEF"
	for _, part := range parts {
		for _, char := range part {
			if !strings.ContainsRune(validChars, char) {
				return false
			}
		}
	}

	return true
}

// ValidateNumeroControl checks if a numero de control matches the required format
func (g *Generator) ValidateNumeroControl(numeroControl string) bool {
	if len(numeroControl) != 31 {
		return false
	}

	// Check format: DTE-XX-XXXXXXXXX-XXXXXXXXXXXXXXX
	parts := strings.Split(numeroControl, "-")
	if len(parts) != 4 {
		return false
	}

	// Check prefix
	if parts[0] != "DTE" {
		return false
	}

	// Check tipoDte (2 digits)
	if len(parts[1]) != 2 {
		return false
	}

	// Check establishment code (8 characters: M####P####)
	if len(parts[2]) != 8 {
		return false
	}
	if parts[2][0] != 'M' || parts[2][5] != 'P' {
		return false
	}

	// Check sequence (15 digits)
	if len(parts[3]) != 15 {
		return false
	}

	return true
}
