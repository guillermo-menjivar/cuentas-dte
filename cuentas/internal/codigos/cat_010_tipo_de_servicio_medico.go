package codigo

import "strings"

// MedicalServiceType represents the type of medical service
type MedicalServiceType struct {
	Code  string
	Value string
}

// Medical service type codes
const (
	MedicalServiceCirugia         = "1"
	MedicalServiceOperacion       = "2"
	MedicalServiceTratamiento     = "3"
	MedicalServiceCirugiaISBM     = "4"
	MedicalServiceOperacionISBM   = "5"
	MedicalServiceTratamientoISBM = "6"
)

// MedicalServiceTypes is a map of all medical service types
var MedicalServiceTypes = map[string]string{
	MedicalServiceCirugia:         "Cirugía",
	MedicalServiceOperacion:       "Operación",
	MedicalServiceTratamiento:     "Tratamiento médico",
	MedicalServiceCirugiaISBM:     "Cirugía instituto salvadoreño de Bienestar Magisterial",
	MedicalServiceOperacionISBM:   "Operación Instituto Salvadoreño de Bienestar Magisterial",
	MedicalServiceTratamientoISBM: "Tratamiento médico Instituto Salvadoreño de Bienestar Magisterial",
}

// GetMedicalServiceTypeName returns the name of a medical service type by code
func GetMedicalServiceTypeName(code string) (string, bool) {
	name, exists := MedicalServiceTypes[code]
	return name, exists
}

// GetMedicalServiceTypeCode returns the code for a medical service type by name (case-insensitive)
func GetMedicalServiceTypeCode(name string) (string, bool) {
	nameLower := strings.ToLower(strings.TrimSpace(name))

	for code, value := range MedicalServiceTypes {
		if strings.ToLower(value) == nameLower {
			return code, true
		}
	}
	return "", false
}

// GetAllMedicalServiceTypes returns a slice of all medical service types
func GetAllMedicalServiceTypes() []MedicalServiceType {
	types := make([]MedicalServiceType, 0, len(MedicalServiceTypes))
	for code, value := range MedicalServiceTypes {
		types = append(types, MedicalServiceType{
			Code:  code,
			Value: value,
		})
	}
	return types
}

// IsValidMedicalServiceType checks if a medical service type code is valid
func IsValidMedicalServiceType(code string) bool {
	_, exists := MedicalServiceTypes[code]
	return exists
}
