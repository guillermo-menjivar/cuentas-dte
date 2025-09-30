package models

import "strings"

// IVARetentionType represents the type of IVA retention
type IVARetentionType struct {
	Code  string
	Value string
}

// IVA retention type codes
const (
	IVARetention1Percent  = "22"
	IVARetention13Percent = "C4"
	IVARetentionSpecial   = "C9"
)

// IVARetentionTypes is a map of all IVA retention types
var IVARetentionTypes = map[string]string{
	IVARetention1Percent:  "Retención IVA 1%",
	IVARetention13Percent: "Retención IVA 13%",
	IVARetentionSpecial:   "Otras retenciones IVA casos especiales",
}

// GetIVARetentionTypeName returns the name of an IVA retention type by code
func GetIVARetentionTypeName(code string) (string, bool) {
	name, exists := IVARetentionTypes[code]
	return name, exists
}

// GetIVARetentionTypeCode returns the code for an IVA retention type by name (case-insensitive)
func GetIVARetentionTypeCode(name string) (string, bool) {
	nameLower := strings.ToLower(strings.TrimSpace(name))

	for code, value := range IVARetentionTypes {
		if strings.ToLower(value) == nameLower {
			return code, true
		}
	}
	return "", false
}

// GetAllIVARetentionTypes returns a slice of all IVA retention types
func GetAllIVARetentionTypes() []IVARetentionType {
	types := make([]IVARetentionType, 0, len(IVARetentionTypes))
	for code, value := range IVARetentionTypes {
		types = append(types, IVARetentionType{
			Code:  code,
			Value: value,
		})
	}
	return types
}

// IsValidIVARetentionType checks if an IVA retention type code is valid
func IsValidIVARetentionType(code string) bool {
	_, exists := IVARetentionTypes[code]
	return exists
}
