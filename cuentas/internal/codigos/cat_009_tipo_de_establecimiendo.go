package codigos

import "strings"

// EstablishmentType represents the type of establishment
type EstablishmentType struct {
	Code  string
	Value string
}

// Establishment type codes
const (
	EstablishmentSucursal   = "01"
	EstablishmentCasaMatriz = "02"
	EstablishmentBodega     = "04"
	EstablishmentPatio      = "07"
)

// EstablishmentTypes is a map of all establishment types
var EstablishmentTypes = map[string]string{
	EstablishmentSucursal:   "Sucursal",
	EstablishmentCasaMatriz: "Casa Matriz",
	EstablishmentBodega:     "Bodega",
	EstablishmentPatio:      "Patio",
}

// GetEstablishmentTypeName returns the name of an establishment type by code
func GetEstablishmentTypeName(code string) (string, bool) {
	name, exists := EstablishmentTypes[code]
	return name, exists
}

// GetEstablishmentTypeCode returns the code for an establishment type by name (case-insensitive)
func GetEstablishmentTypeCode(name string) (string, bool) {
	nameLower := strings.ToLower(strings.TrimSpace(name))

	for code, value := range EstablishmentTypes {
		if strings.ToLower(value) == nameLower {
			return code, true
		}
	}
	return "", false
}

// GetAllEstablishmentTypes returns a slice of all establishment types
func GetAllEstablishmentTypes() []EstablishmentType {
	types := make([]EstablishmentType, 0, len(EstablishmentTypes))
	for code, value := range EstablishmentTypes {
		types = append(types, EstablishmentType{
			Code:  code,
			Value: value,
		})
	}
	return types
}

// IsValidEstablishmentType checks if an establishment type code is valid
func IsValidEstablishmentType(code string) bool {
	_, exists := EstablishmentTypes[code]
	return exists
}
