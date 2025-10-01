package codigoss

import "strings"

// PersonType represents a type of person
type PersonType struct {
	Code  string
	Value string
}

// Person type codes
const (
	PersonTypeNatural  = "1"
	PersonTypeJuridica = "2"
)

// PersonTypes is a map of all person types
var PersonTypes = map[string]string{
	PersonTypeNatural:  "Persona Natural",
	PersonTypeJuridica: "Persona Jurídica",
}

// GetPersonTypeName returns the name of a person type by code
func GetPersonTypeName(code string) (string, bool) {
	name, exists := PersonTypes[code]
	return name, exists
}

// GetPersonTypeCode returns the code for a person type by name (case-insensitive)
func GetPersonTypeCode(name string) (string, bool) {
	nameLower := strings.ToLower(strings.TrimSpace(name))

	for code, value := range PersonTypes {
		if strings.ToLower(value) == nameLower {
			return code, true
		}
	}
	return "", false
}

// GetAllPersonTypes returns a slice of all person types
func GetAllPersonTypes() []PersonType {
	types := make([]PersonType, 0, len(PersonTypes))
	for code, value := range PersonTypes {
		types = append(types, PersonType{
			Code:  code,
			Value: value,
		})
	}
	return types
}

// IsValidPersonType checks if a person type code is valid
func IsValidPersonType(code string) bool {
	_, exists := PersonTypes[code]
	return exists
}
