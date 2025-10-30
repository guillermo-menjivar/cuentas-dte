package models

import (
	"fmt"
	"time"
)

// ============================================================================
// NOTA DE DÉBITO - Request Models
// ============================================================================

// CreateNotaDebitoRequest represents the request to create a Nota de Débito
type CreateNotaDebitoRequest struct {
	CCFIds       []string                          `json:"ccf_ids" binding:"required,min=1,max=50"`
	LineItems    []CreateNotaDebitoLineItemRequest `json:"line_items" binding:"required,min=1,max=2000"`
	PaymentTerms string                            `json:"payment_terms"`
	Notes        string                            `json:"notes"`
}

// CreateNotaDebitoLineItemRequest represents a line item adjustment in a Nota de Débito
// Nota de Débito can ONLY increase prices of existing CCF line items - cannot add new items
type CreateNotaDebitoLineItemRequest struct {
	// Which CCF this line adjusts
	RelatedCCFId string `json:"related_ccf_id" binding:"required"`

	// Which specific line item in the CCF we're adjusting
	CCFLineItemId string `json:"ccf_line_item_id" binding:"required"`

	// The per-unit price increase
	// Example: Original was $50/unit, adjustment is $10/unit, new effective price is $60/unit
	AdjustmentAmount float64 `json:"adjustment_amount" binding:"required,gt=0"`

	// Optional: Why are we adjusting this?
	AdjustmentReason string `json:"adjustment_reason"`
}

// Validate validates the create nota débito request
func (r *CreateNotaDebitoRequest) Validate() error {
	if len(r.CCFIds) == 0 {
		return fmt.Errorf("at least one CCF ID is required")
	}

	if len(r.CCFIds) > 50 {
		return fmt.Errorf("maximum 50 CCFs allowed per nota")
	}

	if len(r.LineItems) == 0 {
		return fmt.Errorf("at least one line item adjustment is required")
	}

	if len(r.LineItems) > 2000 {
		return fmt.Errorf("maximum 2000 line items allowed per nota")
	}

	for i, item := range r.LineItems {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("line item %d: %w", i+1, err)
		}
	}

	return nil
}

// Validate validates a single line item adjustment
func (r *CreateNotaDebitoLineItemRequest) Validate() error {
	if r.RelatedCCFId == "" {
		return fmt.Errorf("related_ccf_id is required")
	}

	if r.CCFLineItemId == "" {
		return fmt.Errorf("ccf_line_item_id is required")
	}

	if r.AdjustmentAmount <= 0 {
		return fmt.Errorf("adjustment_amount must be greater than 0 (got %.2f)", r.AdjustmentAmount)
	}

	return nil
}

// ============================================================================
// NOTA DE DÉBITO - Database Models
// ============================================================================

// NotaDebito represents a Nota de Débito document
type NotaDebito struct {
	ID              string `json:"id"`
	CompanyID       string `json:"company_id"`
	EstablishmentID string `json:"establishment_id"`
	PointOfSaleID   string `json:"point_of_sale_id"`

	// Identification
	NotaNumber string `json:"nota_number"`
	NotaType   string `json:"nota_type"` // Always "06" for Nota de Débito

	// Client info (inherited from CCFs)
	ClientID                string  `json:"client_id"`
	ClientName              string  `json:"client_name"`
	ClientLegalName         string  `json:"client_legal_name"`
	ClientNit               *string `json:"client_nit,omitempty"`
	ClientNcr               *string `json:"client_ncr,omitempty"`
	ClientDui               *string `json:"client_dui,omitempty"`
	ContactEmail            *string `json:"contact_email,omitempty"`
	ContactWhatsapp         *string `json:"contact_whatsapp,omitempty"`
	ClientAddress           string  `json:"client_address"`
	ClientTipoContribuyente *string `json:"client_tipo_contribuyente,omitempty"`
	ClientTipoPersona       *string `json:"client_tipo_persona,omitempty"`

	// Financial totals
	Subtotal      float64 `json:"subtotal"`
	TotalDiscount float64 `json:"total_discount"`
	TotalTaxes    float64 `json:"total_taxes"`
	Total         float64 `json:"total"`

	Currency string `json:"currency"`

	// Payment
	PaymentTerms  string     `json:"payment_terms"`
	PaymentMethod string     `json:"payment_method"` // Inherited from first CCF
	DueDate       *time.Time `json:"due_date,omitempty"`

	// Status
	Status string `json:"status"` // 'draft', 'finalized', 'voided'

	// DTE tracking
	DteNumeroControl    *string    `json:"dte_numero_control,omitempty"`
	DteCodigoGeneracion *string    `json:"dte_codigo_generacion,omitempty"`
	DteSelloRecibido    *string    `json:"dte_sello_recibido,omitempty"`
	DteStatus           *string    `json:"dte_status,omitempty"`
	DteHaciendaResponse *string    `json:"dte_hacienda_response,omitempty"`
	DteSubmittedAt      *time.Time `json:"dte_submitted_at,omitempty"`

	// Timestamps
	CreatedAt   time.Time  `json:"created_at"`
	FinalizedAt *time.Time `json:"finalized_at,omitempty"`
	VoidedAt    *time.Time `json:"voided_at,omitempty"`

	// Audit
	CreatedBy *string `json:"created_by,omitempty"`
	Notes     *string `json:"notes,omitempty"`

	// Relationships
	LineItems     []NotaDebitoLineItem     `json:"line_items,omitempty"`
	CCFReferences []NotaDebitoCCFReference `json:"ccf_references,omitempty"`
}

// NotaDebitoLineItem represents an adjustment to a CCF line item
type NotaDebitoLineItem struct {
	ID           string `json:"id"`
	NotaDebitoID string `json:"nota_debito_id"`
	LineNumber   int    `json:"line_number"`

	// Which CCF and line item this adjusts
	RelatedCCFId     string `json:"related_ccf_id"`
	RelatedCCFNumber string `json:"related_ccf_number"` // For display
	CCFLineItemId    string `json:"ccf_line_item_id"`

	// Original item details (snapshot from CCF)
	OriginalItemSku       string  `json:"original_item_sku"`
	OriginalItemName      string  `json:"original_item_name"`
	OriginalUnitPrice     float64 `json:"original_unit_price"`
	OriginalQuantity      float64 `json:"original_quantity"`
	OriginalItemTipoItem  string  `json:"original_item_tipo_item"`
	OriginalUnitOfMeasure string  `json:"original_unit_of_measure"`

	// Adjustment details
	AdjustmentAmount float64 `json:"adjustment_amount"` // Per-unit price increase
	AdjustmentReason *string `json:"adjustment_reason,omitempty"`

	// Calculated totals for THIS adjustment
	// Total adjustment = adjustment_amount × original_quantity
	LineSubtotal   float64 `json:"line_subtotal"`   // adjustment_amount × quantity
	DiscountAmount float64 `json:"discount_amount"` // Usually 0 for notas
	TaxableAmount  float64 `json:"taxable_amount"`
	TotalTaxes     float64 `json:"total_taxes"`
	LineTotal      float64 `json:"line_total"`

	CreatedAt time.Time `json:"created_at"`
}

// NotaDebitoCCFReference links a nota to the CCFs it references
type NotaDebitoCCFReference struct {
	ID           string    `json:"id"`
	NotaDebitoID string    `json:"nota_debito_id"`
	CCFId        string    `json:"ccf_id"`
	CCFNumber    string    `json:"ccf_number"`
	CCFDate      time.Time `json:"ccf_date"`
	CreatedAt    time.Time `json:"created_at"`
}
