package codigos

import "strings"

// TributoType represents a type of tax/tribute
type TributoType struct {
	Code    string
	Value   string
	Section string // Section 1, 2, or 3
}

// Tributo codes - Section 1: Applied by items reflected in DTE summary
const (
	TributoIVA13              = "S1.20"
	TributoIVAExportaciones   = "S1.C3"
	TributoTurismoAlojamiento = "S1.59"
	TributoTurismoSalida      = "S1.71"
	TributoFOVIAL             = "S1.D1"
	TributoCOTRANS            = "S1.C8"
	TributoOtrasTasas         = "S1.D5"
	TributoOtrosImpuestos     = "S1.D4"
)

// Tributo codes - Section 2: Applied by items reflected in document body
const (
	TributoEspecialCombustible = "S2.A8"
	TributoIndustriaCemento    = "S2.57"
	TributoEspecialMatricula   = "S2.90"
	TributoOtrosImpuestosD4    = "S2.D4"
	TributoOtrasTasasD5        = "S2.D5"
	TributoAdValoremArmas      = "S2.A6"
)

// Tributo codes - Section 3: Ad-Valorem taxes applied by informative use item
const (
	TributoAdValoremBebidas            = "S3.C5"
	TributoAdValoremTabacoCigarrillos  = "S3.C6"
	TributoAdValoremTabacoCigarros     = "S3.C7"
	TributoFabricanteBebidas           = "S3.19"
	TributoImportadorBebidas           = "S3.28"
	TributoDetallistaBebidas           = "S3.31"
	TributoFabricanteCerveza           = "S3.32"
	TributoImportadorCerveza           = "S3.33"
	TributoFabricanteTabaco            = "S3.34"
	TributoImportadorTabaco            = "S3.35"
	TributoFabricanteArmas             = "S3.36"
	TributoImportadorArmas             = "S3.37"
	TributoFabricanteExplosivos        = "S3.38"
	TributoImportadorExplosivos        = "S3.39"
	TributoFabricantePirotecnicos      = "S3.42"
	TributoImportadorPirotecnicos      = "S3.43"
	TributoProductorTabaco             = "S3.44"
	TributoDistribuidorBebidas         = "S3.50"
	TributoBebAlcoholicas              = "S3.51"
	TributoCerveza                     = "S3.52"
	TributoProductosTabaco             = "S3.53"
	TributoBebidasCarbonatadasGaseosas = "S3.54"
	TributoOtrosEspecificos            = "S3.55"
	TributoAlcohol                     = "S3.58"
	TributoImportadorJugos             = "S3.77"
	TributoDistribuidorJugos           = "S3.78"
	TributoLlamadasTelefonicas         = "S3.79"
	TributoDetallistaJugos             = "S3.85"
	TributoFabricantePreparaciones     = "S3.86"
	TributoFabricanteJugos             = "S3.91"
	TributoImportadorPreparaciones     = "S3.92"
	TributoEspecificosAdValorem        = "S3.A1"
	TributoBebidasGaseosas             = "S3.A5"
	TributoAlcoholEtilico              = "S3.A7"
	TributoSacosSinteticos             = "S3.A9"
)

// Tributos is a map of all tributos with section prefix
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
		// Extract section from code (S1, S2, or S3)
		section := strings.Split(code, ".")[0]
		tributos = append(tributos, TributoType{
			Code:    code,
			Value:   value,
			Section: section,
		})
	}
	return tributos
}

// IsValidTributo checks if a tributo code is valid
func IsValidTributo(code string) bool {
	_, exists := Tributos[code]
	return exists
}

// GetTributosBySection returns all tributos for a specific section
func GetTributosBySection(section string) []TributoType {
	tributos := make([]TributoType, 0)
	prefix := strings.ToUpper(section) + "."

	for code, value := range Tributos {
		if strings.HasPrefix(code, prefix) {
			tributos = append(tributos, TributoType{
				Code:    code,
				Value:   value,
				Section: section,
			})
		}
	}
	return tributos
}
