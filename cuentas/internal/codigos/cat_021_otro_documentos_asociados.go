package codigos

import "strings"

// AssociatedDocument represents a type of associated document
type AssociatedDocument struct {
	Code  string
	Value string
}

// Associated document codes
const (
	AssociatedDocEmisor     = "1"
	AssociatedDocReceptor   = "2"
	AssociatedDocMedico     = "3"
	AssociatedDocTransporte = "4"
)

// AssociatedDocuments is a map of all associated document types
var AssociatedDocuments = map[string]string{
	AssociatedDocEmisor:     "Emisor",
	AssociatedDocReceptor:   "Receptor",
	AssociatedDocMedico:     "Médico (solo aplica para contribuyentes obligados a la presentación de F-958)",
	AssociatedDocTransporte: "Transporte (solo aplica para Factura de exportación)",
}

// GetAssociatedDocumentName returns the name of an associated document type by code
func GetAssociatedDocumentName(code string) (string, bool) {
	name, exists := AssociatedDocuments[code]
	return name, exists
}

// GetAssociatedDocumentCode returns the code for an associated document type by name (case-insensitive)
func GetAssociatedDocumentCode(name string) (string, bool) {
	nameLower := strings.ToLower(strings.TrimSpace(name))

	for code, value := range AssociatedDocuments {
		if strings.ToLower(value) == nameLower {
			return code, true
		}
	}
	return "", false
}

// GetAllAssociatedDocuments returns a slice of all associated document types
func GetAllAssociatedDocuments() []AssociatedDocument {
	docs := make([]AssociatedDocument, 0, len(AssociatedDocuments))
	for code, value := range AssociatedDocuments {
		docs = append(docs, AssociatedDocument{
			Code:  code,
			Value: value,
		})
	}
	return docs
}

// IsValidAssociatedDocument checks if an associated document code is valid
func IsValidAssociatedDocument(code string) bool {
	_, exists := AssociatedDocuments[code]
	return exists
}
