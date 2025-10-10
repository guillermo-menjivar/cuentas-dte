package dte

// CuerpoDocumentoBase contains common line item fields
type CuerpoDocumentoBase struct {
	NumItem         int      `json:"numItem"`
	TipoItem        int      `json:"tipoItem"`
	NumeroDocumento *string  `json:"numeroDocumento"`
	Codigo          *string  `json:"codigo"`
	CodTributo      *string  `json:"codTributo"`
	Descripcion     string   `json:"descripcion"`
	Cantidad        float64  `json:"cantidad"`
	UniMedida       int      `json:"uniMedida"`
	PrecioUni       float64  `json:"precioUni"`
	MontoDescu      float64  `json:"montoDescu"`
	VentaNoSuj      float64  `json:"ventaNoSuj"`
	VentaExenta     float64  `json:"ventaExenta"`
	VentaGravada    float64  `json:"ventaGravada"`
	Tributos        []string `json:"tributos"`
	Psv             float64  `json:"psv"`
	NoGravado       float64  `json:"noGravado"`
}

// CuerpoDocumentoFactura for Factura (has ivaItem)
type CuerpoDocumentoFactura struct {
	CuerpoDocumentoBase
	IvaItem float64 `json:"ivaItem"`
}

// CuerpoDocumentoCreditoFiscal for Crédito Fiscal (no ivaItem)
type CuerpoDocumentoCreditoFiscal struct {
	CuerpoDocumentoBase
}
