package dte

import (
	"fmt"
	"regexp"
)

var (
	// Updated to include B and P for establishments
	establishmentCodeRegex = regexp.MustCompile(`^[MSBP][0-9]{3}$`)
	posCodeRegex           = regexp.MustCompile(`^P[0-9]{3}$`)
	numeroControlRegex     = regexp.MustCompile(`^DTE-[0-9]{2}-[MSBP][0-9]{3}P[0-9]{3}-[0-9]{15}$`)
)

// ValidateEstablishmentCode validates establishment code format
// Valid: M001, S001, B001, P001
func ValidateEstablishmentCode(code string) error {
	if !establishmentCodeRegex.MatchString(code) {
		return fmt.Errorf("invalid establishment code format: %s (expected M###, S###, B### or P###)", code)
	}
	return nil
}

func ValidateNumeroControl(numeroControl string) bool {
	if numeroControl == "" {
		return false
	}
	return numeroControlRegex.MatchString(numeroControl)
}
