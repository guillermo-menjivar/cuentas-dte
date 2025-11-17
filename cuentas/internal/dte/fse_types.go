package dte

import "regexp"

// ============================================
// TYPE 14 - FSE (FACTURA SUJETO EXCLUIDO) TYPES
// ============================================

var (
	numeroControlFSEPattern = regexp.MustCompile(`^DTE-14-[A-Z0-9]{8}-[0-9]{15}$`)
)

// FSE represents a complete FSE (Type 14) DTE
type FSE struct {
	Identificacion  FSEIdentificacion `json:"identificacion"`
	Emisor          Emisor            `json:"emisor"`
	SujetoExcluido  FSESujetoExcluido `json:"sujetoExcluido"`
	CuerpoDocumento []FSECuerpoItem   `json:"cuerpoDocumento"`
	Resumen         FSEResumen        `json:"resumen"`
	Apendice        *[]Apendice       `json:"apendice"`
}

type FSEEmisor struct {
	NIT             string    `json:"nit"`
	NRC             string    `json:"nrc"`
	Nombre          string    `json:"nombre"`
	CodActividad    string    `json:"codActividad"`
	DescActividad   string    `json:"descActividad"`
	Direccion       Direccion `json:"direccion"`
	Telefono        string    `json:"telefono"`
	Correo          string    `json:"correo"`
	CodEstableMH    *string   `json:"codEstableMH"`
	CodEstable      *string   `json:"codEstable"`
	CodPuntoVentaMH *string   `json:"codPuntoVentaMH"`
	CodPuntoVenta   *string   `json:"codPuntoVenta"`
}

// FSEIdentificacion - Type 14 uses version 1 (not 3!)
type FSEIdentificacion struct {
	Version          int     `json:"version"` // Always 1 for Type 14
	Ambiente         string  `json:"ambiente"`
	TipoDte          string  `json:"tipoDte"` // Always "14"
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

// FSESujetoExcluido represents the informal supplier (receptor for purchase)
type FSESujetoExcluido struct {
	TipoDocumento *string   `json:"tipoDocumento,omitempty"` // "36", "13", "37", etc.
	NumDocumento  *string   `json:"numDocumento,omitempty"`
	Nombre        string    `json:"nombre"`
	CodActividad  *string   `json:"codActividad,omitempty"`
	DescActividad *string   `json:"descActividad,omitempty"`
	Direccion     Direccion `json:"direccion"`
	Telefono      *string   `json:"telefono"`
	Correo        *string   `json:"correo"`
}

// FSECuerpoItem represents a line item in FSE purchase
type FSECuerpoItem struct {
	NumItem     int     `json:"numItem"`
	TipoItem    int     `json:"tipoItem"`
	Cantidad    float64 `json:"cantidad"`
	Codigo      *string `json:"codigo"`
	UniMedida   int     `json:"uniMedida"`
	Descripcion string  `json:"descripcion"`
	PrecioUni   float64 `json:"precioUni"`
	MontoDescu  float64 `json:"montoDescu"`
	Compra      float64 `json:"compra"` // ⭐ NOT ventaGravada - it's "compra" for FSE
}

// FSEResumen represents the FSE summary/totals
type FSEResumen struct {
	TotalCompra        float64    `json:"totalCompra"`
	Descu              float64    `json:"descu"` // Discount percentage
	TotalDescu         *float64   `json:"totalDescu"`
	SubTotal           float64    `json:"subTotal"`
	IvaRete1           float64    `json:"ivaRete1"`
	ReteRenta          float64    `json:"reteRenta"`
	TotalPagar         float64    `json:"totalPagar"`
	TotalLetras        string     `json:"totalLetras"`
	CondicionOperacion int        `json:"condicionOperacion"`
	Pagos              *[]FSEPago `json:"pagos"`
	Observaciones      *string    `json:"observaciones"`
}

// FSEPago represents payment information for FSE
type FSEPago struct {
	Codigo     string  `json:"codigo"`
	MontoPago  float64 `json:"montoPago"`
	Referencia *string `json:"referencia"`
	Plazo      *string `json:"plazo"`
	Periodo    *int    `json:"periodo"`
}
