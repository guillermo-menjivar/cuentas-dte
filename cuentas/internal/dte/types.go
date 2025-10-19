// internal/dte/types.go
package dte

import (
	"cuentas/internal/codigos"
	"regexp"
	"strings"
)

// ============================================
// INVOICE TYPE & CALCULATION RESULTS
// ============================================

var (
	// Regex patterns
	nitPattern                  = regexp.MustCompile(`^([0-9]{14}|[0-9]{9})$`)
	codigoGeneracionPattern     = regexp.MustCompile(`^[A-F0-9]{8}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{12}$`)
	numeroControlFacturaPattern = regexp.MustCompile(`^DTE-01-[A-Z0-9]{8}-[0-9]{15}$`)
	numeroControlCCFPattern     = regexp.MustCompile(`^DTE-03-[A-Z0-9]{8}-[0-9]{15}$`)
)

// InvoiceType represents the type of invoice
type InvoiceType string

const (
	InvoiceTypeConsumidorFinal InvoiceType = "consumidor_final" // B2C - price includes IVA
	InvoiceTypeCreditoFiscal   InvoiceType = "credito_fiscal"   // B2B - price excludes IVA
)

// ItemAmounts holds calculated monetary values for a single item
type ItemAmounts struct {
	PrecioUni    float64 // Unit price (with or without IVA depending on invoice type)
	VentaGravada float64 // Taxable amount (with or without IVA depending on invoice type)
	IvaItem      float64 // IVA amount
	MontoDescu   float64 // Discount amount for this item
}

// ResumenAmounts holds calculated totals for the invoice
type ResumenAmounts struct {
	TotalNoSuj          float64 // Total not subject to tax
	TotalExenta         float64 // Total exempt from tax
	TotalGravada        float64 // Total taxable amount
	SubTotalVentas      float64 // Subtotal of sales
	DescuNoSuj          float64 // Discount on non-taxable
	DescuExenta         float64 // Discount on exempt
	DescuGravada        float64 // Discount on taxable
	TotalDescu          float64 // Total discounts
	TotalIva            float64 // Total IVA
	SubTotal            float64 // Subtotal after discounts
	IvaRete1            float64 // IVA retained (1%)
	IvaPerci1           float64
	ReteRenta           float64 // Income tax retained
	MontoTotalOperacion float64 // Total operation amount
	TotalNoGravado      float64 // Total non-taxable charges
	TotalPagar          float64 // Total to pay
	SaldoFavor          float64 // Balance in favor
}

// ============================================
// DATABASE DATA MODELS
// ============================================

// CompanyData represents company (emisor) data from database
type CompanyData struct {
	ID                   string
	NIT                  string
	NCR                  int64
	Name                 string
	Email                string
	CodActividad         string
	DescActividad        string
	NombreComercial      string
	DTEAmbiente          string
	Departamento         string
	Municipio            string
	ComplementoDireccion string
	Telefono             string
}

// EstablishmentData represents establishment and point of sale data
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

// ClientData represents client (receptor) data from database
type ClientData struct {
	ID                string
	TipoPersona       string
	NIT               *int64
	NCR               *int64
	DUI               *int64
	BusinessName      *string
	DepartmentCode    string
	MunicipalityCode  string
	FullAddress       string
	Correo            *string
	Telefono          *string
	CodActividad      *string
	DescActividad     *string
	LegalBusinessName string
}

func (c *ClientData) GetMunicipalityCode() (string, bool) {
	munCode := c.MunicipalityCode

	// Extract municipality code from "05.26" format
	if strings.Contains(munCode, ".") {
		parts := strings.Split(munCode, ".")
		if len(parts) == 2 {
			munCode = parts[1] // Get "26" from "05.26"
		}
	}

	// Validate against codigos
	_, valid := codigos.ValidateMunicipalityWithDepartment(
		c.DepartmentCode,
		munCode,
	)

	return munCode, valid
}

