package codigos

import "strings"

// RegimenType represents the type of customs regime
type RegimenType struct {
	Code  string
	Value string
}

// Regime type codes
const (
	// Exportación Definitiva
	RegimenExportacionDefinitiva                                        = "EX-1.1000.000"
	RegimenExportacionDefinitivaSustitucion                             = "EX-1.1040.000"
	RegimenExportacionDefinitivaFranquiciaProvisionalDAI                = "EX-1.1041.020"
	RegimenExportacionDefinitivaFranquiciaProvisionalDAIIVA             = "EX-1.1041.021"
	RegimenExportacionDefinitivaFranquiciaDefinitivaMaquinaria          = "EX-1.1048.025"
	RegimenExportacionDefinitivaFranquiciaDefinitivaDistribucion        = "EX-1.1048.031"
	RegimenExportacionDefinitivaFranquiciaDefinitivaLogistica           = "EX-1.1048.032"
	RegimenExportacionDefinitivaFranquiciaDefinitivaCallCenter          = "EX-1.1048.033"
	RegimenExportacionDefinitivaFranquiciaDefinitivaTI                  = "EX-1.1048.034"
	RegimenExportacionDefinitivaFranquiciaDefinitivaInvestigacion       = "EX-1.1048.035"
	RegimenExportacionDefinitivaFranquiciaDefinitivaEmbarcaciones       = "EX-1.1048.036"
	RegimenExportacionDefinitivaFranquiciaDefinitivaAeronaves           = "EX-1.1048.037"
	RegimenExportacionDefinitivaFranquiciaDefinitivaProcesos            = "EX-1.1048.038"
	RegimenExportacionDefinitivaFranquiciaDefinitivaMedico              = "EX-1.1048.039"
	RegimenExportacionDefinitivaFranquiciaDefinitivaFinanciero          = "EX-1.1048.040"
	RegimenExportacionDefinitivaFranquiciaDefinitivaContenedores        = "EX-1.1048.043"
	RegimenExportacionDefinitivaFranquiciaDefinitivaEquiposTecnologicos = "EX-1.1048.044"
	RegimenExportacionDefinitivaFranquiciaDefinitivaAncianos            = "EX-1.1048.054"
	RegimenExportacionDefinitivaFranquiciaDefinitivaTelemedicina        = "EX-1.1048.055"
	RegimenExportacionDefinitivaFranquiciaDefinitivaCinematografia      = "EX-1.1048.056"
	RegimenExportacionDefinitivaDPAComprasLocales                       = "EX-1.1052.000"
	RegimenExportacionDefinitivaZonaFrancaComprasLocales                = "EX-1.1054.000"
	RegimenExportacionDefinitivaEnviosSocorro                           = "EX-1.1100.000"
	RegimenExportacionDefinitivaEnviosPostales                          = "EX-1.1200.000"
	RegimenExportacionDefinitivaDespachoUrgente                         = "EX-1.1300.000"
	RegimenExportacionDefinitivaCourier                                 = "EX-1.1400.000"
	RegimenExportacionDefinitivaCourierMuestras                         = "EX-1.1400.011"
	RegimenExportacionDefinitivaCourierPublicitario                     = "EX-1.1400.012"
	RegimenExportacionDefinitivaCourierDocumentos                       = "EX-1.1400.017"
	RegimenExportacionDefinitivaMenajeCasa                              = "EX-1.1500.000"

	// Exportación Temporal
	RegimenExportacionTemporalPerfeccionamientoPasivo  = "EX-2.2100.000"
	RegimenExportacionTemporalReimportacionMismoEstado = "EX-2.2200.000"
	RegimenTrasladosDefinitivos                        = "EX-2.2400.000"

	// Re-Exportación
	RegimenReexportacionImportacionTemporal                             = "EX-3.3050.000"
	RegimenReexportacionTiendasLibres                                   = "EX-3.3051.000"
	RegimenReexportacionAdmisionTemporalPerfeccionamientoActivo         = "EX-3.3052.000"
	RegimenReexportacionAdmisionTemporal                                = "EX-3.3053.000"
	RegimenReexportacionZonaFranca                                      = "EX-3.3054.000"
	RegimenReexportacionAdmisionTemporalPerfeccionamientoActivoGarantia = "EX-3.3055.000"
	RegimenReexportacionAdmisionTemporalDistribucionInternacional       = "EX-3.3056.000"
	RegimenReexportacionAdmisionTemporalDistribucionMismoParque         = "EX-3.3056.057"
	RegimenReexportacionAdmisionTemporalDistribucionDiferenteParque     = "EX-3.3056.058"
	RegimenReexportacionAdmisionTemporalDistribucionDecreto738          = "EX-3.3056.072"
	RegimenReexportacionAdmisionTemporalLogisticaParque                 = "EX-3.3057.000"
	RegimenReexportacionAdmisionTemporalLogisticaMismoParque            = "EX-3.3057.057"
	RegimenReexportacionAdmisionTemporalLogisticaDiferenteParque        = "EX-3.3057.058"
	RegimenReexportacionAdmisionTemporalCallCenter                      = "EX-3.3058.033"
	RegimenReexportacionAdmisionTemporalEmbarcaciones                   = "EX-3.3058.036"
	RegimenReexportacionAdmisionTemporalAeronaves                       = "EX-3.3058.037"
	RegimenReexportacionAdmisionTemporalContenedores                    = "EX-3.3058.043"
	RegimenReexportacionAdmisionTemporalReparacionEquipo                = "EX-3.3059.000"
	RegimenReexportacionAdmisionTemporalReparacionMismoParque           = "EX-3.3059.057"
	RegimenReexportacionAdmisionTemporalReparacionDiferenteParque       = "EX-3.3059.058"
	RegimenReexportacionDeposito                                        = "EX-3.3070.000"
	RegimenReexportacionDepositoDecreto738                              = "EX-3.3070.072"
	RegimenReexportacionProvDeposito                                    = "EX-3.3071.000"
	RegimenReexportacionProvCentroServicioLSI                           = "EX-3.3057.000"
)

