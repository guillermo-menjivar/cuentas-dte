package codigos

import "strings"

// OperationCondition represents the condition/terms of an operation
type OperationCondition struct {
	Code  string
	Value string
}

// Operation condition codes
const (
	OperationContado = "1"
	OperationCredito = "2"
	OperationOtro    = "3"
)

// OperationConditions is a map of all operation conditions
var OperationConditions = map[string]string{
	OperationContado: "Contado",
	OperationCredito: "A crédito",
	OperationOtro:    "Otro",
}

// GetOperationConditionName returns the name of an operation condition by code
func GetOperationConditionName(code string) (string, bool) {
	name, exists := OperationConditions[code]
	return name, exists
}

// GetOperationConditionCode returns the code for an operation condition by name (case-insensitive)
func GetOperationConditionCode(name string) (string, bool) {
	nameLower := strings.ToLower(strings.TrimSpace(name))

	for code, value := range OperationConditions {
		if strings.ToLower(value) == nameLower {
			return code, true
		}
	}
	return "", false
}

// GetAllOperationConditions returns a slice of all operation conditions
func GetAllOperationConditions() []OperationCondition {
	conditions := make([]OperationCondition, 0, len(OperationConditions))
	for code, value := range OperationConditions {
		conditions = append(conditions, OperationCondition{
			Code:  code,
			Value: value,
		})
	}
	return conditions
}

// IsValidOperationCondition checks if an operation condition code is valid
func IsValidOperationCondition(code string) bool {
	_, exists := OperationConditions[code]
	return exists
}
