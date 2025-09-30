package codigo

import "strings"

// GenerationType represents the type of document generation
type GenerationType struct {
	Code  string
	Value string
}

// Generation type codes
const (
	GenerationPhysical   = "1"
	GenerationElectronic = "2"
)

// GenerationTypes is a map of all generation types
var GenerationTypes = map[string]string{
	GenerationPhysical:   "Físico",
	GenerationElectronic: "Electrónico",
}

// GetGenerationTypeName returns the name of a generation type by code
func GetGenerationTypeName(code string) (string, bool) {
	name, exists := GenerationTypes[code]
	return name, exists
}

// GetGenerationTypeCode returns the code for a generation type by name (case-insensitive)
func GetGenerationTypeCode(name string) (string, bool) {
	nameLower := strings.ToLower(strings.TrimSpace(name))

	for code, value := range GenerationTypes {
		if strings.ToLower(value) == nameLower {
			return code, true
		}
	}
	return "", false
}

// GetAllGenerationTypes returns a slice of all generation types
func GetAllGenerationTypes() []GenerationType {
	types := make([]GenerationType, 0, len(GenerationTypes))
	for code, value := range GenerationTypes {
		types = append(types, GenerationType{
			Code:  code,
			Value: value,
		})
	}
	return types
}

// IsValidGenerationType checks if a generation type code is valid
func IsValidGenerationType(code string) bool {
	_, exists := GenerationTypes[code]
	return exists
}
