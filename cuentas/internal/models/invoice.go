package models

import (
	"cuentas/internal/codigos"
	"fmt"
	"strings"
	"time"
)

type InvoiceRelatedDocument struct {
	ID                     string    `json:"id"`
	InvoiceID              string    `json:"invoice_id"`
	RelatedDocumentType    string    `json:"related_document_type"`     // "03", "07"
	RelatedDocumentGenType int       `json:"related_document_gen_type"` // 1 or 2
	RelatedDocumentNumber  string    `json:"related_document_number"`
	RelatedDocumentDate    time.Time `json:"related_document_date"`
	CreatedAt              time.Time `json:"created_at"`
}

// Invoice represents an invoice transaction
type Invoice struct {
	ID              string `json:"id"`
	CompanyID       string `json:"company_id"`
	EstablishmentID string `json:"establishment_id"` // ADD THIS
	PointOfSaleID   string `json:"point_of_sale_id"`
	ClientID        string `json:"client_id"`

	// Invoice identification
	InvoiceNumber string `json:"invoice_number"`
	InvoiceType   string `json:"invoice_type"`

	// Reference (for voids/corrections)
	ReferencesInvoiceID *string `json:"references_invoice_id,omitempty"`
	VoidReason          *string `json:"void_reason,omitempty"`

	// Client snapshot
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

	// Payment tracking
	PaymentTerms  string     `json:"payment_terms"`
	PaymentMethod string     `json:"payment_method"`
	PaymentStatus string     `json:"payment_status"`
	AmountPaid    float64    `json:"amount_paid"`
	BalanceDue    float64    `json:"balance_due"`
	DueDate       *time.Time `json:"due_date,omitempty"`

	// Status
	Status string `json:"status"`

	// DTE tracking
	DteNumeroControl    *string    `json:"dte_numero_control,omitempty"`
	DteSello            *string    `json:"dte_sello"`
	DteStatus           *string    `json:"dte_status,omitempty"`
	DteHaciendaResponse *string    `json:"dte_hacienda_response,omitempty"`
	DteSubmittedAt      *time.Time `json:"dte_submitted_at,omitempty"`
	DteType             *string    `json:"dte_type"`

	// Timestamps
	CreatedAt   time.Time  `json:"created_at"`
	FinalizedAt *time.Time `json:"finalized_at,omitempty"`
	VoidedAt    *time.Time `json:"voided_at,omitempty"`

	// Audit
	CreatedBy *string `json:"created_by,omitempty"`
	VoidedBy  *string `json:"voided_by,omitempty"`
	Notes     *string `json:"notes,omitempty"`

	// Relationships
	LineItems []InvoiceLineItem `json:"line_items,omitempty"`
	Payments  []InvoicePayment  `json:"payments,omitempty"`

	ParentInvoiceType *string                  `json:"parent_invoice_type,omitempty"`
	RelatedDocuments  []InvoiceRelatedDocument `json:"related_documents,omitempty"`
}

// CreateInvoiceRequest represents the request to create an invoice
type CreateInvoiceRequest struct {
	ClientID        string                         `json:"client_id" binding:"required"`
	PaymentTerms    string                         `json:"payment_terms"`
	DueDate         *time.Time                     `json:"due_date"`
	PointOfSaleID   string                         `json:"point_of_sale_id" binding:"required"`
	PaymentMethod   string                         `json:"payment_method" binding:"required"`
	EstablishmentID string                         `json:"establishment_id" binding:"required"`
	Notes           *string                        `json:"notes"`
	ContactEmail    *string                        `json:"contact_email"`
	ContactWhatsapp *string                        `json:"contact_whatsapp"`
	LineItems       []CreateInvoiceLineItemRequest `json:"line_items" binding:"required,min=1"`
	//export fields
	ExportFields    *InvoiceExportFields                 `json:"export_fields,omitempty"`
	ExportDocuments []CreateInvoiceExportDocumentRequest `json:"export_documents,omitempty"`
}

// Validate validates the create invoice request
func (r *CreateInvoiceRequest) Validate() error {
	// Validate client_id
	if strings.TrimSpace(r.ClientID) == "" {
		return fmt.Errorf("client_id is required")
	}

	// Validate payment terms
	validTerms := []string{"cash", "net_30", "net_60", "cuenta"}
	if r.PaymentTerms == "" {
		r.PaymentTerms = "cash"
	} else {
		if !contains(validTerms, r.PaymentTerms) {
			return fmt.Errorf("invalid payment_terms: must be one of %v", validTerms)
		}
	}

	if !codigos.IsValidPaymentMethod(r.PaymentMethod) {
		return ErrInvalidPaymentMethod
	}

	// If payment terms require due date, validate it exists
	if (r.PaymentTerms == "net_30" || r.PaymentTerms == "net_60" || r.PaymentTerms == "cuenta") && r.DueDate == nil {
		return fmt.Errorf("due_date is required for payment_terms: %s", r.PaymentTerms)
	}

	// Validate line items
	if len(r.LineItems) == 0 {
		return fmt.Errorf("at least one line item is required")
	}

	for i, item := range r.LineItems {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("line item %d: %w", i+1, err)
		}
	}

	// If export fields provided, validate as export invoice
	if r.ExportFields != nil {
		if err := ValidateExportInvoice(r.ExportFields, r.ExportDocuments); err != nil {
			return fmt.Errorf("export invoice validation failed: %w", err)
		}
	}

	return nil
}

// FinalizeInvoiceRequest represents payment info required when finalizing
type FinalizeInvoiceRequest struct {
	Payment CreatePaymentRequest `json:"payment" binding:"required"`
}

// Validate validates the finalize invoice request
func (r *FinalizeInvoiceRequest) Validate(invoiceTotal float64) error {
	// Validate the payment itself
	if err := r.Payment.Validate(); err != nil {
		return err
	}

	// Validate amount doesn't exceed invoice total
	if r.Payment.Amount > invoiceTotal {
		return fmt.Errorf("payment amount (%.2f) cannot exceed invoice total (%.2f)", r.Payment.Amount, invoiceTotal)
	}

	return nil
}
