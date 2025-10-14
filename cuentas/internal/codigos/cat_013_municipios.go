package codigos

import "strings"

// Municipality represents a municipality of El Salvador
type Municipality struct {
	Code  string
	Value string
}

// Municipality codes with dot notation (department.municipality)
const (
	MunicipalityOtro = "00.00"

	// Ahuachapán (01)
	MunicipalityAhuachapanNorte  = "01.13"
	MunicipalityAhuachapanCentro = "01.14"
	MunicipalityAhuachapanSur    = "01.15"

	// Santa Ana (02)
	MunicipalitySantaAnaNorte  = "02.14"
	MunicipalitySantaAnaCentro = "02.15"
	MunicipalitySantaAnaEste   = "02.16"
	MunicipalitySantaAnaOeste  = "02.17"

	// Sonsonate (03)
	MunicipalitySonsonateNorte  = "03.17"
	MunicipalitySonsonateCentro = "03.18"
	MunicipalitySonsonateEste   = "03.19"
	MunicipalitySonsonateOeste  = "03.20"

	// Chalatenango (04)
	MunicipalityChalatenangoNorte  = "04.34"
	MunicipalityChalatanangoCentro = "04.35"
	MunicipalityChalatenangoSur    = "04.36"

	// La Libertad (05)
	MunicipalityLaLibertadNorte  = "05.23"
	MunicipalityLaLibertadCentro = "05.24"
	MunicipalityLaLibertadOeste  = "05.25"
	MunicipalityLaLibertadEste   = "05.26"
	MunicipalityLaLibertadCosta  = "05.27"
	MunicipalityLaLibertadSur    = "05.28"

	// San Salvador (06)
	MunicipalitySanSalvadorNorte  = "06.20"
	MunicipalitySanSalvadorOeste  = "06.21"
	MunicipalitySanSalvadorEste   = "06.22"
	MunicipalitySanSalvadorCentro = "06.23"
	MunicipalitySanSalvadorSur    = "06.24"

	// Cuscatlán (07)
	MunicipalityCuscatlanNorte = "07.17"
	MunicipalityCuscatlanSur   = "07.18"

	// La Paz (08)
	MunicipalityLaPazOeste  = "08.23"
	MunicipalityLaPazCentro = "08.24"
	MunicipalityLaPazEste   = "08.25"

	// Cabañas (09)
	MunicipalityCabanasOeste = "09.10"
	MunicipalityCabanasEste  = "09.11"

	// San Vicente (10)
	MunicipalitySanVicenteNorte = "10.14"
	MunicipalitySanVicenteSur   = "10.15"

	// Usulután (11)
	MunicipalityUsulatanNorte = "11.24"
	MunicipalityUsulatanEste  = "11.25"
	MunicipalityUsulatanOeste = "11.26"

	// San Miguel (12)
	MunicipalitySanMiguelNorte  = "12.21"
	MunicipalitySanMiguelCentro = "12.22"
	MunicipalitySanMiguelOeste  = "12.23"

	// Morazán (13)
	MunicipalityMorazanNorte = "13.27"
	MunicipalityMorazanSur   = "13.28"

	// La Unión (14)
	MunicipalityLaUnionNorte = "14.19"
	MunicipalityLaUnionSur   = "14.20"
)

