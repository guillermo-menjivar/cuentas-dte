package models

import "time"

type Nota struct {
	ID              string `json:"id" db:"id"`
	CompanyID       string `json:"company_id" db:"company_id"`
	Type            string `json:"type" db:"type"` // "05" or "06"
	ClientID        string `json:"client_id" db:"client_id"`
	EstablishmentID string `json:"establishment_id" db:"establishment_id"`
	PointOfSaleID   string `json:"point_of_sale_id" db:"point_of_sale_id"`
	NotaNumber      string `json:"nota_number" db:"nota_number"`
	Status          string `json:"status" db:"status"` // draft, finalized
	PaymentStatus   string `json:"payment_status" db:"payment_status"`

	// DTE Fields
	DteNumeroControl *string `json:"dte_numero_control,omitempty" db:"dte_numero_control"`
	DteStatus        *string `json:"dte_status,omitempty" db:"dte_status"`
	DteSelloRecibido *string `json:"dte_sello_recibido,omitempty" db:"dte_sello_recibido"`

	// Financial
	Subtotal   float64 `json:"subtotal" db:"subtotal"`
	TaxAmount  float64 `json:"tax_amount" db:"tax_amount"`
	Total      float64 `json:"total" db:"total"`
	BalanceDue float64 `json:"balance_due" db:"balance_due"`
	Currency   string  `json:"currency" db:"currency"`

	// Payment
	PaymentMethod *string `json:"payment_method,omitempty" db:"payment_method"`
	PaymentTerms  *string `json:"payment_terms,omitempty" db:"payment_terms"`

	// Related
	ParentDocumentType string                `json:"parent_document_type" db:"parent_document_type"` // "03", "07"
	RelatedDocuments   []NotaRelatedDocument `json:"related_documents"`
	LineItems          []NotaLineItem        `json:"line_items"`

	// Metadata
	Notes       *string    `json:"notes,omitempty" db:"notes"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	FinalizedAt *time.Time `json:"finalized_at,omitempty" db:"finalized_at"`
	FinalizedBy *string    `json:"finalized_by,omitempty" db:"finalized_by"`
}

type NotaRelatedDocument struct {
	ID             string    `json:"id" db:"id"`
	NotaID         string    `json:"nota_id" db:"nota_id"`
	CompanyID      string    `json:"company_id" db:"company_id"`
	DocumentType   string    `json:"document_type" db:"document_type"`     // "03", "07"
	GenerationType int       `json:"generation_type" db:"generation_type"` // 1=physical, 2=electronic
	DocumentNumber string    `json:"document_number" db:"document_number"` // UUID or physical number
	DocumentDate   time.Time `json:"document_date" db:"document_date"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

type NotaLineItem struct {
	ID                 string    `json:"id" db:"id"`
	NotaID             string    `json:"nota_id" db:"nota_id"`
	LineNumber         int       `json:"line_number" db:"line_number"`
	ItemType           int       `json:"item_type" db:"item_type"`
	ItemSku            string    `json:"item_sku" db:"item_sku"`
	ItemName           string    `json:"item_name" db:"item_name"`
	Quantity           float64   `json:"quantity" db:"quantity"`
	UnitOfMeasure      int       `json:"unit_of_measure" db:"unit_of_measure"`
	UnitPrice          float64   `json:"unit_price" db:"unit_price"`
	DiscountAmount     float64   `json:"discount_amount" db:"discount_amount"`
	TaxableAmount      float64   `json:"taxable_amount" db:"taxable_amount"`
	TaxAmount          float64   `json:"tax_amount" db:"tax_amount"`
	TotalAmount        float64   `json:"total_amount" db:"total_amount"`
	RelatedDocumentRef *string   `json:"related_document_ref,omitempty" db:"related_document_ref"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
}

type CreateNotaRequest struct {
	ClientID           string                             `json:"client_id" binding:"required"`
	EstablishmentID    string                             `json:"establishment_id" binding:"required"`
	PointOfSaleID      string                             `json:"point_of_sale_id" binding:"required"`
	ParentDocumentType string                             `json:"parent_document_type" binding:"required"` // "03", "07"
	RelatedDocuments   []CreateNotaRelatedDocumentRequest `json:"related_documents" binding:"required,min=1,max=50"`
	LineItems          []CreateNotaLineItemRequest        `json:"line_items" binding:"required,min=1,max=2000"`
	PaymentTerms       string                             `json:"payment_terms"`
	Notes              string                             `json:"notes"`
}

type CreateNotaRelatedDocumentRequest struct {
	DocumentType   string `json:"document_type" binding:"required"`   // "03", "07"
	GenerationType int    `json:"generation_type" binding:"required"` // 1 or 2
	DocumentNumber string `json:"document_number" binding:"required"`
	DocumentDate   string `json:"document_date" binding:"required"` // YYYY-MM-DD
}

type CreateNotaLineItemRequest struct {
	ItemType           int     `json:"item_type" binding:"required"`
	ItemSku            string  `json:"item_sku" binding:"required"`
	ItemName           string  `json:"item_name" binding:"required"`
	Quantity           float64 `json:"quantity" binding:"required,gt=0"`
	UnitOfMeasure      int     `json:"unit_of_measure" binding:"required"`
	UnitPrice          float64 `json:"unit_price" binding:"required,gte=0"`
	DiscountAmount     float64 `json:"discount_amount"`
	RelatedDocumentRef string  `json:"related_document_ref" binding:"required"`
}

type FinalizeNotaRequest struct {
	Payment PaymentInfo `json:"payment" binding:"required"`
}

func (r *FinalizeNotaRequest) Validate(total float64) error {
	return r.Payment.Validate(total)
}
