// internal/models/nota.go

package models

type CreateNotaDebitoRequest struct {
	CCFIds       []string                    `json:"ccf_ids" binding:"required,min=1,max=50"`
	LineItems    []CreateNotaLineItemRequest `json:"line_items" binding:"required,min=1,max=2000"`
	PaymentTerms string                      `json:"payment_terms"`
	Notes        string                      `json:"notes"`
}

type CreateNotaLineItemRequest struct {
	// Which CCF this line adjusts
	RelatedCCFId string `json:"related_ccf_id" binding:"required"`

	// ⭐ NEW: Explicit flag
	IsNewItem bool `json:"is_new_item" binding:"required"`

	// For ADJUSTING existing CCF line item (is_new_item = false)
	CCFLineItemId    *string `json:"ccf_line_item_id,omitempty"`
	AdjustmentAmount float64 `json:"adjustment_amount,omitempty"`

	// For ADDING new item (is_new_item = true)
	InventoryItemId *string `json:"inventory_item_id,omitempty"`
	ItemName        string  `json:"item_name" binding:"required"`
	Quantity        float64 `json:"quantity" binding:"required,gt=0"`
	UnitPrice       float64 `json:"unit_price" binding:"required,gte=0"`
	DiscountAmount  float64 `json:"discount_amount"`
}
