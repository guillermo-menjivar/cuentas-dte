package dte

// Emisor represents the document issuer
// Note: Minor field length differences between documents are handled at validation level
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
