package tools

import (
	"regexp"
)

// ValidateNIT validates the Salvadorian NIT format: XXXX-XXXXXX-XXX-X
// where X represents digits (0-9)
// Example: 0614-123456-001-2
func ValidateNIT(nit string) bool {
	if nit == "" {
		return false
	}

	// Pattern: 4 digits - 6 digits - 3 digits - 1 digit
	pattern := `^\d{4}-\d{6}-\d{3}-\d$`
	matched, err := regexp.MatchString(pattern, nit)
	if err != nil {
		return false
	}

	return matched
}

// ValidateNRC validates the Salvadorian NCR format: XXXXXX-X
// where X represents digits (0-9)
// Example: 12345-6 or 123456-7
func ValidateNRC(ncr string) bool {
	if ncr == "" {
		return false
	}

	// Pattern: 5 or 6 digits - 1 digit
	pattern := `^\d{5,6}-\d$`
	matched, err := regexp.MatchString(pattern, ncr)
	if err != nil {
		return false
	}

	return matched
}
