package dte

// ReceptorFactura is for Factura (optional, more flexible)
type ReceptorFactura struct {
	TipoDocumento *string    `json:"tipoDocumento"`
	NumDocumento  *string    `json:"numDocumento"`
	NRC           *string    `json:"nrc"`
	Nombre        *string    `json:"nombre"`
	CodActividad  *string    `json:"codActividad"`
	DescActividad *string    `json:"descActividad"`
	Direccion     *Direccion `json:"direccion"`
	Telefono      *string    `json:"telefono"`
	Correo        *string    `json:"correo"`
}

// ReceptorCreditoFiscal is for Crédito Fiscal (required, stricter)
type ReceptorCreditoFiscal struct {
	NIT             string    `json:"nit"`
	NRC             string    `json:"nrc"`
	Nombre          string    `json:"nombre"`
	CodActividad    string    `json:"codActividad"`
	DescActividad   string    `json:"descActividad"`
	NombreComercial *string   `json:"nombreComercial"`
	Direccion       Direccion `json:"direccion"`
	Telefono        *string   `json:"telefono"`
	Correo          string    `json:"correo"`
}
