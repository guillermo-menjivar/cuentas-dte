package dte

// Document type constants
const (
	TipoDteFactura       = "01"
	TipoDteCreditoFiscal = "03"
	// Add more as needed: "04", "05", "06", "07", "08", "09", "11", "14", "15"
)

// Version constants
const (
	VersionFactura       = 1
	VersionCreditoFiscal = 3
)

// Ambiente constants
const (
	AmbientePrueba     = "00"
	AmbienteProduccion = "01"
)

// DTE represents any Documento Tributario Electrónico
type DTE interface {
	GetIdentificacion() Identificacion
	GetTipoDte() string
	GetVersion() int
	Validate() error
}

// BaseIdentificacion contains common identification fields
type BaseIdentificacion struct {
	Version          int     `json:"version"`
	Ambiente         string  `json:"ambiente"`
	TipoDte          string  `json:"tipoDte"`
	NumeroControl    string  `json:"numeroControl"`
	CodigoGeneracion string  `json:"codigoGeneracion"`
	TipoModelo       int     `json:"tipoModelo"`
	TipoOperacion    int     `json:"tipoOperacion"`
	TipoContingencia *int    `json:"tipoContingencia"`
	MotivoContin     *string `json:"motivoContin"`
	FecEmi           string  `json:"fecEmi"`
	HorEmi           string  `json:"horEmi"`
	TipoMoneda       string  `json:"tipoMoneda"`
}

// Identificacion is an alias for BaseIdentificacion
// Each document type embeds this with the same structure
type Identificacion = BaseIdentificacion