// ============================================
// DTE DOCUMENT STRUCTURE (JSON Output)
// ============================================

// DTE represents a complete electronic tax document (Factura Electrónica)
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

// DocumentoRelacionado represents a related document
type DocumentoRelacionado struct {
	TipoDocumento   string `json:"tipoDocumento"`
	TipoGeneracion  int    `json:"tipoGeneracion"`
	NumeroDocumento string `json:"numeroDocumento"`
	FechaEmision    string `json:"fechaEmision"`
}

// Emisor represents the issuer (company)
type Emisor struct {
	NIT                 string    `json:"nit"`
	NRC                 string    `json:"nrc"`
	Nombre              string    `json:"nombre"`
	CodActividad        string    `json:"codActividad"`
	DescActividad       string    `json:"descActividad"`
	NombreComercial     *string   `json:"nombreComercial"`
	TipoEstablecimiento string    `json:"tipoEstablecimiento"`
	Direccion           Direccion `json:"direccion"`
	Telefono            string    `json:"telefono"`
	Correo              string    `json:"correo"`
	CodEstableMH        *string   `json:"codEstableMH"`
	CodEstable          *string   `json:"codEstable"`
	CodPuntoVentaMH     *string   `json:"codPuntoVentaMH"`
	CodPuntoVenta       *string   `json:"codPuntoVenta"`
}

// Receptor represents the recipient (customer/client)
type Receptor struct {
	NIT             *string    `json:"nit,omitempty"`
	TipoDocumento   *string    `json:"tipoDocumento,omitempty"` // ⚠️ Factura only - omit for CCF
	NumDocumento    *string    `json:"numDocumento,omitempty"`  // ⚠️ Factura only - omit for CCF
	NRC             *string    `json:"nrc,omitempty"`
	Nombre          *string    `json:"nombre,omitempty"`
	NombreComercial *string    `json:"nombreComercial,omitempty"` // ⚠️ CCF only - omit for Factura
	CodActividad    *string    `json:"codActividad,omitempty"`    // ⚠️ CCF only - omit for Factura
	DescActividad   *string    `json:"descActividad,omitempty"`   // ⚠️ CCF only - omit for Factura
	Direccion       *Direccion `json:"direccion,omitempty"`
	Telefono        *string    `json:"telefono,omitempty"`
	Correo          *string    `json:"correo,omitempty"`
}

// Direccion represents an address
type Direccion struct {
	Departamento string `json:"departamento"`
	Municipio    string `json:"municipio"`
	Complemento  string `json:"complemento"`
}

// OtroDocumento represents associated documents
type OtroDocumento struct {
	CodDocAsociado   int     `json:"codDocAsociado"`
	DescDocumento    *string `json:"descDocumento"`
	DetalleDocumento *string `json:"detalleDocumento"`
	Medico           *Medico `json:"medico"`
}

// Medico represents medical information (for medical services)
type Medico struct {
	Nombre            string  `json:"nombre"`
	NIT               *string `json:"nit"`
	DocIdentificacion *string `json:"docIdentificacion"`
	TipoServicio      int     `json:"tipoServicio"`
}

// VentaTercero represents third-party sales
type VentaTercero struct {
	NIT    string `json:"nit"`
	Nombre string `json:"nombre"`
}

// CuerpoDocumentoItem represents a line item in the invoice
type CuerpoDocumentoItem struct {
	NumItem         int      `json:"numItem"`
	TipoItem        int      `json:"tipoItem"`
	NumeroDocumento *string  `json:"numeroDocumento"`
	Cantidad        float64  `json:"cantidad"`
	Codigo          *string  `json:"codigo"`
	CodTributo      *string  `json:"codTributo"`
	UniMedida       int      `json:"uniMedida"`
	Descripcion     string   `json:"descripcion"`
	PrecioUni       float64  `json:"precioUni"`
	MontoDescu      float64  `json:"montoDescu"`
	VentaNoSuj      float64  `json:"ventaNoSuj"`
	VentaExenta     float64  `json:"ventaExenta"`
	VentaGravada    float64  `json:"ventaGravada"`
	Tributos        []string `json:"tributos"`
	Psv             float64  `json:"psv"`
	NoGravado       float64  `json:"noGravado"`
	IvaItem         float64  `json:"ivaItem,omitempty"`
}

