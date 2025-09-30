package codigos

import "strings"

// DonationType represents a type of donation
type DonationType struct {
	Code  string
	Value string
}

// Donation type codes
const (
	DonationEfectivo = "1"
	DonationBien     = "2"
	DonationServicio = "3"
)

// DonationTypes is a map of all donation types
var DonationTypes = map[string]string{
	DonationEfectivo: "Efectivo",
	DonationBien:     "Bien",
	DonationServicio: "Servicio",
}

// GetDonationTypeName returns the name of a donation type by code
func GetDonationTypeName(code string) (string, bool) {
	name, exists := DonationTypes[code]
	return name, exists
}

// GetDonationTypeCode returns the code for a donation type by name (case-insensitive)
func GetDonationTypeCode(name string) (string, bool) {
	nameLower := strings.ToLower(strings.TrimSpace(name))

	for code, value := range DonationTypes {
		if strings.ToLower(value) == nameLower {
			return code, true
		}
	}
	return "", false
}

// GetAllDonationTypes returns a slice of all donation types
func GetAllDonationTypes() []DonationType {
	types := make([]DonationType, 0, len(DonationTypes))
	for code, value := range DonationTypes {
		types = append(types, DonationType{
			Code:  code,
			Value: value,
		})
	}
	return types
}

// IsValidDonationType checks if a donation type code is valid
func IsValidDonationType(code string) bool {
	_, exists := DonationTypes[code]
	return exists
}
