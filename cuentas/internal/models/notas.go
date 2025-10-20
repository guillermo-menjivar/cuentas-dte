// internal/models/nota.go

package models

import (
	"cuentas/internal/codigos"
	"time"
)

// ==================== MAIN NOTA MODEL ====================

type Nota struct {
	// Identity
	ID        string `json:"id" db:"id"`
	CompanyID string `json:"company_id" db:"company_id"`
	Type      string `json:"type" db:"type"` // "05" (Crédito) or "06" (Débito)

	// Business Info
	ClientID        string `json:"client_id" db:"client_id"`
	EstablishmentID string `json:"establishment_id" db:"establishment_id"`
	PointOfSaleID   string `json:"point_of_sale_id" db:"point_of_sale_id"`
	NotaNumber      string `json:"nota_number" db:"nota_number"` // e.g., "ND-2025-00001"

	// Status
	Status        string `json:"status" db:"status"`                 // "draft", "finalized"
	PaymentStatus string `json:"payment_status" db:"payment_status"` // "unpaid", "paid", "partial"

	// DTE Fields (Hacienda)
	DteNumeroControl *string `json:"dte_numero_control,omitempty" db:"dte_numero_control"` // "DTE-06-M003P002-000000000000001"
	DteStatus        *string `json:"dte_status,omitempty" db:"dte_status"`                 // "signed", "failed_signing"
	DteSelloRecibido *string `json:"dte_sello_recibido,omitempty" db:"dte_sello_recibido"` // Hacienda seal

	// Financial Totals
	Subtotal   float64 `json:"subtotal" db:"subtotal"`       // Total before tax
	TaxAmount  float64 `json:"tax_amount" db:"tax_amount"`   // IVA amount
	Total      float64 `json:"total" db:"total"`             // subtotal + tax
	BalanceDue float64 `json:"balance_due" db:"balance_due"` // Amount still owed
	Currency   string  `json:"currency" db:"currency"`       // "USD"

	// Payment Terms (for the adjustment)
	PaymentMethod *string `json:"payment_method,omitempty" db:"payment_method"` // "01"=cash, "02"=check, etc.
	PaymentTerms  *string `json:"payment_terms,omitempty" db:"payment_terms"`   // "immediate", "credit_30", etc.

	// Parent Document Info
	ParentDocumentType string                `json:"parent_document_type" db:"parent_document_type"` // "03" (CCF), "07" (Liquidación)
	RelatedDocuments   []NotaRelatedDocument `json:"related_documents"`                              // 1-50 related docs
	LineItems          []NotaLineItem        `json:"line_items"`                                     // 1-2000 items

	// Additional Info
	Notes *string `json:"notes,omitempty" db:"notes"` // Internal notes

	// Timestamps
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	FinalizedAt *time.Time `json:"finalized_at,omitempty" db:"finalized_at"`
	FinalizedBy *string    `json:"finalized_by,omitempty" db:"finalized_by"` // User ID
}

// ==================== RELATED DOCUMENT ====================

