package models

import (
	"time"
)

// NotaCredito represents a credit note (tipo DTE 05)
type NotaCredito struct {
	ID              string `json:"id" db:"id"`
	CompanyID       string `json:"company_id" db:"company_id"`
	EstablishmentID string `json:"establishment_id" db:"establishment_id"`
	PointOfSaleID   string `json:"point_of_sale_id" db:"point_of_sale_id"`

	// Document identification
	NotaNumber string `json:"nota_number" db:"nota_number"`
	NotaType   string `json:"nota_type" db:"nota_type"`

	// Client information
	ClientID                string  `json:"client_id" db:"client_id"`
	ClientName              string  `json:"client_name" db:"client_name"`
	ClientLegalName         *string `json:"client_legal_name,omitempty" db:"client_legal_name"`
	ClientNit               *string `json:"client_nit,omitempty" db:"client_nit"`
	ClientNcr               *string `json:"client_ncr,omitempty" db:"client_ncr"`
	ClientDui               *string `json:"client_dui,omitempty" db:"client_dui"`
	ContactEmail            *string `json:"contact_email,omitempty" db:"contact_email"`
	ContactWhatsapp         *string `json:"contact_whatsapp,omitempty" db:"contact_whatsapp"`
	ClientAddress           *string `json:"client_address,omitempty" db:"client_address"`
	ClientTipoContribuyente *string `json:"client_tipo_contribuyente,omitempty" db:"client_tipo_contribuyente"`
	ClientTipoPersona       *string `json:"client_tipo_persona,omitempty" db:"client_tipo_persona"`

	// ⭐ NEW: Credit-specific fields
	CreditReason      string  `json:"credit_reason" db:"credit_reason"`
	CreditDescription *string `json:"credit_description,omitempty" db:"credit_description"`
	IsFullAnnulment   bool    `json:"is_full_annulment" db:"is_full_annulment"`

	// Financial totals (POSITIVE numbers)
	Subtotal      float64 `json:"subtotal" db:"subtotal"`
	TotalDiscount float64 `json:"total_discount" db:"total_discount"`
	TotalTaxes    float64 `json:"total_taxes" db:"total_taxes"`
	Total         float64 `json:"total" db:"total"`
	Currency      string  `json:"currency" db:"currency"`

	// Payment information
	PaymentTerms  string     `json:"payment_terms" db:"payment_terms"`
	PaymentMethod string     `json:"payment_method" db:"payment_method"`
	DueDate       *time.Time `json:"due_date,omitempty" db:"due_date"`

	// Status
	Status string `json:"status" db:"status"`

	// DTE tracking
	DteNumeroControl    *string    `json:"dte_numero_control,omitempty" db:"dte_numero_control"`
	DteCodigoGeneracion *string    `json:"dte_codigo_generacion,omitempty" db:"dte_codigo_generacion"`
	DteSelloRecibido    *string    `json:"dte_sello_recibido,omitempty" db:"dte_sello_recibido"`
	DteStatus           *string    `json:"dte_status,omitempty" db:"dte_status"`
	DteHaciendaResponse []byte     `json:"dte_hacienda_response,omitempty" db:"dte_hacienda_response"`
	DteSubmittedAt      *time.Time `json:"dte_submitted_at,omitempty" db:"dte_submitted_at"`

	// Timestamps
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	FinalizedAt *time.Time `json:"finalized_at,omitempty" db:"finalized_at"`
	VoidedAt    *time.Time `json:"voided_at,omitempty" db:"voided_at"`

	// Audit
	CreatedBy *string `json:"created_by,omitempty" db:"created_by"`
	Notes     *string `json:"notes,omitempty" db:"notes"`

	// Related data (not in DB)
	LineItems     []NotaCreditoLineItem     `json:"line_items,omitempty" db:"-"`
	CCFReferences []NotaCreditoCCFReference `json:"ccf_references,omitempty" db:"-"`
}

// NotaCreditoLineItem represents a line item credit
type NotaCreditoLineItem struct {
	ID            string `json:"id" db:"id"`
	NotaCreditoID string `json:"nota_credito_id" db:"nota_credito_id"`
	LineNumber    int    `json:"line_number" db:"line_number"`

	// References
	RelatedCCFId     string `json:"related_ccf_id" db:"related_ccf_id"`
	RelatedCCFNumber string `json:"related_ccf_number" db:"related_ccf_number"`
	CCFLineItemId    string `json:"ccf_line_item_id" db:"ccf_line_item_id"`

	// Original item snapshot
	OriginalItemSku       string  `json:"original_item_sku" db:"original_item_sku"`
	OriginalItemName      string  `json:"original_item_name" db:"original_item_name"`
	OriginalUnitPrice     float64 `json:"original_unit_price" db:"original_unit_price"`
	OriginalQuantity      float64 `json:"original_quantity" db:"original_quantity"`
	OriginalItemTipoItem  string  `json:"original_item_tipo_item" db:"original_item_tipo_item"`
	OriginalUnitOfMeasure string  `json:"original_unit_of_measure" db:"original_unit_of_measure"`

	// ⭐ Credit details
	QuantityCredited float64 `json:"quantity_credited" db:"quantity_credited"`
	CreditAmount     float64 `json:"credit_amount" db:"credit_amount"`
	CreditReason     *string `json:"credit_reason,omitempty" db:"credit_reason"`

	// Calculated totals
	LineSubtotal   float64 `json:"line_subtotal" db:"line_subtotal"`
	DiscountAmount float64 `json:"discount_amount" db:"discount_amount"`
	TaxableAmount  float64 `json:"taxable_amount" db:"taxable_amount"`
	TotalTaxes     float64 `json:"total_taxes" db:"total_taxes"`
	LineTotal      float64 `json:"line_total" db:"line_total"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// NotaCreditoCCFReference links to CCF invoices
type NotaCreditoCCFReference struct {
	ID            string    `json:"id" db:"id"`
	NotaCreditoID string    `json:"nota_credito_id" db:"nota_credito_id"`
	CCFId         string    `json:"ccf_id" db:"ccf_id"`
	CCFNumber     string    `json:"ccf_number" db:"ccf_number"`
	CCFDate       time.Time `json:"ccf_date" db:"ccf_date"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}