// RegimenTypes is a map of all regime types
var RegimenTypes = map[string]string{
	RegimenExportacionDefinitiva:                                        "Exportación Definitiva, Exportación Definitiva, Régimen Común",
	RegimenExportacionDefinitivaSustitucion:                             "Exportación Definitiva, Exportación Definitiva Sustitución de Mercancías, Régimen Común",
	RegimenExportacionDefinitivaFranquiciaProvisionalDAI:                "Exportación Definitiva, Exportación Definitiva Proveniente de Franquicia Provisional, Franq. Presidenciales exento de DAI",
	RegimenExportacionDefinitivaFranquiciaProvisionalDAIIVA:             "Exportación Definitiva, Exportación Definitiva Proveniente de Franquicia Provisional, Franq. Presidenciales exento de DAI e IVA",
	RegimenExportacionDefinitivaFranquiciaDefinitivaMaquinaria:          "Exportación Definitiva, Exportación Definitiva Proveniente de Franquicia Definitiva, Maquinaria y Equipo LZF. DPA",
	RegimenExportacionDefinitivaFranquiciaDefinitivaDistribucion:        "Exportación Definitiva, Exportación Definitiva Proveniente de Franquicia Definitiva, Distribución Internacional",
	RegimenExportacionDefinitivaFranquiciaDefinitivaLogistica:           "Exportación Definitiva, Exportación Definitiva Proveniente. de Franquicia Definitiva, Operaciones Internacionales de Logística",
	RegimenExportacionDefinitivaFranquiciaDefinitivaCallCenter:          "Exportación Definitiva, Exportación Definitiva Proveniente de Franquicia Definitiva, Centro Internacional de llamadas (Call Center)",
	RegimenExportacionDefinitivaFranquiciaDefinitivaTI:                  "Exportación Definitiva, Exportación Definitiva Proveniente de Franquicia Definitiva, Tecnologías de Información LSI",
	RegimenExportacionDefinitivaFranquiciaDefinitivaInvestigacion:       "Exportación Definitiva, Exportación Definitiva Proveniente de Franquicia Definitiva, Investigación y Desarrollo LSI",
	RegimenExportacionDefinitivaFranquiciaDefinitivaEmbarcaciones:       "Exportación Definitiva, Exportación Definitiva Proveniente de Franquicia Definitiva, Reparación y Mantenimiento de Embarcaciones Marítimas LSI",
	RegimenExportacionDefinitivaFranquiciaDefinitivaAeronaves:           "Exportación Definitiva, Exportación Definitiva Proveniente de Franquicia Definitiva, Reparación y Mantenimiento de Aeronaves LSI",
	RegimenExportacionDefinitivaFranquiciaDefinitivaProcesos:            "Exportación Definitiva, Exportación Definitiva Proveniente de Franquicia Definitiva, Procesos Empresariales LSI",
	RegimenExportacionDefinitivaFranquiciaDefinitivaMedico:              "Exportación Definitiva, Exportación Definitiva Proveniente de Franquicia Definitiva, Servicios Medico-Hospitalarios LSI",
	RegimenExportacionDefinitivaFranquiciaDefinitivaFinanciero:          "Exportación Definitiva, Exportación Definitiva Proveniente de Franquicia Definitiva, Servicios Financieros Internacionales LSI",
	RegimenExportacionDefinitivaFranquiciaDefinitivaContenedores:        "Exportación Definitiva, Exportación Definitiva Proveniente de Franquicia Definitiva, Reparación y Mantenimiento de Contenedores LSI",
	RegimenExportacionDefinitivaFranquiciaDefinitivaEquiposTecnologicos: "Exportación Definitiva, Exportación Definitiva Proveniente de Franquicia Definitiva, Reparación de Equipos Tecnológicos LSI",
	RegimenExportacionDefinitivaFranquiciaDefinitivaAncianos:            "Exportación Definitiva, Exportación Definitiva Proveniente de Franquicia Definitiva, Atención Ancianos y Convalecientes LSI",
	RegimenExportacionDefinitivaFranquiciaDefinitivaTelemedicina:        "Exportación Definitiva, Exportación Definitiva Proveniente de Franquicia Definitiva, Telemedicina LSI",
	RegimenExportacionDefinitivaFranquiciaDefinitivaCinematografia:      "Exportación Definitiva, Exportación Definitiva Proveniente de Franquicia Definitiva, Cinematografía LSI",
	RegimenExportacionDefinitivaDPAComprasLocales:                       "Exportación Definitiva, Exportación Definitiva de DPA con origen en Compras Locales, Régimen Común",
	RegimenExportacionDefinitivaZonaFrancaComprasLocales:                "Exportación Definitiva, Exportación Definitiva de Zona Franca con origen en Compras Locales, Régimen Común",
	RegimenExportacionDefinitivaEnviosSocorro:                           "Exportación Definitiva, Exportación Definitiva de Envíos de Socorro, Régimen Común",
	RegimenExportacionDefinitivaEnviosPostales:                          "Exportación Definitiva, Exportación Definitiva de Envíos Postales, Régimen Común",
	RegimenExportacionDefinitivaDespachoUrgente:                         "Exportación Definitiva, Exportación Definitiva Envíos que requieren despacho urgente, Régimen Común",
	RegimenExportacionDefinitivaCourier:                                 "Exportación Definitiva, Exportación Definitiva Courier, Régimen Común",
	RegimenExportacionDefinitivaCourierMuestras:                         "Exportación Definitiva, Exportación Definitiva Courier, Muestras Sin Valor Comercial",
	RegimenExportacionDefinitivaCourierPublicitario:                     "Exportación Definitiva, Exportación Definitiva Courier, Material Publicitario",
	RegimenExportacionDefinitivaCourierDocumentos:                       "Exportación Definitiva, Exportación Definitiva Courier, Declaración de Documentos",
	RegimenExportacionDefinitivaMenajeCasa:                              "Exportación Definitiva, Exportación Definitiva Menaje de casa, Régimen Común",
	RegimenExportacionTemporalPerfeccionamientoPasivo:                   "Exportación Temporal, Exportación Temporal para Perfeccionamiento Pasivo, Régimen Común",
	RegimenExportacionTemporalReimportacionMismoEstado:                  "Exportación Temporal, Exportación Temporal con Reimportación en el mismo estado, Régimen Común",
	RegimenTrasladosDefinitivos:                                         "Traslados Definitivos",
	RegimenReexportacionImportacionTemporal:                             "Re-Exportación, Reexportación Proveniente de Importación Temporal, Régimen Común",
	RegimenReexportacionTiendasLibres:                                   "Re-Exportación, Reexportación Proveniente de Tiendas Libres, Régimen Común",
	RegimenReexportacionAdmisionTemporalPerfeccionamientoActivo:         "Re-Exportación, Reexportación Proveniente de Admisión Temporal para Perfeccionamiento Activo, Régimen Común",
	RegimenReexportacionAdmisionTemporal:                                "Re-Exportación, Reexportación Proveniente de Admisión Temporal, Régimen Común",
	RegimenReexportacionZonaFranca:                                      "Re-Exportación, Reexportación Proveniente de Régimen de Zona Franca, Régimen Común",
	RegimenReexportacionAdmisionTemporalPerfeccionamientoActivoGarantia: "Re-Exportación, Reexportación Proveniente de Admisión Temporal para Perfeccionamiento Activo con Garantía, Régimen Común",
	RegimenReexportacionAdmisionTemporalDistribucionInternacional:       "Re-Exportación, Reexportación Proveniente de Admisión Temporal Distribución Internacional Parque de Servicios, Régimen Común",
	RegimenReexportacionAdmisionTemporalDistribucionMismoParque:         "Re-Exportación, Reexportación Proveniente de Admisión Temporal Distribución Internacional Parque de Servicios, Remisión entre Usuarios Directos del Mismo Parque de Servicios",
	RegimenReexportacionAdmisionTemporalDistribucionDiferenteParque:     "Re-Exportación, Reexportación Proveniente de Admisión Temporal Distribución Internacional Parque de Servicios, Remisión entre Usuarios Directos de Diferente Parque de Servicios",
	RegimenReexportacionAdmisionTemporalDistribucionDecreto738:          "Re-Exportación, Reexportación Proveniente de Admisión Temporal Distribución Internacional Parque de Servicios, Decreto 738 Eléctricos e Híbridos",
	RegimenReexportacionAdmisionTemporalLogisticaParque:                 "Re-Exportación, Reexportación Proveniente de Admisión Temporal Operaciones Internacional de Logística Parque de Servicios, Régimen Común",
	RegimenReexportacionAdmisionTemporalLogisticaMismoParque:            "Re-Exportación, Reexportación Proveniente de Admisión Temporal Operaciones Internacional de Logística Parque de Servicios, Remisión entre Usuarios Directos del Mismo Parque de Servicios",
	RegimenReexportacionAdmisionTemporalLogisticaDiferenteParque:        "Re-Exportación, Reexportación Proveniente de Admisión Temporal Operaciones Internacional de Logística Parque de Servicios, Remisión entre Usuarios Directos de Diferente Parque de Servicios",
	RegimenReexportacionAdmisionTemporalCallCenter:                      "Re-Exportación, Reexportación Proveniente de Admisión Temporal Centro Servicio LSI, Centro Internacional de llamadas (Call Center)",
	RegimenReexportacionAdmisionTemporalEmbarcaciones:                   "Re-Exportación, Reexportación Proveniente de Admisión Temporal Centro Servicio LSI, Reparación y Mantenimiento de Embarcaciones Marítimas LSI",
	RegimenReexportacionAdmisionTemporalAeronaves:                       "Re-Exportación, Reexportación Proveniente de Admisión Temporal Centro Servicio LSI, Reparación y Mantenimiento de Aeronaves LSI",
	RegimenReexportacionAdmisionTemporalContenedores:                    "Re-Exportación, Reexportación Proveniente de Admisión Temporal Centro Servicio LSI, Reparación y Mantenimiento de Contenedores LSI",
	RegimenReexportacionAdmisionTemporalReparacionEquipo:                "Re-Exportación, Reexportación Proveniente de Admisión Temporal Reparación de Equipo Tecnológico Parque de Servicios, Régimen Común",
	RegimenReexportacionAdmisionTemporalReparacionMismoParque:           "Re-Exportación, Reexportación Proveniente de Admisión Temporal Reparación de Equipo Tecnológico Parque de Servicios, Remisión entre Usuarios Directos del Mismo Parque de Servicios",
	RegimenReexportacionAdmisionTemporalReparacionDiferenteParque:       "Re-Exportación, Reexportación Proveniente de Admisión Temporal Reparación de Equipo Tecnológico Parque de Servicios, Remisión entre Usuarios Directos de Diferente Parque de Servicios",
	RegimenReexportacionDeposito:                                        "Re-Exportación, Reexportación Proveniente de Depósito., Régimen Común",
	RegimenReexportacionDepositoDecreto738:                              "Re-Exportación, Reexportación Proveniente de Depósito., Decreto 738 Eléctricos e Híbridos",
	RegimenReexportacionProvDeposito:                                    "Reexp. Prov. de Deposito.",
	RegimenReexportacionProvCentroServicioLSI:                           "Reexportación Prov. de Centro de Servicio LSI",
}

// GetRegimenTypeName returns the name of a regime type by code
func GetRegimenTypeName(code string) (string, bool) {
	name, exists := RegimenTypes[code]
	return name, exists
}

// GetRegimenTypeCode returns the code for a regime type by name (case-insensitive)
func GetRegimenTypeCode(name string) (string, bool) {
	nameLower := strings.ToLower(strings.TrimSpace(name))
	for code, value := range RegimenTypes {
		if strings.ToLower(value) == nameLower {
			return code, true
		}
	}
	return "", false
}

// GetAllRegimenTypes returns a slice of all regime types
func GetAllRegimenTypes() []RegimenType {
	types := make([]RegimenType, 0, len(RegimenTypes))
	for code, value := range RegimenTypes {
		types = append(types, RegimenType{
			Code:  code,
			Value: value,
		})
	}
	return types
}

// IsValidRegimenType checks if a regime type code is valid
func IsValidRegimenType(code string) bool {
	_, exists := RegimenTypes[code]
	return exists
}
