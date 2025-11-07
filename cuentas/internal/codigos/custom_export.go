package codigos

import "strings"

// ============================================
// EXPORT DOCUMENT TYPES (codDocAsociado)
// ============================================

// ExportDocumentType represents types of documents associated with exports
type ExportDocumentType struct {
	Code  string
	Value string
}

// Export document type codes
const (
	ExportDocAutorizacion = "1"
	ExportDocAduana       = "2"
	ExportDocOtro         = "3"
	ExportDocTransporte   = "4"
)

// ExportDocumentTypes is a map of all export document types
var ExportDocumentTypes = map[string]string{
	ExportDocAutorizacion: "Autorización de exportación",
	ExportDocAduana:       "Documento aduanero",
	ExportDocOtro:         "Otro documento",
	ExportDocTransporte:   "Conocimiento de embarque / Documento de transporte",
}

// GetExportDocumentTypeName returns the name of an export document type by code
func GetExportDocumentTypeName(code string) (string, bool) {
	name, exists := ExportDocumentTypes[code]
	return name, exists
}

// GetExportDocumentTypeCode returns the code for an export document type by name (case-insensitive)
func GetExportDocumentTypeCode(name string) (string, bool) {
	nameLower := strings.ToLower(strings.TrimSpace(name))
	for code, value := range ExportDocumentTypes {
		if strings.ToLower(value) == nameLower {
			return code, true
		}
	}
	return "", false
}

// GetAllExportDocumentTypes returns a slice of all export document types
func GetAllExportDocumentTypes() []ExportDocumentType {
	types := make([]ExportDocumentType, 0, len(ExportDocumentTypes))
	for code, value := range ExportDocumentTypes {
		types = append(types, ExportDocumentType{
			Code:  code,
			Value: value,
		})
	}
	return types
}

// IsValidExportDocumentType checks if an export document type code is valid
func IsValidExportDocumentType(code string) bool {
	_, exists := ExportDocumentTypes[code]
	return exists
}

// ============================================
// MODO TRANSPORTE ADICIONAL (Type 11)
// ============================================

// Note: Type 11 uses modoTransp with codes 1-7
// Codes 1-6 are the same as general TransportTypes
// Code 7 is additional for Type 11 only

const (
	TransportFijo = "7" // Fixed transport (pipelines, cables, etc.) - Type 11 only
)

// IsValidModoTransporteExportacion checks if a transport mode is valid for Type 11
// Accepts codes 1-7 (1-6 from TransportTypes + 7 for fixed transport)
func IsValidModoTransporteExportacion(code string) bool {
	// Check general transport types (1-6)
	if IsValidTransportType(code) {
		return true
	}
	// Check Type 11 specific (7)
	return code == TransportFijo
}

// GetModoTransporteExportacionName returns the name for Type 11 transport modes
func GetModoTransporteExportacionName(code string) (string, bool) {
	// Try general transport types first
	if name, exists := GetTransportTypeName(code); exists {
		return name, true
	}
	// Check Type 11 specific
	if code == TransportFijo {
		return "TRANSPORTE FIJO", true
	}
	return "", false
}
