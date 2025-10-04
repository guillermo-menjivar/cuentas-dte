package models

import (
	"fmt"
	"time"
)

// InvoiceLineItem represents a line item on an invoice
type InvoiceLineItem struct {
	ID         string `json:"id"`
	InvoiceID  string `json:"invoice_id"`
	LineNumber int    `json:"line_number"`

	// Item reference
	ItemID *string `json:"item_id,omitempty"`

	// Item snapshot
	ItemSku         string  `json:"item_sku"`
	ItemName        string  `json:"item_name"`
	ItemDescription *string `json:"item_description,omitempty"`
	ItemTipoItem    string  `json:"item_tipo_item"`
	UnitOfMeasure   string  `json:"unit_of_measure"`

	// Pricing
	UnitPrice    float64 `json:"unit_price"`
	Quantity     float64 `json:"quantity"`
	LineSubtotal float64 `json:"line_subtotal"`

	// Discount
	DiscountPercentage float64 `json:"discount_percentage"`
	DiscountAmount     float64 `json:"discount_amount"`

	// Tax calculations
	TaxableAmount float64 `json:"taxable_amount"`
	TotalTaxes    float64 `json:"total_taxes"`
	LineTotal     float64 `json:"line_total"`

	CreatedAt time.Time `json:"created_at"`

	// Relationships
	Taxes []InvoiceLineItemTax `json:"taxes,omitempty"`
}

// CreateInvoiceLineItemRequest represents a line item in the create invoice request
type CreateInvoiceLineItemRequest struct {
	ItemID             string  `json:"item_id" binding:"required"`
	Quantity           float64 `json:"quantity" binding:"required"`
	DiscountPercentage float64 `json:"discount_percentage"`
}

// Validate validates the create invoice line item request
func (r *CreateInvoiceLineItemRequest) Validate() error {
	if r.ItemID == "" {
		return fmt.Errorf("item_id is required")
	}

	if r.Quantity <= 0 {
		return fmt.Errorf("quantity must be greater than 0")
	}

	if r.DiscountPercentage < 0 || r.DiscountPercentage > 100 {
		return fmt.Errorf("discount_percentage must be between 0 and 100")
	}

	return nil
}
