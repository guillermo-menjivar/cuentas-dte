package codigoss

import "strings"

// TaxEnclosure represents a tax enclosure/precinct
type TaxEnclosure struct {
	Code  string
	Value string
}

// Tax enclosure codes
const (
	TaxEnclosureSanBartolo         = "01"
	TaxEnclosureAcajutla           = "02"
	TaxEnclosureComalapa           = "03"
	TaxEnclosureLasChinamas        = "04"
	TaxEnclosureLaHachadura        = "05"
	TaxEnclosureSantaAna           = "06"
	TaxEnclosureSanCristobal       = "07"
	TaxEnclosureAnguiatu           = "08"
	TaxEnclosureElAmatillo         = "09"
	TaxEnclosureLaUnion            = "10"
	TaxEnclosureElPoy              = "11"
	TaxEnclosureMetalio            = "12"
	TaxEnclosureFardosPostales     = "15"
	TaxEnclosureSanMarcos          = "16"
	TaxEnclosureElPedregal         = "17"
	TaxEnclosureSanBartoloZF       = "18"
	TaxEnclosureExportsalva        = "20"
	TaxEnclosureAmericanPark       = "21"
	TaxEnclosureInternacional      = "23"
	TaxEnclosureDiez               = "24"
	TaxEnclosureMiramar            = "26"
	TaxEnclosureSantoTomas         = "27"
	TaxEnclosureSantaTecla         = "28"
	TaxEnclosureSantaAnaZF         = "29"
	TaxEnclosureLaConcordia        = "30"
	TaxEnclosureIlopango           = "31"
	TaxEnclosurePipil              = "32"
	TaxEnclosurePuertoBarillas     = "33"
	TaxEnclosureCalvoConservas     = "34"
	TaxEnclosureFeriaInternacional = "35"
	TaxEnclosureElPapalon          = "36"
	TaxEnclosureSamLi              = "37"
	TaxEnclosureSanJose            = "38"
	TaxEnclosureLasMercedes        = "39"
	TaxEnclosureAldesa             = "71"
	TaxEnclosureAgdosaMerliot      = "72"
	TaxEnclosureBodesa             = "73"
	TaxEnclosureDelegacionDHL      = "76"
	TaxEnclosureTransauto          = "77"
	TaxEnclosureNejapa             = "80"
	TaxEnclosureAlmaconsa          = "81"
	TaxEnclosureAgdosaApopa        = "83"
	TaxEnclosureGutierrezCourier   = "85"
	TaxEnclosureSanBartoloEnvio    = "99"
)

// TaxEnclosures is a map of all tax enclosures
var TaxEnclosures = map[string]string{
	"01": "Terrestre San Bartolo",
	"02": "Marítima de Acajutla",
	"03": "Aérea De Comalapa",
	"04": "Terrestre Las Chinamas",
	"05": "Terrestre La Hachadura",
	"06": "Terrestre Santa Ana",
	"07": "Terrestre San Cristóbal",
	"08": "Terrestre Anguiatu",
	"09": "Terrestre El Amatillo",
	"10": "Marítima La Unión",
	"11": "Terrestre El Poy",
	"12": "Terrestre Metalio",
	"15": "Fardos Postales",
	"16": "Z.F. San Marcos",
	"17": "Z.F. El Pedregal",
	"18": "Z.F. San Bartolo",
	"20": "Z.F. Exportsalva",
	"21": "Z.F. American Park",
	"23": "Z.F. Internacional",
	"24": "Z.F. Diez",
	"26": "Z.F. Miramar",
	"27": "Z.F. Santo Tomas",
	"28": "Z.F. Santa Tecla",
	"29": "Z.F. Santa Ana",
	"30": "Z.F. La Concordia",
	"31": "Aérea Ilopango",
	"32": "Z.F. Pipil",
	"33": "Puerto Barillas",
	"34": "Z.F. Calvo Conservas",
	"35": "Feria Internacional",
	"36": "Aduana El Papalón",
	"37": "Z.F. Sam-Li",
	"38": "Z.F. San José",
	"39": "Z.F. Las Mercedes",
	"71": "Aldesa",
	"72": "Agdosa Merliot",
	"73": "Bodesa",
	"76": "Delegacion DHL",
	"77": "Transauto",
	"80": "Nejapa",
	"81": "Almaconsa",
	"83": "Agdosa Apopa",
	"85": "Gutiérrez Courier Y Cargo",
	"99": "San Bartolo Envío Hn/Gt",
}

// GetTaxEnclosureName returns the name of a tax enclosure by code
func GetTaxEnclosureName(code string) (string, bool) {
	name, exists := TaxEnclosures[code]
	return name, exists
}

// GetTaxEnclosureCode returns the code for a tax enclosure by name (case-insensitive)
func GetTaxEnclosureCode(name string) (string, bool) {
	nameLower := strings.ToLower(strings.TrimSpace(name))

	for code, value := range TaxEnclosures {
		if strings.ToLower(value) == nameLower {
			return code, true
		}
	}
	return "", false
}

// GetAllTaxEnclosures returns a slice of all tax enclosures
func GetAllTaxEnclosures() []TaxEnclosure {
	enclosures := make([]TaxEnclosure, 0, len(TaxEnclosures))
	for code, value := range TaxEnclosures {
		enclosures = append(enclosures, TaxEnclosure{
			Code:  code,
			Value: value,
		})
	}
	return enclosures
}

// IsValidTaxEnclosure checks if a tax enclosure code is valid
func IsValidTaxEnclosure(code string) bool {
	_, exists := TaxEnclosures[code]
	return exists
}
