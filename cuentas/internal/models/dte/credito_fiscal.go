package dte

// CreditoFiscal represents Comprobante de Crédito Fiscal (tipoDte: "03")
type CreditoFiscal struct {
	Identificacion       Identificacion                 `json:"identificacion"`
	DocumentoRelacionado []DocumentoRelacionado         `json:"documentoRelacionado,omitempty"`
	Emisor               Emisor                         `json:"emisor"`
	Receptor             ReceptorCreditoFiscal          `json:"receptor"`
	OtrosDocumentos      []OtroDocumento                `json:"otrosDocumentos,omitempty"`
	VentaTercero         *VentaTercero                  `json:"ventaTercero"`
	CuerpoDocumento      []CuerpoDocumentoCreditoFiscal `json:"cuerpoDocumento"`
	Resumen              ResumenCreditoFiscal           `json:"resumen"`
	Extension            *Extension                     `json:"extension"`
	Apendice             []Apendice                     `json:"apendice,omitempty"`
}

// Implement DTE interface
func (c *CreditoFiscal) GetIdentificacion() Identificacion {
	return c.Identificacion
}

func (c *CreditoFiscal) GetTipoDte() string {
	return TipoDteCreditoFiscal
}

func (c *CreditoFiscal) GetVersion() int {
	return VersionCreditoFiscal
}

func (c *CreditoFiscal) Validate() error {
	// Validation logic for Crédito Fiscal
	// This will be implemented in validator.go
	return ValidateCreditoFiscal(c)
}
