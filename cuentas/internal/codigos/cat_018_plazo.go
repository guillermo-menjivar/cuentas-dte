package codigoss

import "strings"

// PaymentTerm represents a payment term period
type PaymentTerm struct {
	Code  string
	Value string
}

// Payment term codes
const (
	PaymentTermDias  = "01"
	PaymentTermMeses = "02"
	PaymentTermAnos  = "03"
)

// PaymentTerms is a map of all payment terms
var PaymentTerms = map[string]string{
	PaymentTermDias:  "Días",
	PaymentTermMeses: "Meses",
	PaymentTermAnos:  "Años",
}

// GetPaymentTermName returns the name of a payment term by code
func GetPaymentTermName(code string) (string, bool) {
	name, exists := PaymentTerms[code]
	return name, exists
}

// GetPaymentTermCode returns the code for a payment term by name (case-insensitive)
func GetPaymentTermCode(name string) (string, bool) {
	nameLower := strings.ToLower(strings.TrimSpace(name))

	for code, value := range PaymentTerms {
		if strings.ToLower(value) == nameLower {
			return code, true
		}
	}
	return "", false
}

// GetAllPaymentTerms returns a slice of all payment terms
func GetAllPaymentTerms() []PaymentTerm {
	terms := make([]PaymentTerm, 0, len(PaymentTerms))
	for code, value := range PaymentTerms {
		terms = append(terms, PaymentTerm{
			Code:  code,
			Value: value,
		})
	}
	return terms
}

// IsValidPaymentTerm checks if a payment term code is valid
func IsValidPaymentTerm(code string) bool {
	_, exists := PaymentTerms[code]
	return exists
}
