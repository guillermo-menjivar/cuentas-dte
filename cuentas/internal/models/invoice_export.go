package models

import (
	"fmt"
	"strings"

	"cuentas/internal/codigos"
)

// ============================================
// EXPORT DOCUMENT MODEL
// ============================================

// InvoiceExportDocument represents an export-related document (customs, transport, etc.)
type InvoiceExportDocument struct {
	ID        string `json:"id"`
	InvoiceID string `json:"invoice_id"`

	// Document classification
	CodDocAsociado   int     `json:"cod_doc_asociado"`  // 1-4
	DescDocumento    *string `json:"desc_documento"`    // Document ID/number
	DetalleDocumento *string `json:"detalle_documento"` // Document description

	// Transport information (required if cod_doc_asociado = 4)
	PlacaTrans      *string `json:"placa_trans"`      // Vehicle ID
	ModoTransp      *int    `json:"modo_transp"`      // 1-7
	NumConductor    *string `json:"num_conductor"`    // Driver ID
	NombreConductor *string `json:"nombre_conductor"` // Driver name

	CreatedAt string `json:"created_at"`
}

// CreateInvoiceExportDocumentRequest represents a request to add an export document
type CreateInvoiceExportDocumentRequest struct {
	CodDocAsociado   int     `json:"cod_doc_asociado" binding:"required"`
	DescDocumento    *string `json:"desc_documento"`
	DetalleDocumento *string `json:"detalle_documento"`
	PlacaTrans       *string `json:"placa_trans"`
	ModoTransp       *int    `json:"modo_transp"`
	NumConductor     *string `json:"num_conductor"`
	NombreConductor  *string `json:"nombre_conductor"`
}

// Validate validates the export document request
func (r *CreateInvoiceExportDocumentRequest) Validate() error {
	// Validate cod_doc_asociado
	if !codigos.IsValidExportDocumentType(fmt.Sprintf("%d", r.CodDocAsociado)) {
		return fmt.Errorf("cod_doc_asociado must be 1, 2, 3, or 4")
	}

	// If transport document (4), all transport fields required
	if r.CodDocAsociado == 4 {
		if r.PlacaTrans == nil || strings.TrimSpace(*r.PlacaTrans) == "" {
			return fmt.Errorf("placa_trans is required for transport documents")
		}
		if r.ModoTransp == nil {
			return fmt.Errorf("modo_transp is required for transport documents")
		}
		if !codigos.IsValidModoTransporteExportacion(fmt.Sprintf("%d", *r.ModoTransp)) {
			return fmt.Errorf("modo_transp must be 1-7")
		}
		if r.NumConductor == nil || strings.TrimSpace(*r.NumConductor) == "" {
			return fmt.Errorf("num_conductor is required for transport documents")
		}
		if r.NombreConductor == nil || strings.TrimSpace(*r.NombreConductor) == "" {
			return fmt.Errorf("nombre_conductor is required for transport documents")
		}
	} else {
		// Non-transport docs must NOT have transport fields
		if r.PlacaTrans != nil || r.ModoTransp != nil || r.NumConductor != nil || r.NombreConductor != nil {
			return fmt.Errorf("transport fields not allowed for non-transport documents")
		}
	}

	// If authorization or customs (1, 2), desc and detalle required
	if r.CodDocAsociado == 1 || r.CodDocAsociado == 2 {
		if r.DescDocumento == nil || strings.TrimSpace(*r.DescDocumento) == "" {
			return fmt.Errorf("desc_documento is required for authorization/customs documents")
		}
		if r.DetalleDocumento == nil || strings.TrimSpace(*r.DetalleDocumento) == "" {
			return fmt.Errorf("detalle_documento is required for authorization/customs documents")
		}
	}

	return nil
}

// ============================================
// EXPORT FIELDS FOR INVOICE
// ============================================

// InvoiceExportFields holds all export-specific fields for Type 11 invoices
type InvoiceExportFields struct {
	// Emisor export fields
	TipoItemExpor *int    `json:"tipo_item_expor,omitempty"` // 1, 2, or 3
	RecintoFiscal *string `json:"recinto_fiscal,omitempty"`  // 2 chars, nullable
	Regimen       *string `json:"regimen,omitempty"`         // max 13 chars, nullable

	// Resumen export fields
	IncotermsCode *string  `json:"incoterms_code,omitempty"` // FOB, CIF, etc.
	IncotermsDesc *string  `json:"incoterms_desc,omitempty"` // Description
	Seguro        *float64 `json:"seguro,omitempty"`         // Insurance
	Flete         *float64 `json:"flete,omitempty"`          // Freight
	Observaciones *string  `json:"observaciones,omitempty"`  // Observations

	// International receptor (Type 11 only)
	ReceptorCodPais     *string `json:"receptor_cod_pais,omitempty"`       // Country code
	ReceptorNombrePais  *string `json:"receptor_nombre_pais,omitempty"`    // Country name
	ReceptorTipoDoc     *string `json:"receptor_tipo_documento,omitempty"` // 36,13,02,03,37
	ReceptorNumDoc      *string `json:"receptor_num_documento,omitempty"`  // International doc #
	ReceptorComplemento *string `json:"receptor_complemento,omitempty"`    // Free-form address
}