// Resumen represents the invoice summary/totals
type Resumen struct {
	TotalNoSuj          float64    `json:"totalNoSuj"`
	TotalExenta         float64    `json:"totalExenta"`
	TotalGravada        float64    `json:"totalGravada"`
	SubTotalVentas      float64    `json:"subTotalVentas"`
	DescuNoSuj          float64    `json:"descuNoSuj"`
	DescuExenta         float64    `json:"descuExenta"`
	DescuGravada        float64    `json:"descuGravada"`
	PorcentajeDescuento float64    `json:"porcentajeDescuento"`
	TotalDescu          float64    `json:"totalDescu"`
	Tributos            *[]Tributo `json:"tributos"`
	SubTotal            float64    `json:"subTotal"`
	IvaRete1            float64    `json:"ivaRete1"`
	IvaPerci1           float64    `json:"ivaPerci1"`
	ReteRenta           float64    `json:"reteRenta"`
	MontoTotalOperacion float64    `json:"montoTotalOperacion"`
	TotalNoGravado      float64    `json:"totalNoGravado"`
	TotalPagar          float64    `json:"totalPagar"`
	TotalLetras         string     `json:"totalLetras"`
	TotalIva            float64    `json:"totalIva,omitempty"`
	SaldoFavor          float64    `json:"saldoFavor"`
	CondicionOperacion  int        `json:"condicionOperacion"`
	Pagos               *[]Pago    `json:"pagos"`
	NumPagoElectronico  *string    `json:"numPagoElectronico"`
}

// Tributo represents a tax/tribute item
type Tributo struct {
	Codigo      string  `json:"codigo"`
	Descripcion string  `json:"descripcion"`
	Valor       float64 `json:"valor"`
}

// Pago represents payment information
type Pago struct {
	Codigo     string   `json:"codigo"`
	MontoPago  float64  `json:"montoPago"`
	Referencia *string  `json:"referencia"`
	Plazo      *string  `json:"plazo"`
	Periodo    *float64 `json:"periodo"`
}

// Extension represents additional invoice information
type Extension struct {
	NombEntrega   *string `json:"nombEntrega"`
	DocuEntrega   *string `json:"docuEntrega"`
	NombRecibe    *string `json:"nombRecibe"`
	DocuRecibe    *string `json:"docuRecibe"`
	Observaciones *string `json:"observaciones"`
	PlacaVehiculo *string `json:"placaVehiculo"`
}

// Apendice represents appendix information
type Apendice struct {
	Campo    string `json:"campo"`
	Etiqueta string `json:"etiqueta"`
	Valor    string `json:"valor"`
}

type DocumentoRelacionado struct {
	TipoDocumento   string `json:"tipoDocumento"`
	TipoGeneracion  int    `json:"tipoGeneracion"`
	NumeroDocumento string `json:"numeroDocumento"`
	FechaEmision    string `json:"fechaEmision"`
}

type NotaDebitoElectronica struct {
	Identificacion       Identificacion          `json:"identificacion"`
	DocumentoRelacionado *[]DocumentoRelacionado `json:"documentoRelacionado"` // ⭐ Mandatory for ND
	Emisor               Emisor                  `json:"emisor"`
	Receptor             Receptor                `json:"receptor"`
	VentaTercero         *VentaTercero           `json:"ventaTercero"`
	CuerpoDocumento      []CuerpoDocumentoItem   `json:"cuerpoDocumento"`
	Resumen              Resumen                 `json:"resumen"`
	Extension            *Extension              `json:"extension"`
	Apendice             *[]Apendice             `json:"apendice"`
}
