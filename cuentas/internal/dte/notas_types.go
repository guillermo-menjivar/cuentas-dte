package dte

// NotaDebitoDTE is the root structure for Nota de Débito (tipo 06)
// It excludes fields that are not allowed in the schema
type NotaDebitoDTE struct {
	Identificacion       Identificacion                  `json:"identificacion"`
	DocumentoRelacionado *[]DocumentoRelacionado         `json:"documentoRelacionado"`
	Emisor               NotaDebitoEmisor                `json:"emisor"`
	Receptor             *Receptor                       `json:"receptor"`
	VentaTercero         *VentaTercero                   `json:"ventaTercero"`
	CuerpoDocumento      []NotaDebitoCuerpoDocumentoItem `json:"cuerpoDocumento"`
	Resumen              NotaDebitoResumen               `json:"resumen"`
	Extension            *NotaDebitoExtension            `json:"extension"`
	Apendice             *[]Apendice                     `json:"apendice"`
}

// NotaDebitoEmisor - emisor without codEstable, codPuntoVenta, etc.
type NotaDebitoEmisor struct {
	NIT                 string    `json:"nit"`
	NRC                 string    `json:"nrc"`
	Nombre              string    `json:"nombre"`
	CodActividad        string    `json:"codActividad"`
	DescActividad       string    `json:"descActividad"`
	NombreComercial     *string   `json:"nombreComercial"`
	TipoEstablecimiento string    `json:"tipoEstablecimiento"`
	Direccion           Direccion `json:"direccion"`
	Telefono            *string   `json:"telefono"`
	Correo              string    `json:"correo"`
}

// NotaDebitoCuerpoDocumentoItem - without noGravado, psv
type NotaDebitoCuerpoDocumentoItem struct {
	NumItem         int      `json:"numItem"`
	TipoItem        int      `json:"tipoItem"`
	NumeroDocumento string   `json:"numeroDocumento"` // Required for Nota de Débito
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
}

// NotaDebitoResumen - without pagos, porcentajeDescuento, saldoFavor, totalNoGravado, totalPagar
type NotaDebitoResumen struct {
	TotalNoSuj          float64    `json:"totalNoSuj"`
	TotalExenta         float64    `json:"totalExenta"`
	TotalGravada        float64    `json:"totalGravada"`
	SubTotalVentas      float64    `json:"subTotalVentas"`
	DescuNoSuj          float64    `json:"descuNoSuj"`
	DescuExenta         float64    `json:"descuExenta"`
	DescuGravada        float64    `json:"descuGravada"`
	TotalDescu          float64    `json:"totalDescu"`
	Tributos            *[]Tributo `json:"tributos"`
	SubTotal            float64    `json:"subTotal"`
	IvaPerci1           float64    `json:"ivaPerci1"`
	IvaRete1            float64    `json:"ivaRete1"`
	ReteRenta           float64    `json:"reteRenta"`
	MontoTotalOperacion float64    `json:"montoTotalOperacion"`
	TotalLetras         string     `json:"totalLetras"`
	CondicionOperacion  int        `json:"condicionOperacion"`
	NumPagoElectronico  *string    `json:"numPagoElectronico"`
}

// NotaDebitoExtension - without placaVehiculo
type NotaDebitoExtension struct {
	NombEntrega   *string `json:"nombEntrega"`
	DocuEntrega   *string `json:"docuEntrega"`
	NombRecibe    *string `json:"nombRecibe"`
	DocuRecibe    *string `json:"docuRecibe"`
	Observaciones *string `json:"observaciones"`
}
