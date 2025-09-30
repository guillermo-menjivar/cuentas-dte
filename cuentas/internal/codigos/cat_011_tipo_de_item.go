package codigo

import "strings"

// ItemType represents the type of item
type ItemType struct {
	Code  string
	Value string
}

// Item type codes
const (
	ItemTypeBienes    = "1"
	ItemTypeServicios = "2"
	ItemTypeAmbos     = "3"
	ItemTypeOtros     = "4"
)

// ItemTypes is a map of all item types
var ItemTypes = map[string]string{
	ItemTypeBienes:    "Bienes",
	ItemTypeServicios: "Servicios",
	ItemTypeAmbos:     "Ambos (Bienes y Servicios, incluye los dos inherente a los Productos o servicios)",
	ItemTypeOtros:     "Otros tributos por ítem",
}

// GetItemTypeName returns the name of an item type by code
func GetItemTypeName(code string) (string, bool) {
	name, exists := ItemTypes[code]
	return name, exists
}

// GetItemTypeCode returns the code for an item type by name (case-insensitive)
func GetItemTypeCode(name string) (string, bool) {
	nameLower := strings.ToLower(strings.TrimSpace(name))

	for code, value := range ItemTypes {
		if strings.ToLower(value) == nameLower {
			return code, true
		}
	}
	return "", false
}

// GetAllItemTypes returns a slice of all item types
func GetAllItemTypes() []ItemType {
	types := make([]ItemType, 0, len(ItemTypes))
	for code, value := range ItemTypes {
		types = append(types, ItemType{
			Code:  code,
			Value: value,
		})
	}
	return types
}

// IsValidItemType checks if an item type code is valid
func IsValidItemType(code string) bool {
	_, exists := ItemTypes[code]
	return exists
}
