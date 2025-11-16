package models

import (
	"cuentas/internal/codigos"
	"fmt"
	"strings"
	"time"
)

// DateOnly is a custom type for date-only fields (YYYY-MM-DD)
type DateOnly struct {
	time.Time
}

// UnmarshalJSON implements custom JSON unmarshaling for date-only format
func (d *DateOnly) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	if s == "null" || s == "" {
		return nil
	}

	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return err
	}

	d.Time = t
	return nil
}

// MarshalJSON implements custom JSON marshaling for date-only format
func (d DateOnly) MarshalJSON() ([]byte, error) {
	if d.Time.IsZero() {
		return []byte("null"), nil
	}
	return []byte("\"" + d.Time.Format("2006-01-02") + "\""), nil
}

// Purchase represents a purchase transaction (FSE, regular purchases, imports, etc.)
type Purchase struct {
	ID              string `json:"id"` // This is also the codigoGeneracion
	CompanyID       string `json:"company_id"`
	EstablishmentID string `json:"establishment_id"`
	PointOfSaleID   string `json:"point_of_sale_id"`

	// Purchase identification
	PurchaseNumber string    `json:"purchase_number"`
	PurchaseType   string    `json:"purchase_type"` // 'fse', 'regular', 'import', 'other'
	PurchaseDate   time.Time `json:"purchase_date"`

	// Supplier reference (for regular purchases from registered suppliers)
	SupplierID *string `json:"supplier_id,omitempty"`

	// FSE-specific: Informal supplier info (when supplier_id is NULL)
	SupplierName              *string `json:"supplier_name,omitempty"`
	SupplierDocumentType      *string `json:"supplier_document_type,omitempty"` // '36' (NIT), '13' (DUI), '37' (Otro), etc.
	SupplierDocumentNumber    *string `json:"supplier_document_number,omitempty"`
	SupplierNRC               *string `json:"supplier_nrc,omitempty"`
	SupplierActivityCode      *string `json:"supplier_activity_code,omitempty"`
	SupplierActivityDesc      *string `json:"supplier_activity_desc,omitempty"`
	SupplierAddressDept       *string `json:"supplier_address_dept,omitempty"`
	SupplierAddressMuni       *string `json:"supplier_address_muni,omitempty"`
	SupplierAddressComplement *string `json:"supplier_address_complement,omitempty"`
	SupplierPhone             *string `json:"supplier_phone,omitempty"`
	SupplierEmail             *string `json:"supplier_email,omitempty"`

	// Financial totals
	Subtotal           float64 `json:"subtotal"`
	TotalDiscount      float64 `json:"total_discount"`
	DiscountPercentage float64 `json:"discount_percentage"`
	TotalTaxes         float64 `json:"total_taxes"`         // For regular purchases with IVA
	IVARetained        float64 `json:"iva_retained"`        // IVA retenido
	IncomeTaxRetained  float64 `json:"income_tax_retained"` // Retención de renta
	Total              float64 `json:"total"`
	Currency           string  `json:"currency"`

	// Payment information
	PaymentCondition *int       `json:"payment_condition,omitempty"` // 1=Contado, 2=Crédito, 3=Otro
	PaymentMethod    *string    `json:"payment_method,omitempty"`    // '01'=Efectivo, '02'=Cheque, etc.
	PaymentReference *string    `json:"payment_reference,omitempty"`
	PaymentTerm      *string    `json:"payment_term,omitempty"`   // '01', '02', '03'
	PaymentPeriod    *int       `json:"payment_period,omitempty"` // Period in days/months
	PaymentStatus    string     `json:"payment_status"`           // 'pending', 'paid', 'partial'
	AmountPaid       float64    `json:"amount_paid"`
	BalanceDue       float64    `json:"balance_due"`
	DueDate          *time.Time `json:"due_date,omitempty"`

	// DTE tracking
	DteNumeroControl    *string    `json:"dte_numero_control,omitempty"`
	DteStatus           *string    `json:"dte_status,omitempty"`
	DteHaciendaResponse *string    `json:"dte_hacienda_response,omitempty"`
	DteSelloRecibido    *string    `json:"dte_sello_recibido,omitempty"`
	DteSubmittedAt      *time.Time `json:"dte_submitted_at,omitempty"`
	DteType             string     `json:"dte_type"` // '14' for FSE, future types

	// Status
	Status string `json:"status"` // 'draft', 'finalized', 'voided'

	// Timestamps
	CreatedAt   time.Time  `json:"created_at"`
	FinalizedAt *time.Time `json:"finalized_at,omitempty"`
	VoidedAt    *time.Time `json:"voided_at,omitempty"`

	// Audit
	CreatedBy *string `json:"created_by,omitempty"`
	VoidedBy  *string `json:"voided_by,omitempty"`
	Notes     *string `json:"notes,omitempty"`

	// Relationships
	LineItems []PurchaseLineItem `json:"line_items,omitempty"`
}