// Municipalities is a map of all municipalities using dot notation
var Municipalities = map[string]string{
	MunicipalityOtro: "Otro (Para extranjeros)",

	// Ahuachapán
	MunicipalityAhuachapanNorte:  "AHUACHAPAN NORTE",
	MunicipalityAhuachapanCentro: "AHUACHAPAN CENTRO",
	MunicipalityAhuachapanSur:    "AHUACHAPAN SUR",

	// Santa Ana
	MunicipalitySantaAnaNorte:  "SANTA ANA NORTE",
	MunicipalitySantaAnaCentro: "SANTA ANA CENTRO",
	MunicipalitySantaAnaEste:   "SANTA ANA ESTE",
	MunicipalitySantaAnaOeste:  "SANTA ANA OESTE",

	// Sonsonate
	MunicipalitySonsonateNorte:  "SONSONATE NORTE",
	MunicipalitySonsonateCentro: "SONSONATE CENTRO",
	MunicipalitySonsonateEste:   "SONSONATE ESTE",
	MunicipalitySonsonateOeste:  "SONSONATE OESTE",

	// Chalatenango
	MunicipalityChalatenangoNorte:  "CHALATENANGO NORTE",
	MunicipalityChalatanangoCentro: "CHALATENANGO CENTRO",
	MunicipalityChalatenangoSur:    "CHALATENANGO SUR",

	// La Libertad
	MunicipalityLaLibertadNorte:  "LA LIBERTAD NORTE",
	MunicipalityLaLibertadCentro: "LA LIBERTAD CENTRO",
	MunicipalityLaLibertadOeste:  "LA LIBERTAD OESTE",
	MunicipalityLaLibertadEste:   "LA LIBERTAD ESTE",
	MunicipalityLaLibertadCosta:  "LA LIBERTAD COSTA",
	MunicipalityLaLibertadSur:    "LA LIBERTAD SUR",

	// San Salvador
	MunicipalitySanSalvadorNorte:  "SAN SALVADOR NORTE",
	MunicipalitySanSalvadorOeste:  "SAN SALVADOR OESTE",
	MunicipalitySanSalvadorEste:   "SAN SALVADOR ESTE",
	MunicipalitySanSalvadorCentro: "SAN SALVADOR CENTRO",
	MunicipalitySanSalvadorSur:    "SAN SALVADOR SUR",

	// Cuscatlán
	MunicipalityCuscatlanNorte: "CUSCATLAN NORTE",
	MunicipalityCuscatlanSur:   "CUSCATLAN SUR",

	// La Paz
	MunicipalityLaPazOeste:  "LA PAZ OESTE",
	MunicipalityLaPazCentro: "LA PAZ CENTRO",
	MunicipalityLaPazEste:   "LA PAZ ESTE",

	// Cabañas
	MunicipalityCabanasOeste: "CABAÑAS OESTE",
	MunicipalityCabanasEste:  "CABAÑAS ESTE",

	// San Vicente
	MunicipalitySanVicenteNorte: "SAN VICENTE NORTE",
	MunicipalitySanVicenteSur:   "SAN VICENTE SUR",

	// Usulután
	MunicipalityUsulatanNorte: "USULUTAN NORTE",
	MunicipalityUsulatanEste:  "USULUTAN ESTE",
	MunicipalityUsulatanOeste: "USULUTAN OESTE",

	// San Miguel
	MunicipalitySanMiguelNorte:  "SAN MIGUEL NORTE",
	MunicipalitySanMiguelCentro: "SAN MIGUEL CENTRO",
	MunicipalitySanMiguelOeste:  "SAN MIGUEL OESTE",

	// Morazán
	MunicipalityMorazanNorte: "MORAZAN NORTE",
	MunicipalityMorazanSur:   "MORAZAN SUR",

	// La Unión
	MunicipalityLaUnionNorte: "LA UNION NORTE",
	MunicipalityLaUnionSur:   "LA UNION SUR",
}

// MunicipalitiesByDepartment maps department names to their municipalities
// Note: The Code field here contains only the 2-digit municipality part
var MunicipalitiesByDepartment = map[string][]Municipality{
	"ahuachapán": {
		{Code: "13", Value: "AHUACHAPAN NORTE"},
		{Code: "14", Value: "AHUACHAPAN CENTRO"},
		{Code: "15", Value: "AHUACHAPAN SUR"},
	},
	"santa ana": {
		{Code: "14", Value: "SANTA ANA NORTE"},
		{Code: "15", Value: "SANTA ANA CENTRO"},
		{Code: "16", Value: "SANTA ANA ESTE"},
		{Code: "17", Value: "SANTA ANA OESTE"},
	},
	"sonsonate": {
		{Code: "17", Value: "SONSONATE NORTE"},
		{Code: "18", Value: "SONSONATE CENTRO"},
		{Code: "19", Value: "SONSONATE ESTE"},
		{Code: "20", Value: "SONSONATE OESTE"},
	},
	"chalatenango": {
		{Code: "34", Value: "CHALATENANGO NORTE"},
		{Code: "35", Value: "CHALATENANGO CENTRO"},
		{Code: "36", Value: "CHALATENANGO SUR"},
	},
	"la libertad": {
		{Code: "23", Value: "LA LIBERTAD NORTE"},
		{Code: "24", Value: "LA LIBERTAD CENTRO"},
		{Code: "25", Value: "LA LIBERTAD OESTE"},
		{Code: "26", Value: "LA LIBERTAD ESTE"},
		{Code: "27", Value: "LA LIBERTAD COSTA"},
		{Code: "28", Value: "LA LIBERTAD SUR"},
	},
	"san salvador": {
		{Code: "20", Value: "SAN SALVADOR NORTE"},
		{Code: "21", Value: "SAN SALVADOR OESTE"},
		{Code: "22", Value: "SAN SALVADOR ESTE"},
		{Code: "23", Value: "SAN SALVADOR CENTRO"},
		{Code: "24", Value: "SAN SALVADOR SUR"},
	},
	"cuscatlán": {
		{Code: "17", Value: "CUSCATLAN NORTE"},
		{Code: "18", Value: "CUSCATLAN SUR"},
	},
	"la paz": {
		{Code: "23", Value: "LA PAZ OESTE"},
		{Code: "24", Value: "LA PAZ CENTRO"},
		{Code: "25", Value: "LA PAZ ESTE"},
	},
	"cabañas": {
		{Code: "10", Value: "CABAÑAS OESTE"},
		{Code: "11", Value: "CABAÑAS ESTE"},
	},
	"san vicente": {
		{Code: "14", Value: "SAN VICENTE NORTE"},
		{Code: "15", Value: "SAN VICENTE SUR"},
	},
	"usulután": {
		{Code: "24", Value: "USULUTAN NORTE"},
		{Code: "25", Value: "USULUTAN ESTE"},
		{Code: "26", Value: "USULUTAN OESTE"},
	},
	"san miguel": {
		{Code: "21", Value: "SAN MIGUEL NORTE"},
		{Code: "22", Value: "SAN MIGUEL CENTRO"},
		{Code: "23", Value: "SAN MIGUEL OESTE"},
	},
	"morazán": {
		{Code: "27", Value: "MORAZAN NORTE"},
		{Code: "28", Value: "MORAZAN SUR"},
	},
	"la unión": {
		{Code: "19", Value: "LA UNION NORTE"},
		{Code: "20", Value: "LA UNION SUR"},
	},
}

