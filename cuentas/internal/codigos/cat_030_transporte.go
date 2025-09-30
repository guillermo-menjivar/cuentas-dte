package codigos

import "strings"

// TransportType represents a type of transport
type TransportType struct {
	Code  string
	Value string
}

// Transport type codes
const (
	TransportTerrestre  = "1"
	TransportAereo      = "2"
	TransportMaritimo   = "3"
	TransportFerreo     = "4"
	TransportMultimodal = "5"
	TransportCorreo     = "6"
)

// TransportTypes is a map of all transport types
var TransportTypes = map[string]string{
	TransportTerrestre:  "TERRESTRE",
	TransportAereo:      "AÉREO",
	TransportMaritimo:   "MARÍTIMO",
	TransportFerreo:     "FÉRREO",
	TransportMultimodal: "MULTIMODAL",
	TransportCorreo:     "CORREO",
}

// GetTransportTypeName returns the name of a transport type by code
func GetTransportTypeName(code string) (string, bool) {
	name, exists := TransportTypes[code]
	return name, exists
}

// GetTransportTypeCode returns the code for a transport type by name (case-insensitive)
func GetTransportTypeCode(name string) (string, bool) {
	nameLower := strings.ToLower(strings.TrimSpace(name))

	for code, value := range TransportTypes {
		if strings.ToLower(value) == nameLower {
			return code, true
		}
	}
	return "", false
}

// GetAllTransportTypes returns a slice of all transport types
func GetAllTransportTypes() []TransportType {
	types := make([]TransportType, 0, len(TransportTypes))
	for code, value := range TransportTypes {
		types = append(types, TransportType{
			Code:  code,
			Value: value,
		})
	}
	return types
}

// IsValidTransportType checks if a transport type code is valid
func IsValidTransportType(code string) bool {
	_, exists := TransportTypes[code]
	return exists
}