// PurchaseLineItem represents a line item on a purchase
type PurchaseLineItem struct {
	ID         string `json:"id"`
	PurchaseID string `json:"purchase_id"`
	LineNumber int    `json:"line_number"`

	// Item reference (NULL for FSE free-form items)
	ItemID *string `json:"item_id,omitempty"`

	// Item information (required for FSE, can reference inventory for regular purchases)
	ItemCode        *string `json:"item_code,omitempty"`
	ItemName        string  `json:"item_name"`
	ItemDescription *string `json:"item_description,omitempty"`
	ItemType        int     `json:"item_type"`                // 1=Bien, 2=Servicio, 3=Ambos
	ItemTipoItem    *string `json:"item_tipo_item,omitempty"` // Maps to Hacienda tipo item codes
	UnitOfMeasure   int     `json:"unit_of_measure"`          // Hacienda unit codes

	// Pricing
	Quantity     float64 `json:"quantity"`
	UnitPrice    float64 `json:"unit_price"`
	LineSubtotal float64 `json:"line_subtotal"`

	// Discount
	DiscountPercentage float64 `json:"discount_percentage"`
	DiscountAmount     float64 `json:"discount_amount"`

	// Tax calculations
	TaxableAmount float64 `json:"taxable_amount"`
	TotalTaxes    float64 `json:"total_taxes"`
	LineTotal     float64 `json:"line_total"` // "compra" field in FSE

	CreatedAt time.Time `json:"created_at"`

	// Relationships
	Taxes []PurchaseLineItemTax `json:"taxes,omitempty"`
}

// PurchaseLineItemTax represents a tax on a purchase line item
type PurchaseLineItemTax struct {
	ID         string `json:"id"`
	LineItemID string `json:"line_item_id"`

	// Tax snapshot
	TributoCode string `json:"tributo_code"`
	TributoName string `json:"tributo_name"`

	// Calculation
	TaxRate     float64 `json:"tax_rate"` // 0.13 for 13% IVA
	TaxableBase float64 `json:"taxable_base"`
	TaxAmount   float64 `json:"tax_amount"`

	CreatedAt time.Time `json:"created_at"`
}

// SupplierInfo represents informal supplier information for FSE
type SupplierInfo struct {
	Name           string  `json:"name" binding:"required"`
	DocumentType   string  `json:"document_type" binding:"required"` // '36', '13', '37', etc.
	DocumentNumber string  `json:"document_number" binding:"required"`
	NRC            *string `json:"nrc,omitempty"`
	ActivityCode   string  `json:"activity_code" binding:"required"`
	ActivityDesc   string  `json:"activity_description" binding:"required"`
	Address        Address `json:"address" binding:"required"`
	Phone          *string `json:"phone,omitempty"`
	Email          *string `json:"email,omitempty"`
}

// Address represents an address (reusing pattern)
type Address struct {
	Department   string `json:"department" binding:"required"`
	Municipality string `json:"municipality" binding:"required"`
	Complement   string `json:"complement" binding:"required"`
}

