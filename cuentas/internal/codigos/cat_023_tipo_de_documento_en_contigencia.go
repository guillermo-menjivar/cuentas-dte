package codigos

import "strings"

// ContingencyDocumentType represents a type of document in contingency
type ContingencyDocumentType struct {
	Code  string
	Value string
}

// Contingency document type codes
const (
	ContingencyDocFactura               = "01"
	ContingencyDocComprobanteCredito    = "03"
	ContingencyDocNotaRemision          = "04"
	ContingencyDocNotaCredito           = "05"
	ContingencyDocNotaDebito            = "06"
	ContingencyDocFacturaExportacion    = "11"
	ContingencyDocFacturaSujetoExcluido = "14"
)

// ContingencyDocumentTypes is a map of all contingency document types
var ContingencyDocumentTypes = map[string]string{
	ContingencyDocFactura:               "Factura Electrónico",
	ContingencyDocComprobanteCredito:    "Comprobante de Crédito Fiscal Electrónico",
	ContingencyDocNotaRemision:          "Nota de Remisión Electrónica",
	ContingencyDocNotaCredito:           "Nota de Crédito Electrónica",
	ContingencyDocNotaDebito:            "Nota de Débito Electrónica",
	ContingencyDocFacturaExportacion:    "Factura de Exportación Electrónica",
	ContingencyDocFacturaSujetoExcluido: "Factura de Sujeto Excluido Electrónica",
}

// GetContingencyDocumentTypeName returns the name of a contingency document type by code
func GetContingencyDocumentTypeName(code string) (string, bool) {
	name, exists := ContingencyDocumentTypes[code]
	return name, exists
}

// GetContingencyDocumentTypeCode returns the code for a contingency document type by name (case-insensitive)
func GetContingencyDocumentTypeCode(name string) (string, bool) {
	nameLower := strings.ToLower(strings.TrimSpace(name))

	for code, value := range ContingencyDocumentTypes {
		if strings.ToLower(value) == nameLower {
			return code, true
		}
	}
	return "", false
}

// GetAllContingencyDocumentTypes returns a slice of all contingency document types
func GetAllContingencyDocumentTypes() []ContingencyDocumentType {
	types := make([]ContingencyDocumentType, 0, len(ContingencyDocumentTypes))
	for code, value := range ContingencyDocumentTypes {
		types = append(types, ContingencyDocumentType{
			Code:  code,
			Value: value,
		})
	}
	return types
}

// IsValidContingencyDocumentType checks if a contingency document type code is valid
func IsValidContingencyDocumentType(code string) bool {
	_, exists := ContingencyDocumentTypes[code]
	return exists
}
