package codigo

import "strings"

// DocumentType represents the type of fiscal document
type DocumentType struct {
	Code  string
	Value string
}

// Document type codes
const (
	DocTypeFactura                = "01"
	DocTypeComprobanteCredito     = "03"
	DocTypeNotaRemision           = "04"
	DocTypeNotaCredito            = "05"
	DocTypeNotaDebito             = "06"
	DocTypeComprobanteRetencion   = "07"
	DocTypeComprobanteLiquidacion = "08"
	DocTypeDocumentoLiquidacion   = "09"
	DocTypeFacturasExportacion    = "11"
	DocTypeFacturaSujetoExcluido  = "14"
	DocTypeComprobanteDonacion    = "15"
)

// DocumentTypes is a map of all document types
var DocumentTypes = map[string]string{
	DocTypeFactura:                "Factura",
	DocTypeComprobanteCredito:     "Comprobante de crédito fiscal",
	DocTypeNotaRemision:           "Nota de remisión",
	DocTypeNotaCredito:            "Nota de crédito",
	DocTypeNotaDebito:             "Nota de débito",
	DocTypeComprobanteRetencion:   "Comprobante de retención",
	DocTypeComprobanteLiquidacion: "Comprobante de liquidación",
	DocTypeDocumentoLiquidacion:   "Documento contable de liquidación",
	DocTypeFacturasExportacion:    "Facturas de exportación",
	DocTypeFacturaSujetoExcluido:  "Factura de sujeto excluido",
	DocTypeComprobanteDonacion:    "Comprobante de donación",
}

// GetDocumentTypeName returns the name of a document type by code
func GetDocumentTypeName(code string) (string, bool) {
	name, exists := DocumentTypes[code]
	return name, exists
}

// GetDocumentTypeCode returns the code for a document type by name (case-insensitive)
func GetDocumentTypeCode(name string) (string, bool) {
	nameLower := strings.ToLower(strings.TrimSpace(name))

	for code, value := range DocumentTypes {
		if strings.ToLower(value) == nameLower {
			return code, true
		}
	}
	return "", false
}

// GetAllDocumentTypes returns a slice of all document types
func GetAllDocumentTypes() []DocumentType {
	types := make([]DocumentType, 0, len(DocumentTypes))
	for code, value := range DocumentTypes {
		types = append(types, DocumentType{
			Code:  code,
			Value: value,
		})
	}
	return types
}

// IsValidDocumentType checks if a document type code is valid
func IsValidDocumentType(code string) bool {
	_, exists := DocumentTypes[code]
	return exists
}
