package codigos

import "strings"

// TributoType represents a type of tax/tribute
type TributoType struct {
	Code  string
	Value string
}

// Tributo codes - Section 1: Applied by items reflected in DTE summary
const (
	TributoIVA13              = "20"
	TributoIVAExportaciones   = "C3"
	TributoTurismoAlojamiento = "59"
	TributoTurismoSalida      = "71"
	TributoFOVIAL             = "D1"
	TributoCOTRANS            = "C8"
	TributoOtrasTasas         = "D5"
	TributoOtrosImpuestos     = "D4"
)

// Tributo codes - Section 2: Applied by items reflected in document body
const (
	TributoEspecialCombustible = "A8"
	TributoIndustriaCemento    = "57"
	TributoEspecialMatricula   = "90"
	TributoOtrosImpuestosD4    = "D4"
	TributoOtrasTasasD5        = "D5"
	TributoAdValoremArmas      = "A6"
)

// Tributo codes - Section 3: Ad-Valorem taxes applied by informative use item
const (
	TributoAdValoremBebidas            = "C5"
	TributoAdValoremTabacoCigarrillos  = "C6"
	TributoAdValoremTabacoCigarros     = "C7"
	TributoFabricanteBebidas           = "19"
	TributoImportadorBebidas           = "28"
	TributoDetallistaBebidas           = "31"
	TributoFabricanteCerveza           = "32"
	TributoImportadorCerveza           = "33"
	TributoFabricanteTabaco            = "34"
	TributoImportadorTabaco            = "35"
	TributoFabricanteArmas             = "36"
	TributoImportadorArmas             = "37"
	TributoFabricanteExplosivos        = "38"
	TributoImportadorExplosivos        = "39"
	TributoFabricantePirotecnicos      = "42"
	TributoImportadorPirotecnicos      = "43"
	TributoProductorTabaco             = "44"
	TributoDistribuidorBebidas         = "50"
	TributoBebAlcoholicas              = "51"
	TributoCerveza                     = "52"
	TributoProductosTabaco             = "53"
	TributoBebidasCarbonatadasGaseosas = "54"
	TributoOtrosEspecificos            = "55"
	TributoAlcohol                     = "58"
	TributoImportadorJugos             = "77"
	TributoDistribuidorJugos           = "78"
	TributoLlamadasTelefonicas         = "79"
	TributoDetallistaJugos             = "85"
	TributoFabricantePreparaciones     = "86"
	TributoFabricanteJugos             = "91"
	TributoImportadorPreparaciones     = "92"
	TributoEspecificosAdValorem        = "A1"
	TributoBebidasGaseosas             = "A5"
	TributoAlcoholEtilico              = "A7"
	TributoSacosSinteticos             = "A9"
)

