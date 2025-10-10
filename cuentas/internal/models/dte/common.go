package dte

// Direccion represents an address (identical across all DTEs)
type Direccion struct {
	Departamento string `json:"departamento"`
	Municipio    string `json:"municipio"`
	Complemento  string `json:"complemento"`
}

// DocumentoRelacionado represents a related document
type DocumentoRelacionado struct {
	TipoDocumento   string `json:"tipoDocumento"`
	TipoGeneracion  int    `json:"tipoGeneracion"`
	NumeroDocumento string `json:"numeroDocumento"`
	FechaEmision    string `json:"fechaEmision"`
}

// OtroDocumento represents an associated document
type OtroDocumento struct {
	CodDocAsociado   int     `json:"codDocAsociado"`
	DescDocumento    *string `json:"descDocumento"`
	DetalleDocumento *string `json:"detalleDocumento"`
	Medico           *Medico `json:"medico"`
}

// Medico represents medical professional information
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

// Extension represents extension information
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

// Pago represents payment information
type Pago struct {
	Codigo     string  `json:"codigo"`
	MontoPago  float64 `json:"montoPago"`
	Referencia *string `json:"referencia"`
	Plazo      *string `json:"plazo"`
	Periodo    *int    `json:"periodo"`
}

// Tributo represents tax information in summary
type Tributo struct {
	Codigo      string  `json:"codigo"`
	Descripcion string  `json:"descripcion"`
	Valor       float64 `json:"valor"`
}
