package models

import (
	"fmt"
	"time"
)

// ============================================================================
// NOTA DE CRÉDITO - Request Models
// ============================================================================

// CreateNotaCreditoRequest represents the request to create a Nota de Crédito
type CreateNotaCreditoRequest struct {
	CCFIds            []string                           `json:"ccf_ids" binding:"required,min=1,max=50"`
	CreditReason      string                             `json:"credit_reason" binding:"required"`
	CreditDescription string                             `json:"credit_description"`
	LineItems         []CreateNotaCreditoLineItemRequest `json:"line_items" binding:"required,min=1,max=2000"`
	PaymentTerms      string                             `json:"payment_terms"`
	Notes             string                             `json:"notes"`
}

// CreateNotaCreditoLineItemRequest represents a line item credit in a Nota de Crédito
// Nota de Crédito can credit partial or full amounts from existing CCF line items
type CreateNotaCreditoLineItemRequest struct {
	// Which CCF this line credits
	RelatedCCFId string `json:"related_ccf_id" binding:"required"`

	// Which specific line item in the CCF we're crediting
	CCFLineItemId string `json:"ccf_line_item_id" binding:"required"`

	// How many units are being credited (can be partial: 3 of 10)
	QuantityCredited float64 `json:"quantity_credited" binding:"required,gt=0"`

	// The per-unit credit amount
	// Example: Original was $50/unit, credit is $50/unit for returns
	// Or: Original was $50/unit, credit is $5/unit for partial discount
	CreditAmount float64 `json:"credit_amount" binding:"required,gte=0"`

	// Optional: Why are we crediting this specific line?
	CreditReason string `json:"credit_reason"`
}

// Validate validates the create nota crédito request
func (r *CreateNotaCreditoRequest) Validate() error {
	if len(r.CCFIds) == 0 {
		return fmt.Errorf("at least one CCF ID is required")
	}

	if len(r.CCFIds) > 50 {
		return fmt.Errorf("maximum 50 CCFs allowed per nota")
	}

	if r.CreditReason == "" {
		return fmt.Errorf("credit_reason is required")
	}

	if !IsValidCreditReason(r.CreditReason) {
		return fmt.Errorf("invalid credit_reason: %s", r.CreditReason)
	}

	if len(r.LineItems) == 0 {
		return fmt.Errorf("at least one line item credit is required")
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

// Validate validates a single line item credit
func (r *CreateNotaCreditoLineItemRequest) Validate() error {
	if r.RelatedCCFId == "" {
		return fmt.Errorf("related_ccf_id is required")
	}

	if r.CCFLineItemId == "" {
		return fmt.Errorf("ccf_line_item_id is required")
	}

	if r.QuantityCredited <= 0 {
		return fmt.Errorf("quantity_credited must be greater than 0 (got %.8f)", r.QuantityCredited)
	}

	if r.CreditAmount < 0 {
		return fmt.Errorf("credit_amount cannot be negative (got %.2f)", r.CreditAmount)
	}

	return nil
}

// ============================================================================
// NOTA DE CRÉDITO - Database Models
// ============================================================================

// NotaCredito represents a Nota de Crédito document
type NotaCredito struct {
	ID              string `json:"id"`
	CompanyID       string `json:"company_id"`
	EstablishmentID string `json:"establishment_id"`
	PointOfSaleID   string `json:"point_of_sale_id"`

	// Identification
	NotaNumber string `json:"nota_number"`
	NotaType   string `json:"nota_type"` // Always "05" for Nota de Crédito

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

	// Credit-specific fields
	CreditReason      string  `json:"credit_reason"`
	CreditDescription *string `json:"credit_description,omitempty"`
	IsFullAnnulment   bool    `json:"is_full_annulment"`

	// Financial totals (POSITIVE numbers - amount being credited)
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
	LineItems     []NotaCreditoLineItem     `json:"line_items,omitempty"`
	CCFReferences []NotaCreditoCCFReference `json:"ccf_references,omitempty"`
}

// NotaCreditoLineItem represents a credit to a CCF line item
type NotaCreditoLineItem struct {
	ID            string `json:"id"`
	NotaCreditoID string `json:"nota_credito_id"`
	LineNumber    int    `json:"line_number"`

	// Which CCF and line item this credits
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

	// Credit details
	QuantityCredited float64 `json:"quantity_credited"` // Can be partial (3 of 10)
	CreditAmount     float64 `json:"credit_amount"`     // Per-unit credit amount
	CreditReason     *string `json:"credit_reason,omitempty"`

	// Calculated totals for THIS credit
	// Total credit = credit_amount × quantity_credited
	LineSubtotal   float64 `json:"line_subtotal"`   // credit_amount × quantity_credited
	DiscountAmount float64 `json:"discount_amount"` // Usually 0 for notas
	TaxableAmount  float64 `json:"taxable_amount"`
	TotalTaxes     float64 `json:"total_taxes"`
	LineTotal      float64 `json:"line_total"`

	CreatedAt time.Time `json:"created_at"`
}

// NotaCreditoCCFReference links a nota to the CCFs it credits
type NotaCreditoCCFReference struct {
	ID            string    `json:"id"`
	NotaCreditoID string    `json:"nota_credito_id"`
	CCFId         string    `json:"ccf_id"`
	CCFNumber     string    `json:"ccf_number"`
	CCFDate       time.Time `json:"ccf_date"`
	CreatedAt     time.Time `json:"created_at"`
}

// ============================================================================
// Constants and Validation
// ============================================================================

// Credit reason constants
const (
	CreditReasonVoid         = "void"         // Full cancellation/annulment
	CreditReasonReturn       = "return"       // Goods returned
	CreditReasonDiscount     = "discount"     // Post-sale price reduction
	CreditReasonDefect       = "defect"       // Defective product
	CreditReasonOverbilling  = "overbilling"  // Billed too much
	CreditReasonCorrection   = "correction"   // General correction
	CreditReasonQuality      = "quality"      // Quality issue
	CreditReasonCancellation = "cancellation" // Order cancelled
	CreditReasonOther        = "other"        // Other reason
)

// Status constants
const (
	NotaCreditoStatusDraft     = "draft"
	NotaCreditoStatusFinalized = "finalized"
	NotaCreditoStatusVoided    = "voided"
)

// ValidCreditReasons returns list of valid credit reasons
func ValidCreditReasons() []string {
	return []string{
		CreditReasonVoid,
		CreditReasonReturn,
		CreditReasonDiscount,
		CreditReasonDefect,
		CreditReasonOverbilling,
		CreditReasonCorrection,
		CreditReasonQuality,
		CreditReasonCancellation,
		CreditReasonOther,
	}
}

// IsValidCreditReason checks if a credit reason is valid
func IsValidCreditReason(reason string) bool {
	for _, valid := range ValidCreditReasons() {
		if reason == valid {
			return true
		}
	}
	return false
}
