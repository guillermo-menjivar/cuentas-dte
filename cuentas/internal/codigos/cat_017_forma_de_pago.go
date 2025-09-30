package codigos

import "strings"

// PaymentMethod represents a payment method
type PaymentMethod struct {
	Code  string
	Value string
}

// Payment method codes
const (
	PaymentBilletesMonedas     = "01"
	PaymentTarjetaDebito       = "02"
	PaymentTarjetaCredito      = "03"
	PaymentCheque              = "04"
	PaymentTransferencia       = "05"
	PaymentDineroElectronico   = "08"
	PaymentMonederoElectronico = "09"
	PaymentBitcoin             = "11"
	PaymentOtrasCriptomonedas  = "12"
	PaymentCuentasPorPagar     = "13"
	PaymentGiroBancario        = "14"
	PaymentOtros               = "99"
)

// PaymentMethods is a map of all payment methods
var PaymentMethods = map[string]string{
	PaymentBilletesMonedas:     "Billetes y monedas",
	PaymentTarjetaDebito:       "Tarjeta Débito",
	PaymentTarjetaCredito:      "Tarjeta Crédito",
	PaymentCheque:              "Cheque",
	PaymentTransferencia:       "Transferencia-Depósito Bancario",
	PaymentDineroElectronico:   "Dinero electrónico",
	PaymentMonederoElectronico: "Monedero electrónico",
	PaymentBitcoin:             "Bitcoin",
	PaymentOtrasCriptomonedas:  "Otras Criptomonedas",
	PaymentCuentasPorPagar:     "Cuentas por pagar del receptor",
	PaymentGiroBancario:        "Giro bancario",
	PaymentOtros:               "Otros (se debe indicar el medio de pago)",
}

// GetPaymentMethodName returns the name of a payment method by code
func GetPaymentMethodName(code string) (string, bool) {
	name, exists := PaymentMethods[code]
	return name, exists
}

// GetPaymentMethodCode returns the code for a payment method by name (case-insensitive)
func GetPaymentMethodCode(name string) (string, bool) {
	nameLower := strings.ToLower(strings.TrimSpace(name))

	for code, value := range PaymentMethods {
		if strings.ToLower(value) == nameLower {
			return code, true
		}
	}
	return "", false
}

// GetAllPaymentMethods returns a slice of all payment methods
func GetAllPaymentMethods() []PaymentMethod {
	methods := make([]PaymentMethod, 0, len(PaymentMethods))
	for code, value := range PaymentMethods {
		methods = append(methods, PaymentMethod{
			Code:  code,
			Value: value,
		})
	}
	return methods
}

// IsValidPaymentMethod checks if a payment method code is valid
func IsValidPaymentMethod(code string) bool {
	_, exists := PaymentMethods[code]
	return exists
}