// PaymentInfo represents payment information for FSE
type PaymentInfo struct {
	Condition int     `json:"condition" binding:"required"` // 1=Contado, 2=Crédito, 3=Otro
	Method    string  `json:"method" binding:"required"`    // '01', '02', etc.
	Reference *string `json:"reference,omitempty"`
	Term      *string `json:"term,omitempty"`   // '01', '02', '03'
	Period    *int    `json:"period,omitempty"` // For credit terms
}

// CreateFSERequest represents the request to create an FSE purchase
type CreateFSERequest struct {
	EstablishmentID    string                     `json:"establishment_id" binding:"required"`
	PointOfSaleID      string                     `json:"point_of_sale_id" binding:"required"`
	PurchaseDate       DateOnly                   `json:"purchase_date" binding:"required"`
	Supplier           SupplierInfo               `json:"supplier" binding:"required"`
	LineItems          []CreateFSELineItemRequest `json:"line_items" binding:"required,min=1"`
	Payment            PaymentInfo                `json:"payment" binding:"required"`
	DiscountPercentage float64                    `json:"discount_percentage"`
	IVARetained        float64                    `json:"iva_retained"`
	IncomeTaxRetained  float64                    `json:"income_tax_retained"`
	Notes              *string                    `json:"notes,omitempty"`
}

// Validate validates the create FSE request
func (r *CreateFSERequest) Validate() error {
	// Validate establishment and POS
	if strings.TrimSpace(r.EstablishmentID) == "" {
		return fmt.Errorf("establishment_id is required")
	}
	if strings.TrimSpace(r.PointOfSaleID) == "" {
		return fmt.Errorf("point_of_sale_id is required")
	}

	// Validate purchase date
	if r.PurchaseDate.IsZero() {
		return fmt.Errorf("purchase_date is required")
	}

	// Validate supplier
	if err := r.Supplier.Validate(); err != nil {
		return fmt.Errorf("supplier validation failed: %w", err)
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

	// Validate payment
	if err := r.Payment.Validate(); err != nil {
		return fmt.Errorf("payment validation failed: %w", err)
	}

	// Validate discount percentage
	if r.DiscountPercentage < 0 || r.DiscountPercentage > 100 {
		return fmt.Errorf("discount_percentage must be between 0 and 100")
	}

	// Validate retentions
	if r.IVARetained < 0 {
		return fmt.Errorf("iva_retained cannot be negative")
	}
	if r.IncomeTaxRetained < 0 {
		return fmt.Errorf("income_tax_retained cannot be negative")
	}

	return nil
}

// CreateFSELineItemRequest represents a line item in the create FSE request
type CreateFSELineItemRequest struct {
	ItemType       int     `json:"item_type" binding:"required"` // 1=Bien, 2=Servicio, 3=Ambos
	Code           *string `json:"code,omitempty"`
	Description    string  `json:"description" binding:"required"`
	Quantity       float64 `json:"quantity" binding:"required"`
	UnitOfMeasure  int     `json:"unit_of_measure" binding:"required"` // Hacienda unit code
	UnitPrice      float64 `json:"unit_price" binding:"required"`
	DiscountAmount float64 `json:"discount_amount"`
}

// Validate validates the create FSE line item request
func (r *CreateFSELineItemRequest) Validate() error {
	if r.ItemType < 1 || r.ItemType > 3 {
		return fmt.Errorf("item_type must be 1 (Bien), 2 (Servicio), or 3 (Ambos)")
	}
	if strings.TrimSpace(r.Description) == "" {
		return fmt.Errorf("description is required")
	}
	if r.Quantity <= 0 {
		return fmt.Errorf("quantity must be greater than 0")
	}
	if r.UnitOfMeasure <= 0 {
		return fmt.Errorf("unit_of_measure is required")
	}
	if r.UnitPrice < 0 {
		return fmt.Errorf("unit_price cannot be negative")
	}
	if r.DiscountAmount < 0 {
		return fmt.Errorf("discount_amount cannot be negative")
	}
	return nil
}

// Validate validates the supplier info
func (s *SupplierInfo) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("supplier name is required")
	}
	if strings.TrimSpace(s.DocumentType) == "" {
		return fmt.Errorf("supplier document_type is required")
	}
	if strings.TrimSpace(s.DocumentNumber) == "" {
		return fmt.Errorf("supplier document_number is required")
	}
	if strings.TrimSpace(s.ActivityCode) == "" {
		return fmt.Errorf("supplier activity_code is required")
	}
	if strings.TrimSpace(s.ActivityDesc) == "" {
		return fmt.Errorf("supplier activity_description is required")
	}

	// Validate document type
	validDocTypes := []string{"36", "13", "02", "03", "37"}
	if !contains(validDocTypes, s.DocumentType) {
		return fmt.Errorf("invalid document_type: must be one of %v", validDocTypes)
	}

	// Validate address
	if err := s.Address.Validate(); err != nil {
		return fmt.Errorf("address validation failed: %w", err)
	}

	return nil
}