// GetMunicipalitiesByDepartment returns all municipalities for a given department
func GetMunicipalitiesByDepartment(departmentName string) ([]Municipality, bool) {
	deptLower := strings.ToLower(strings.TrimSpace(departmentName))
	municipalities, exists := MunicipalitiesByDepartment[deptLower]
	return municipalities, exists
}

// GetMunicipalityName returns the name of a municipality by full code (DD.MM)
func GetMunicipalityName(code string) (string, bool) {
	name, exists := Municipalities[code]
	return name, exists
}

// GetMunicipalityCode returns the full code (DD.MM) for a municipality by name (case-insensitive)
func GetMunicipalityCode(name string) (string, bool) {
	nameLower := strings.ToLower(strings.TrimSpace(name))

	for code, value := range Municipalities {
		if strings.ToLower(value) == nameLower {
			return code, true
		}
	}
	return "", false
}

// GetAllMunicipalities returns a slice of all municipalities with full codes
func GetAllMunicipalities() []Municipality {
	municipalities := make([]Municipality, 0, len(Municipalities))
	for code, value := range Municipalities {
		municipalities = append(municipalities, Municipality{
			Code:  code,
			Value: value,
		})
	}
	return municipalities
}

// IsValidMunicipality checks if a municipality code (full DD.MM format) is valid
func IsValidMunicipality(code string) bool {
	_, exists := Municipalities[code]
	return exists
}

// IsValidMunicipalityInDepartment checks if a 2-digit municipality code is valid for a specific department
func IsValidMunicipalityInDepartment(departmentCode, municipalityCode string) bool {
	fullCode := departmentCode + "." + municipalityCode
	return IsValidMunicipality(fullCode)
}

// ExtractMunicipalityCode extracts the 2-digit municipality code from either format
// Input: "06.20" or "20"
// Output: "20"
func ExtractMunicipalityCode(municipio string) string {
	// If it contains a dot, it's in full format (DD.MM)
	if strings.Contains(municipio, ".") {
		parts := strings.Split(municipio, ".")
		if len(parts) == 2 {
			return parts[1] // Return just the MM part
		}
	}
	// Already in short format or invalid
	return municipio
}

// ValidateAndExtractMunicipality validates a municipality code and extracts the 2-digit code
// Returns the 2-digit code and whether it's valid
func ValidateAndExtractMunicipality(municipio string) (string, bool) {
	// If it's in full format (DD.MM), validate it exists
	if strings.Contains(municipio, ".") {
		if !IsValidMunicipality(municipio) {
			return "", false
		}
		parts := strings.Split(municipio, ".")
		if len(parts) == 2 {
			return parts[1], true // Return the MM part
		}
		return "", false
	}

	// If it's just 2 digits, assume it's already extracted
	// We can't validate it without knowing the department
	return municipio, true
}

// ValidateMunicipalityWithDepartment validates that a municipality code belongs to a department
// Accepts municipality in either format: "20" or "06.20"
// Returns the 2-digit code if valid
func ValidateMunicipalityWithDepartment(departmentCode, municipio string) (string, bool) {
	// Extract the municipality code
	munCode := ExtractMunicipalityCode(municipio)

	// Validate it exists for this department
	if IsValidMunicipalityInDepartment(departmentCode, munCode) {
		return munCode, true
	}

	return "", false
}
