package dte

// ResumenBase contains common summary fields
type ResumenBase struct {
	TotalNoSuj          float64   `json:"totalNoSuj"`
	TotalExenta         float64   `json:"totalExenta"`
	TotalGravada        float64   `json:"totalGravada"`
	SubTotalVentas      float64   `json:"subTotalVentas"`
	DescuNoSuj          float64   `json:"descuNoSuj"`
	DescuExenta         float64   `json:"descuExenta"`
	DescuGravada        float64   `json:"descuGravada"`
	PorcentajeDescuento float64   `json:"porcentajeDescuento"`
	TotalDescu          float64   `json:"totalDescu"`
	Tributos            []Tributo `json:"tributos"`
	SubTotal            float64   `json:"subTotal"`
	ReteRenta           float64   `json:"reteRenta"`
	MontoTotalOperacion float64   `json:"montoTotalOperacion"`
	TotalNoGravado      float64   `json:"totalNoGravado"`
	TotalPagar          float64   `json:"totalPagar"`
	TotalLetras         string    `json:"totalLetras"`
	SaldoFavor          float64   `json:"saldoFavor"`
	CondicionOperacion  int       `json:"condicionOperacion"`
	Pagos               []Pago    `json:"pagos"`
	NumPagoElectronico  *string   `json:"numPagoElectronico"`
}

// ResumenFactura for Factura
type ResumenFactura struct {
	ResumenBase
	IvaRete1 float64 `json:"ivaRete1"` // IVA Retenido
	TotalIva float64 `json:"totalIva"` // IVA 13%
}

// ResumenCreditoFiscal for Crédito Fiscal
type ResumenCreditoFiscal struct {
	ResumenBase
	IvaPerci1 float64 `json:"ivaPerci1"` // IVA Percibido
	IvaRete1  float64 `json:"ivaRete1"`  // IVA Retenido
}
