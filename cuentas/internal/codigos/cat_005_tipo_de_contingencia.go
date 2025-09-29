package codigo

import "strings"

// ContingencyType represents the type of contingency
type ContingencyType struct {
	Code  string
	Value string
}

// Contingency type codes
const (
	ContingencyMHSystem        = "1"
	ContingencyEmisorSystem    = "2"
	ContingencyInternetService = "3"
	ContingencyPowerService    = "4"
	ContingencyOther           = "5"
)

// ContingencyTypes is a map of all contingency types
var ContingencyTypes = map[string]string{
	ContingencyMHSystem:        "No disponibilidad de sistema del MH",
	ContingencyEmisorSystem:    "No disponibilidad de sistema del emisor",
	ContingencyInternetService: "Falla en el suministro de servicio de Internet del Emisor",
	ContingencyPowerService:    "Falla en el suministro de servicio de energía eléctrica del emisor que impida la transmisión de los DTE",
	ContingencyOther:           "Otro (deberá digitar un máximo de 500 caracteres explicando el motivo)",
}

// GetContingencyTypeName returns the name of a contingency type by code
func GetContingencyTypeName(code string) (string, bool) {
	name, exists := ContingencyTypes[code]
	return name, exists
}

// GetContingencyTypeCode returns the code for a contingency type by name (case-insensitive)
func GetContingencyTypeCode(name string) (string, bool) {
	nameLower := strings.ToLower(strings.TrimSpace(name))

	for code, value := range ContingencyTypes {
		if strings.ToLower(value) == nameLower {
			return code, true
		}
	}
	return "", false
}

// GetAllContingencyTypes returns a slice of all contingency types
func GetAllContingencyTypes() []ContingencyType {
	types := make([]ContingencyType, 0, len(ContingencyTypes))
	for code, value := range ContingencyTypes {
		types = append(types, ContingencyType{
			Code:  code,
			Value: value,
		})
	}
	return types
}

// IsValidContingencyType checks if a contingency type code is valid
func IsValidContingencyType(code string) bool {
	_, exists := ContingencyTypes[code]
	return exists
}
