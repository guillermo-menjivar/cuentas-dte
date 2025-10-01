package codigoss

import "strings"

// Municipality represents a municipality of El Salvador
type Municipality struct {
	Code  string
	Value string
}

// Municipality codes
const (
	MunicipalityOtro = "00"

	// Ahuachapán
	MunicipalityAhuachapanNorte  = "13"
	MunicipalityAhuachapanCentro = "14"
	MunicipalityAhuachapanSur    = "15"

	// Santa Ana
	MunicipalitySantaAnaNorte  = "14"
	MunicipalitySantaAnaCentro = "15"
	MunicipalitySantaAnaEste   = "16"
	MunicipalitySantaAnaOeste  = "17"

	// Sonsonate
	MunicipalitySonsonateNorte  = "17"
	MunicipalitySonsonateCentro = "18"
	MunicipalitySonsonateEste   = "19"
	MunicipalitySonsonateOeste  = "20"

	// Chalatenango
	MunicipalityChalatenangoNorte  = "34"
	MunicipalityChalatanangoCentro = "35"
	MunicipalityChalatenangoSur    = "36"

	// La Libertad
	MunicipalityLaLibertadNorte  = "23"
	MunicipalityLaLibertadCentro = "24"
	MunicipalityLaLibertadOeste  = "25"
	MunicipalityLaLibertadEste   = "26"
	MunicipalityLaLibertadCosta  = "27"
	MunicipalityLaLibertadSur    = "28"

	// San Salvador
	MunicipalitySanSalvadorNorte  = "20"
	MunicipalitySanSalvadorOeste  = "21"
	MunicipalitySanSalvadorEste   = "22"
	MunicipalitySanSalvadorCentro = "23"
	MunicipalitySanSalvadorSur    = "24"

	// Cuscatlán
	MunicipalityCuscatlanNorte = "17"
	MunicipalityCuscatlanSur   = "18"

	// La Paz
	MunicipalityLaPazOeste  = "23"
	MunicipalityLaPazCentro = "24"
	MunicipalityLaPazEste   = "25"

	// Cabañas
	MunicipalityCabanasOeste = "10"
	MunicipalityCabanasEste  = "11"

	// San Vicente
	MunicipalitySanVicenteNorte = "14"
	MunicipalitySanVicenteSur   = "15"

	// Usulután
	MunicipalityUsulatanNorte = "24"
	MunicipalityUsulatanEste  = "25"
	MunicipalityUsulatanOeste = "26"

	// San Miguel
	MunicipalitySanMiguelNorte  = "21"
	MunicipalitySanMiguelCentro = "22"
	MunicipalitySanMiguelOeste  = "23"

	// Morazán
	MunicipalityMorazanNorte = "27"
	MunicipalityMorazanSur   = "28"

	// La Unión
	MunicipalityLaUnionNorte = "19"
	MunicipalityLaUnionSur   = "20"
)

// Municipalities is a map of all municipalities
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

// GetMunicipalityName returns the name of a municipality by code
func GetMunicipalityName(code string) (string, bool) {
	name, exists := Municipalities[code]
	return name, exists
}

// GetMunicipalityCode returns the code for a municipality by name (case-insensitive)
func GetMunicipalityCode(name string) (string, bool) {
	nameLower := strings.ToLower(strings.TrimSpace(name))

	for code, value := range Municipalities {
		if strings.ToLower(value) == nameLower {
			return code, true
		}
	}
	return "", false
}

// GetAllMunicipalities returns a slice of all municipalities
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

// IsValidMunicipality checks if a municipality code is valid
func IsValidMunicipality(code string) bool {
	_, exists := Municipalities[code]
	return exists
}