type NotaRelatedDocument struct {
	ID             string    `json:"id" db:"id"`
	NotaID         string    `json:"nota_id" db:"nota_id"`
	CompanyID      string    `json:"company_id" db:"company_id"`
	DocumentType   string    `json:"document_type" db:"document_type"`     // "03" (CCF), "07" (Liquidación)
	GenerationType int       `json:"generation_type" db:"generation_type"` // 1=physical, 2=electronic
	DocumentNumber string    `json:"document_number" db:"document_number"` // UUID or physical number
	DocumentDate   time.Time `json:"document_date" db:"document_date"`     // Date of original document
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// ==================== LINE ITEM ====================

type NotaLineItem struct {
	ID                 string    `json:"id" db:"id"`
	NotaID             string    `json:"nota_id" db:"nota_id"`
	LineNumber         int       `json:"line_number" db:"line_number"`                             // 1, 2, 3...
	ItemType           int       `json:"item_type" db:"item_type"`                                 // 1=goods, 2=services, 3=both, 4=other
	ItemSku            string    `json:"item_sku" db:"item_sku"`                                   // Product code
	ItemName           string    `json:"item_name" db:"item_name"`                                 // Description
	Quantity           float64   `json:"quantity" db:"quantity"`                                   // Amount
	UnitOfMeasure      int       `json:"unit_of_measure" db:"unit_of_measure"`                     // 59=unit, 99=service
	UnitPrice          float64   `json:"unit_price" db:"unit_price"`                               // Price per unit (without IVA)
	DiscountAmount     float64   `json:"discount_amount" db:"discount_amount"`                     // Discount per item
	TaxableAmount      float64   `json:"taxable_amount" db:"taxable_amount"`                       // Amount subject to tax
	TaxAmount          float64   `json:"tax_amount" db:"tax_amount"`                               // IVA for this item
	TotalAmount        float64   `json:"total_amount" db:"total_amount"`                           // Total for this line
	RelatedDocumentRef *string   `json:"related_document_ref,omitempty" db:"related_document_ref"` // Which related doc this item adjusts
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
}

// ==================== REQUEST TYPES ====================

// CreateNotaRequest - Used for both Débito and Crédito
type CreateNotaRequest struct {
	ClientID           string                             `json:"client_id" binding:"required"`
	EstablishmentID    string                             `json:"establishment_id" binding:"required"`
	PointOfSaleID      string                             `json:"point_of_sale_id" binding:"required"`
	ParentDocumentType string                             `json:"parent_document_type" binding:"required"` // "03" or "07"
	RelatedDocuments   []CreateNotaRelatedDocumentRequest `json:"related_documents" binding:"required,min=1,max=50"`
	LineItems          []CreateNotaLineItemRequest        `json:"line_items" binding:"required,min=1,max=2000"`
	PaymentTerms       string                             `json:"payment_terms"` // "immediate", "credit_30", etc.
	Notes              string                             `json:"notes"`
}

type CreateNotaRelatedDocumentRequest struct {
	DocumentType   string `json:"document_type" binding:"required"`   // "03", "07"
	GenerationType int    `json:"generation_type" binding:"required"` // 1 or 2
	DocumentNumber string `json:"document_number" binding:"required"` // UUID or physical number
	DocumentDate   string `json:"document_date" binding:"required"`   // "2025-01-10" (YYYY-MM-DD)
}

type CreateNotaLineItemRequest struct {
	ItemType           int     `json:"item_type" binding:"required"` // 1, 2, 3, or 4
	ItemSku            string  `json:"item_sku" binding:"required"`
	ItemName           string  `json:"item_name" binding:"required"`
	Quantity           float64 `json:"quantity" binding:"required,gt=0"`
	UnitOfMeasure      int     `json:"unit_of_measure" binding:"required"`  // 59, 99, etc.
	UnitPrice          float64 `json:"unit_price" binding:"required,gte=0"` // Price WITHOUT IVA
	DiscountAmount     float64 `json:"discount_amount"`
	RelatedDocumentRef string  `json:"related_document_ref" binding:"required"` // Which doc this adjusts
}

// FinalizeNotaRequest - Optional body when finalizing
type FinalizeNotaRequest struct {
	Notes string `json:"notes,omitempty"` // Optional notes when finalizing
}

func (r *FinalizeNotaRequest) Validate() error {
	// No validation needed - finalize doesn't require payment
	return nil
}

// ==================== HELPER METHODS ====================

// IsDebito returns true if this is a Nota de Débito
func (n *Nota) IsDebito() bool {
	return n.Type == codigos.DocTypeNotaDebito
}

// IsCredito returns true if this is a Nota de Crédito
func (n *Nota) IsCredito() bool {
	return n.Type == codigos.DocTypeNotaCredito
}

// IsDraft returns true if nota is in draft status
func (n *Nota) IsDraft() bool {
	return n.Status == "draft"
}

// IsFinalized returns true if nota has been finalized
func (n *Nota) IsFinalized() bool {
	return n.Status == "finalized"
}
