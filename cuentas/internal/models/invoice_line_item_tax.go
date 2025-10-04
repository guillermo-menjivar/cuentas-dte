package models

import "time"

// InvoiceLineItemTax represents a tax on a line item
type InvoiceLineItemTax struct {
	ID         string `json:"id"`
	LineItemID string `json:"line_item_id"`

	// Tax snapshot
	TributoCode string `json:"tributo_code"`
	TributoName string `json:"tributo_name"`

	// Calculation
	TaxRate     float64 `json:"tax_rate"`
	TaxableBase float64 `json:"taxable_base"`
	TaxAmount   float64 `json:"tax_amount"`

	CreatedAt time.Time `json:"created_at"`
}