// Validate validates the address
func (a *Address) Validate() error {
	if strings.TrimSpace(a.Department) == "" {
		return fmt.Errorf("department is required")
	}
	if strings.TrimSpace(a.Municipality) == "" {
		return fmt.Errorf("municipality is required")
	}
	if strings.TrimSpace(a.Complement) == "" {
		return fmt.Errorf("complement is required")
	}
	return nil
}

// Validate validates the payment info
func (p *PaymentInfo) Validate() error {
	if p.Condition < 1 || p.Condition > 3 {
		return fmt.Errorf("payment condition must be 1 (Contado), 2 (Crédito), or 3 (Otro)")
	}
	if strings.TrimSpace(p.Method) == "" {
		return fmt.Errorf("payment method is required")
	}

	// Validate payment method code
	if !codigos.IsValidPaymentMethod(p.Method) {
		return fmt.Errorf("invalid payment method code: %s", p.Method)
	}

	// If credit, require term and period
	if p.Condition == 2 {
		if p.Term == nil || *p.Term == "" {
			return fmt.Errorf("payment term is required for credit payments")
		}
		if p.Period == nil || *p.Period <= 0 {
			return fmt.Errorf("payment period is required for credit payments")
		}

		// Validate term
		validTerms := []string{"01", "02", "03"}
		if !contains(validTerms, *p.Term) {
			return fmt.Errorf("invalid payment term: must be one of %v", validTerms)
		}
	}

	return nil
}

// FinalizePurchaseRequest represents the request to finalize a purchase
type FinalizePurchaseRequest struct {
	// For FSE, typically no additional info needed
	// For regular purchases, might need payment confirmation
}

// Validate validates the finalize purchase request
func (r *FinalizePurchaseRequest) Validate() error {
	// No validation needed for FSE finalization
	return nil
}

// Helper methods

// IsFSE checks if this is an FSE purchase
func (p *Purchase) IsFSE() bool {
	return p.PurchaseType == "fse"
}

// IsRegular checks if this is a regular purchase
func (p *Purchase) IsRegular() bool {
	return p.PurchaseType == "regular"
}

// HasSupplierID checks if purchase references a registered supplier
func (p *Purchase) HasSupplierID() bool {
	return p.SupplierID != nil && *p.SupplierID != ""
}

// GetSupplierName returns the supplier name (from embedded info or referenced supplier)
func (p *Purchase) GetSupplierName() string {
	if p.SupplierName != nil {
		return *p.SupplierName
	}
	return ""
}

// IsPaid checks if the purchase is fully paid
func (p *Purchase) IsPaid() bool {
	return p.PaymentStatus == "paid"
}

// IsFinalized checks if the purchase is finalized
func (p *Purchase) IsFinalized() bool {
	return p.Status == "finalized"
}