// ValidateExportFields validates export fields for Type 11 invoices
func (e *InvoiceExportFields) Validate() error {
	// TipoItemExpor is required
	if e.TipoItemExpor == nil {
		return fmt.Errorf("tipo_item_expor is required for export invoices")
	}

	if !codigos.IsValidItemType(fmt.Sprintf("%d", *e.TipoItemExpor)) {
		return fmt.Errorf("tipo_item_expor must be 1, 2, or 3")
	}

	// If services only (2), recinto & regimen must be null
	if *e.TipoItemExpor == 2 {
		if e.RecintoFiscal != nil || e.Regimen != nil {
			return fmt.Errorf("recinto_fiscal and regimen must be null for services-only exports")
		}
	}

	// Validate recinto_fiscal length if provided
	if e.RecintoFiscal != nil && len(*e.RecintoFiscal) != 2 {
		return fmt.Errorf("recinto_fiscal must be exactly 2 characters")
	}

	// Validate regimen length if provided
	if e.Regimen != nil && len(*e.Regimen) > 13 {
		return fmt.Errorf("regimen must be max 13 characters")
	}

	// International receptor fields are required
	if e.ReceptorCodPais == nil || strings.TrimSpace(*e.ReceptorCodPais) == "" {
		return fmt.Errorf("receptor_cod_pais is required for export invoices")
	}

	if e.ReceptorNombrePais == nil || strings.TrimSpace(*e.ReceptorNombrePais) == "" {
		return fmt.Errorf("receptor_nombre_pais is required for export invoices")
	}

	if e.ReceptorTipoDoc == nil {
		return fmt.Errorf("receptor_tipo_documento is required for export invoices")
	}

	if !codigos.IsValidReceptorDocumentType(*e.ReceptorTipoDoc) {
		return fmt.Errorf("receptor_tipo_documento must be 36, 13, 02, 03, or 37")
	}

	if e.ReceptorNumDoc == nil || strings.TrimSpace(*e.ReceptorNumDoc) == "" {
		return fmt.Errorf("receptor_num_documento is required for export invoices")
	}

	if e.ReceptorComplemento == nil || strings.TrimSpace(*e.ReceptorComplemento) == "" {
		return fmt.Errorf("receptor_complemento (address) is required for export invoices")
	}

	// Validate INCOTERMS if provided
	if e.IncotermsCode != nil && *e.IncotermsCode != "" {
		// Just validate it's not empty, don't enforce specific codes
		if len(*e.IncotermsCode) > 10 {
			return fmt.Errorf("incoterms_code must be max 10 characters")
		}
	}

	if e.IncotermsDesc != nil && len(*e.IncotermsDesc) > 150 {
		return fmt.Errorf("incoterms_desc must be max 150 characters")
	}

	// Validate monetary fields are non-negative if provided
	if e.Seguro != nil && *e.Seguro < 0 {
		return fmt.Errorf("seguro must be non-negative")
	}

	if e.Flete != nil && *e.Flete < 0 {
		return fmt.Errorf("flete must be non-negative")
	}

	return nil
}

// ============================================
// UPDATED CREATE INVOICE REQUEST
// ============================================

// Note: Add these fields to your existing CreateInvoiceRequest struct in invoice.go:
//
// // Export-specific fields (Type 11 only) - embed the struct
// ExportFields *InvoiceExportFields `json:"export_fields,omitempty"`
// ExportDocuments []CreateInvoiceExportDocumentRequest `json:"export_documents,omitempty"`
//
// Then update the Validate() method to check if it's an export invoice

// IsExportInvoice checks if this is an export invoice (Type 11)
func IsExportInvoice(req interface{}) bool {
	// Check if ExportFields is present
	// This is a helper - you'll implement the actual check based on your request structure
	return false // Placeholder
}

// ValidateExportInvoice validates a Type 11 export invoice request
func ValidateExportInvoice(exportFields *InvoiceExportFields, exportDocs []CreateInvoiceExportDocumentRequest) error {
	// Validate export fields
	if exportFields == nil {
		return fmt.Errorf("export_fields is required for export invoices")
	}

	if err := exportFields.Validate(); err != nil {
		return fmt.Errorf("export_fields validation failed: %w", err)
	}

	// At least one export document required
	if len(exportDocs) == 0 {
		return fmt.Errorf("at least one export document is required for export invoices")
	}

	if len(exportDocs) > 20 {
		return fmt.Errorf("maximum 20 export documents allowed")
	}

	// Validate each export document
	for i, doc := range exportDocs {
		if err := doc.Validate(); err != nil {
			return fmt.Errorf("export document %d validation failed: %w", i+1, err)
		}
	}

	return nil
}
