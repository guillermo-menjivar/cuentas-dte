package tools

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidateNIT validates the Salvadorian NIT format: XXXX-XXXXXX-XXX-X
func ValidateNIT(nit string) bool {
	if nit == "" {
		return false
	}
	fmt.Println("inspecting nit", nit)
	pattern := `^\d{4}-\d{6}-\d{3}-\d$`
	matched, err := regexp.MatchString(pattern, nit)
	if err != nil {
		return false
	}
	return matched
}

// ValidateNRC validates the Salvadorian NCR format: XXXXX-X or XXXXXX-X
func ValidateNRC(ncr string) bool {
	if ncr == "" {
		return false
	}
	pattern := `^\d{5,6}-\d$`
	matched, err := regexp.MatchString(pattern, ncr)
	if err != nil {
		return false
	}
	return matched
}

// StripNIT removes dashes from NIT and returns as integer string
// Input: "0614-123456-001-2" -> Output: "06141234560012"
func StripNIT(nit string) string {
	return strings.ReplaceAll(nit, "-", "")
}

// StripNRC removes dashes from NCR and returns as integer string
// Input: "12345-6" -> Output: "123456"
func StripNRC(ncr string) string {
	return strings.ReplaceAll(ncr, "-", "")
}

// FormatNIT formats a NIT number with dashes
// Input: "06141234560012" or 6141234560012 -> Output: "0614-123456-001-2"
func FormatNIT(nit string) string {
	// Remove any existing dashes first
	nit = strings.ReplaceAll(nit, "-", "")

	// Pad with leading zeros if needed (should be 14 digits)
	if len(nit) < 14 {
		nit = fmt.Sprintf("%014s", nit)
	}

	if len(nit) != 14 {
		return nit // Return as-is if invalid length
	}

	return fmt.Sprintf("%s-%s-%s-%s", nit[0:4], nit[4:10], nit[10:13], nit[13:14])
}

// FormatNRC formats a NCR number with dashes
// Input: "123456" or 123456 -> Output: "12345-6"
func FormatNRC(ncr string) string {
	// Remove any existing dashes first
	ncr = strings.ReplaceAll(ncr, "-", "")

	if len(ncr) < 6 || len(ncr) > 7 {
		return ncr // Return as-is if invalid length
	}

	// Split at last digit
	return fmt.Sprintf("%s-%s", ncr[0:len(ncr)-1], ncr[len(ncr)-1:])
}

func ValidateDUI(dui string) bool {
	if dui == "" {
		return false
	}
	pattern := `^\d{8}-\d$`
	matched, err := regexp.MatchString(pattern, dui)
	if err != nil {
		return false
	}
	return matched
}

// StripDUI removes dashes from DUI and returns as integer string
// Input: "12345678-9" -> Output: "123456789"
func StripDUI(dui string) string {
	return strings.ReplaceAll(dui, "-", "")
}

// FormatDUI formats a DUI number with dashes
// Input: "123456789" or 123456789 -> Output: "12345678-9"
func FormatDUI(dui string) string {
	// Remove any existing dashes first
	dui = strings.ReplaceAll(dui, "-", "")

	if len(dui) != 9 {
		return dui // Return as-is if invalid length
	}

	return fmt.Sprintf("%s-%s", dui[0:8], dui[8:9])
}
