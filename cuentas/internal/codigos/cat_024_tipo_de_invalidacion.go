package codigos

import "strings"

// InvalidationType represents a type of invalidation
type InvalidationType struct {
	Code  string
	Value string
}

// Invalidation type codes
const (
	InvalidationError     = "1"
	InvalidationRescindir = "2"
	InvalidationOtro      = "3"
)

// InvalidationTypes is a map of all invalidation types
var InvalidationTypes = map[string]string{
	InvalidationError:     "Error en la Información del Documento Tributario Electrónico a invalidar.",
	InvalidationRescindir: "Rescindir de la operación realizada.",
	InvalidationOtro:      "Otro",
}

// GetInvalidationTypeName returns the name of an invalidation type by code
func GetInvalidationTypeName(code string) (string, bool) {
	name, exists := InvalidationTypes[code]
	return name, exists
}

// GetInvalidationTypeCode returns the code for an invalidation type by name (case-insensitive)
func GetInvalidationTypeCode(name string) (string, bool) {
	nameLower := strings.ToLower(strings.TrimSpace(name))

	for code, value := range InvalidationTypes {
		if strings.ToLower(value) == nameLower {
			return code, true
		}
	}
	return "", false
}

// GetAllInvalidationTypes returns a slice of all invalidation types
func GetAllInvalidationTypes() []InvalidationType {
	types := make([]InvalidationType, 0, len(InvalidationTypes))
	for code, value := range InvalidationTypes {
		types = append(types, InvalidationType{
			Code:  code,
			Value: value,
		})
	}
	return types
}

// IsValidInvalidationType checks if an invalidation type code is valid
func IsValidInvalidationType(code string) bool {
	_, exists := InvalidationTypes[code]
	return exists
}