// Tributos is a map of all tributos
var Tributos = map[string]string{
	// Section 1
	TributoIVA13:              "Impuesto al Valor Agregado 13%",
	TributoIVAExportaciones:   "Impuesto al Valor Agregado (exportaciones) 0%",
	TributoTurismoAlojamiento: "Turismo: por alojamiento (5%)",
	TributoTurismoSalida:      "Turismo: salida del país por vía aérea $7.00",
	TributoFOVIAL:             "FOVIAL ($0.20 Ctvs. por galón)",
	TributoCOTRANS:            "COTRANS ($0.10 Ctvs. por galón)",
	TributoOtrasTasas:         "Otras tasas casos especiales",
	TributoOtrosImpuestos:     "Otros impuestos casos especiales",

	// Section 2
	TributoEspecialCombustible: "Impuesto Especial al Combustible (0%, 0.5%, 1%)",
	TributoIndustriaCemento:    "Impuesto industria de Cemento",
	TributoEspecialMatricula:   "Impuesto especial a la primera matrícula",
	TributoOtrosImpuestosD4:    "Otros impuestos casos especiales",
	TributoOtrasTasasD5:        "Otras tasas casos especiales",
	TributoAdValoremArmas:      "Impuesto ad- valorem, armas de fuego, municiones explosivas y artículos similares",

	// Section 3
	TributoAdValoremBebidas:            "Impuesto ad- valorem por diferencial de precios de bebidas alcohólicas (8%)",
	TributoAdValoremTabacoCigarrillos:  "Impuesto ad- valorem por diferencial de precios al tabaco cigarrillos (39%)",
	TributoAdValoremTabacoCigarros:     "Impuesto ad- valorem por diferencial de precios al tabaco cigarros (100%)",
	TributoFabricanteBebidas:           "Fabricante de Bebidas Gaseosas, Isotónicas, Deportivas, Fortificantes, Energizante o Estimulante",
	TributoImportadorBebidas:           "Importador de Bebidas Gaseosas, Isotónicas, Deportivas, Fortificantes, Energizante o Estimulante",
	TributoDetallistaBebidas:           "Detallistas o Expendedores de Bebidas Alcohólicas",
	TributoFabricanteCerveza:           "Fabricante de Cerveza",
	TributoImportadorCerveza:           "Importador de Cerveza",
	TributoFabricanteTabaco:            "Fabricante de Productos de Tabaco",
	TributoImportadorTabaco:            "Importador de Productos de Tabaco",
	TributoFabricanteArmas:             "Fabricante de Armas de Fuego, Municiones y Artículos Similares",
	TributoImportadorArmas:             "Importador de Arma de Fuego, Munición y Artículos. Similares",
	TributoFabricanteExplosivos:        "Fabricante de Explosivos",
	TributoImportadorExplosivos:        "Importador de Explosivos",
	TributoFabricantePirotecnicos:      "Fabricante de Productos Pirotécnicos",
	TributoImportadorPirotecnicos:      "Importador de Productos Pirotécnicos",
	TributoProductorTabaco:             "Productor de Tabaco",
	TributoDistribuidorBebidas:         "Distribuidor de Bebidas Gaseosas, Isotónicas, Deportivas, Fortificantes, Energizante o Estimulante",
	TributoBebAlcoholicas:              "Bebidas Alcohólicas",
	TributoCerveza:                     "Cerveza",
	TributoProductosTabaco:             "Productos del Tabaco",
	TributoBebidasCarbonatadasGaseosas: "Bebidas Carbonatadas o Gaseosas Simples o Endulzadas",
	TributoOtrosEspecificos:            "Otros Específicos",
	TributoAlcohol:                     "Alcohol",
	TributoImportadorJugos:             "Importador de Jugos, Néctares, Bebidas con Jugo y Refrescos",
	TributoDistribuidorJugos:           "Distribuidor de Jugos, Néctares, Bebidas con Jugo y Refrescos",
	TributoLlamadasTelefonicas:         "Sobre Llamadas Telefónicas Provenientes del Ext.",
	TributoDetallistaJugos:             "Detallista de Jugos, Néctares, Bebidas con Jugo y Refrescos",
	TributoFabricantePreparaciones:     "Fabricante de Preparaciones Concentradas o en Polvo para la Elaboración de Bebidas",
	TributoFabricanteJugos:             "Fabricante de Jugos, Néctares, Bebidas con Jugo y Refrescos",
	TributoImportadorPreparaciones:     "Importador de Preparaciones Concentradas o en Polvo para la Elaboración de Bebidas",
	TributoEspecificosAdValorem:        "Específicos y Ad-Valorem",
	TributoBebidasGaseosas:             "Bebidas Gaseosas, Isotónicas, Deportivas, Fortificantes, Energizantes o Estimulantes",
	TributoAlcoholEtilico:              "Alcohol Etílico",
	TributoSacosSinteticos:             "Sacos Sintéticos",
}

// GetTributoName returns the name of a tributo by code
func GetTributoName(code string) (string, bool) {
	name, exists := Tributos[code]
	return name, exists
}

// GetTributoCode returns the code for a tributo by name (case-insensitive)
func GetTributoCode(name string) (string, bool) {
	nameLower := strings.ToLower(strings.TrimSpace(name))

	for code, value := range Tributos {
		if strings.ToLower(value) == nameLower {
			return code, true
		}
	}
	return "", false
}

// GetAllTributos returns a slice of all tributos
func GetAllTributos() []TributoType {
	tributos := make([]TributoType, 0, len(Tributos))
	for code, value := range Tributos {
		tributos = append(tributos, TributoType{
			Code:  code,
			Value: value,
		})
	}
	return tributos
}

// IsValidTributo checks if a tributo code is valid
func IsValidTributo(code string) bool {
	_, exists := Tributos[code]
	return exists
}
