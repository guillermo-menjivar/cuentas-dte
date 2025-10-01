package codigoss

import "strings"

// UnitOfMeasure represents a unit of measure
type UnitOfMeasure struct {
	Code  string
	Value string
}

// Unit of measure codes
const (
	UnitMetro             = "1"
	UnitYarda             = "2"
	UnitMilimetro         = "6"
	UnitKilometroCuadrado = "9"
	UnitHectarea          = "10"
	UnitMetroCuadrado     = "13"
	UnitVaraCuadrada      = "15"
	UnitMetroCubico       = "18"
	UnitBarril            = "20"
	UnitGalon             = "22"
	UnitLitro             = "23"
	UnitBotella           = "24"
	UnitMililitro         = "26"
	UnitTonelada          = "30"
	UnitQuintal           = "32"
	UnitArroba            = "33"
	UnitKilogramo         = "34"
	UnitLibra             = "36"
	UnitOnzaTroy          = "37"
	UnitOnza              = "38"
	UnitGramo             = "39"
	UnitMiligramo         = "40"
	UnitMegawatt          = "42"
	UnitKilowatt          = "43"
	UnitWatt              = "44"
	UnitMegavoltioAmperio = "45"
	UnitKilovoltioAmperio = "46"
	UnitVoltioAmperio     = "47"
	UnitGigawattHora      = "49"
	UnitMegawattHora      = "50"
	UnitKilowattHora      = "51"
	UnitWattHora          = "52"
	UnitKilovoltio        = "53"
	UnitVoltio            = "54"
	UnitMillar            = "55"
	UnitMedioMillar       = "56"
	UnitCiento            = "57"
	UnitDocena            = "58"
	UnitUnidad            = "59"
	UnitOtra              = "99"
)

// UnitsOfMeasure is a map of all units of measure
var UnitsOfMeasure = map[string]string{
	UnitMetro:             "metro",
	UnitYarda:             "Yarda",
	UnitMilimetro:         "milímetro",
	UnitKilometroCuadrado: "kilómetro cuadrado",
	UnitHectarea:          "Hectárea",
	UnitMetroCuadrado:     "metro cuadrado",
	UnitVaraCuadrada:      "Vara cuadrada",
	UnitMetroCubico:       "metro cúbico",
	UnitBarril:            "Barril",
	UnitGalon:             "Galón",
	UnitLitro:             "Litro",
	UnitBotella:           "Botella",
	UnitMililitro:         "Mililitro",
	UnitTonelada:          "Tonelada",
	UnitQuintal:           "Quintal",
	UnitArroba:            "Arroba",
	UnitKilogramo:         "Kilogramo",
	UnitLibra:             "Libra",
	UnitOnzaTroy:          "Onza troy",
	UnitOnza:              "Onza",
	UnitGramo:             "Gramo",
	UnitMiligramo:         "Miligramo",
	UnitMegawatt:          "Megawatt",
	UnitKilowatt:          "Kilowatt",
	UnitWatt:              "Watt",
	UnitMegavoltioAmperio: "Megavoltio-amperio",
	UnitKilovoltioAmperio: "Kilovoltio-amperio",
	UnitVoltioAmperio:     "Voltio-amperio",
	UnitGigawattHora:      "Gigawatt-hora",
	UnitMegawattHora:      "Megawatt-hora",
	UnitKilowattHora:      "Kilowatt-hora",
	UnitWattHora:          "Watt-hora",
	UnitKilovoltio:        "Kilovoltio",
	UnitVoltio:            "Voltio",
	UnitMillar:            "Millar",
	UnitMedioMillar:       "Medio millar",
	UnitCiento:            "Ciento",
	UnitDocena:            "Docena",
	UnitUnidad:            "Unidad",
	UnitOtra:              "Otra",
}

// GetUnitOfMeasureName returns the name of a unit of measure by code
func GetUnitOfMeasureName(code string) (string, bool) {
	name, exists := UnitsOfMeasure[code]
	return name, exists
}

// GetUnitOfMeasureCode returns the code for a unit of measure by name (case-insensitive)
func GetUnitOfMeasureCode(name string) (string, bool) {
	nameLower := strings.ToLower(strings.TrimSpace(name))

	for code, value := range UnitsOfMeasure {
		if strings.ToLower(value) == nameLower {
			return code, true
		}
	}
	return "", false
}

// GetAllUnitsOfMeasure returns a slice of all units of measure
func GetAllUnitsOfMeasure() []UnitOfMeasure {
	units := make([]UnitOfMeasure, 0, len(UnitsOfMeasure))
	for code, value := range UnitsOfMeasure {
		units = append(units, UnitOfMeasure{
			Code:  code,
			Value: value,
		})
	}
	return units
}

// IsValidUnitOfMeasure checks if a unit of measure code is valid
func IsValidUnitOfMeasure(code string) bool {
	_, exists := UnitsOfMeasure[code]
	return exists
}
