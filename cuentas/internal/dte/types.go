package dte

import "database/sql"

// InvoiceType represents the type of invoice
type InvoiceType string

// DTEBuilder builds DTE documents from invoices
type DTEBuilder struct {
	db *sql.DB
}

type CompanyData struct {
	ID                   string
	NIT                  string
	NCR                  int64
	Name                 string
	Email                string
	CodActividad         string
	DescActividad        string
	NombreComercial      *string
	DTEAmbiente          string
	Departamento         string
	Municipio            string
	ComplementoDireccion string
	Telefono             string
}

type EstablishmentData struct {
	ID                   string
	TipoEstablecimiento  string
	CodEstablecimiento   string
	CodPuntoVenta        string
	Departamento         string
	Municipio            string
	ComplementoDireccion string
	Telefono             string
}

type ClientData struct {
	ID               string
	NIT              *int64
	NCR              *int64
	DUI              *int64
	BusinessName     string
	DepartmentCode   string
	MunicipalityCode string
	FullAddress      string
}

const (
	InvoiceTypeConsumidorFinal InvoiceType = "consumidor_final" // B2C
	InvoiceTypeCreditoFiscal   InvoiceType = "credito_fiscal"   // B2B
)

// ItemAmounts holds calculated monetary values for a single item
type ItemAmounts struct {
	PrecioUni    float64
	VentaGravada float64
	IvaItem      float64
}

// ResumenAmounts holds calculated totals for the invoice
type ResumenAmounts struct {
	TotalNoSuj          float64
	TotalExenta         float64
	TotalGravada        float64
	SubTotalVentas      float64
	TotalDescu          float64
	TotalIva            float64
	SubTotal            float64
	MontoTotalOperacion float64
	TotalPagar          float64
}

// DTE represents a complete electronic tax document
type DTE struct {
	Identificacion       Identificacion          `json:"identificacion"`
	DocumentoRelacionado *[]DocumentoRelacionado `json:"documentoRelacionado"`
	Emisor               Emisor                  `json:"emisor"`
	Receptor             *Receptor               `json:"receptor"`
	OtrosDocumentos      *[]OtroDocumento        `json:"otrosDocumentos"`
	VentaTercero         *VentaTercero           `json:"ventaTercero"`
	CuerpoDocumento      []CuerpoDocumentoItem   `json:"cuerpoDocumento"`
	Resumen              Resumen                 `json:"resumen"`
	Extension            *Extension              `json:"extension"`
	Apendice             *[]Apendice             `json:"apendice"`
}

// Identificacion contains document identification data
type Identificacion struct {
	Version          int     `json:"version"`
	Ambiente         string  `json:"ambiente"`
	TipoDte          string  `json:"tipoDte"`
	NumeroControl    string  `json:"numeroControl"`
	CodigoGeneracion string  `json:"codigoGeneracion"`
	TipoModelo       int     `json:"tipoModelo"`
	TipoOperacion    int     `json:"tipoOperacion"`
	TipoContingencia *int    `json:"tipoContingencia"`
	MotivoContin     *string `json:"motivoContin"`
	FecEmi           string  `json:"fecEmi"` // YYYY-MM-DD
	HorEmi           string  `json:"horEmi"` // HH:MM:SS
	TipoMoneda       string  `json:"tipoMoneda"`
}
