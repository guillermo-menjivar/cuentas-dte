package models

import (
	"fmt"
	"time"

	"cuentas/internal/codigos"
)

// InventoryItemTax represents a tax associated with an inventory item
type InventoryItemTax struct {
	ID          string    `json:"id"`
	ItemID      string    `json:"item_id"`
	TributoCode string    `json:"tributo_code"`
	CreatedAt   time.Time `json:"created_at"`
}

// AddItemTaxRequest represents a request to add a tax to an item
type AddItemTaxRequest struct {
	TributoCode string `json:"tributo_code" binding:"required"`
}

// Validate validates the add item tax request
func (r *AddItemTaxRequest) Validate() error {
	// Validate tributo code exists in our codigos package
	if !codigos.IsValidTributo(r.TributoCode) {
		return fmt.Errorf("invalid tributo_code: %s", r.TributoCode)
	}

	return nil
}
