package dte

// FacturaElectronica represents Factura (tipoDte: "01")
type FacturaElectronica struct {
	Identificacion       Identificacion           `json:"identificacion"`
	DocumentoRelacionado []DocumentoRelacionado   `json:"documentoRelacionado,omitempty"`
	Emisor               Emisor                   `json:"emisor"`
	Receptor             *ReceptorFactura         `json:"receptor"`
	OtrosDocumentos      []OtroDocumento          `json:"otrosDocumentos,omitempty"`
	VentaTercero         *VentaTercero            `json:"ventaTercero"`
	CuerpoDocumento      []CuerpoDocumentoFactura `json:"cuerpoDocumento"`
	Resumen              ResumenFactura           `json:"resumen"`
	Extension            *Extension               `json:"extension"`
	Apendice             []Apendice               `json:"apendice,omitempty"`
}

// Implement DTE interface
func (f *FacturaElectronica) GetIdentificacion() Identificacion {
	return f.Identificacion
}

func (f *FacturaElectronica) GetTipoDte() string {
	return TipoDteFactura
}

func (f *FacturaElectronica) GetVersion() int {
	return VersionFactura
}

func (f *FacturaElectronica) Validate() error {
	// Validation logic for Factura
	// This will be implemented in validator.go
	return ValidateFactura(f)
}
