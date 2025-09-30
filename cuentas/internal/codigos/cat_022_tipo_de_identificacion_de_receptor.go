package codigos

import "strings"

// ReceptorDocumentType represents a type of identification document for the receptor
type ReceptorDocumentType struct {
	Code  string
	Value string
}

// Receptor document type codes
const (
	ReceptorDocNIT             = "36"
	ReceptorDocDUI             = "13"
	ReceptorDocOtro            = "37"
	ReceptorDocPasaporte       = "03"
	ReceptorDocCarnetResidente = "02"
)

// ReceptorDocumentTypes is a map of all receptor document types
var ReceptorDocumentTypes = map[string]string{
	ReceptorDocNIT:             "NIT",
	ReceptorDocDUI:             "DUI",
	ReceptorDocOtro:            "Otro",
	ReceptorDocPasaporte:       "Pasaporte",
	ReceptorDocCarnetResidente: "Carnet de Residente",
}

// GetReceptorDocumentTypeName returns the name of a receptor document type by code
func GetReceptorDocumentTypeName(code string) (string, bool) {
	name, exists := ReceptorDocumentTypes[code]
	return name, exists
}

// GetReceptorDocumentTypeCode returns the code for a receptor document type by name (case-insensitive)
func GetReceptorDocumentTypeCode(name string) (string, bool) {
	nameLower := strings.ToLower(strings.TrimSpace(name))

	for code, value := range ReceptorDocumentTypes {
		if strings.ToLower(value) == nameLower {
			return code, true
		}
	}
	return "", false
}

// GetAllReceptorDocumentTypes returns a slice of all receptor document types
func GetAllReceptorDocumentTypes() []ReceptorDocumentType {
	types := make([]ReceptorDocumentType, 0, len(ReceptorDocumentTypes))
	for code, value := range ReceptorDocumentTypes {
		types = append(types, ReceptorDocumentType{
			Code:  code,
			Value: value,
		})
	}
	return types
}

// IsValidReceptorDocumentType checks if a receptor document type code is valid
func IsValidReceptorDocumentType(code string) bool {
	_, exists := ReceptorDocumentTypes[code]
	return exists
}
