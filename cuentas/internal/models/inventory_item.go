package models

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// InventoryItem represents an inventory item (product or service)
type InventoryItem struct {
	ID           string  `json:"id"`
	CompanyID    string  `json:"company_id"`
	TipoItem     string  `json:"tipo_item"` // 1=Bienes, 2=Servicios
	SKU          string  `json:"sku"`       // ← Always has value (not pointer)
	CodigoBarras *string `json:"codigo_barras,omitempty"`

	Name         string  `json:"name"`
	Description  *string `json:"description,omitempty"`
	Manufacturer *string `json:"manufacturer,omitempty"`
	ImageURL     *string `json:"image_url,omitempty"`

	// Pricing (tax-exclusive)
	CostPrice     *float64 `json:"cost_price,omitempty"`
	UnitPrice     float64  `json:"unit_price"`
	UnitOfMeasure string   `json:"unit_of_measure"`

	Color *string `json:"color,omitempty"`

	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships (loaded separately)
	Taxes       []InventoryItemTax `json:"taxes,omitempty"`
	IsTaxExempt bool               `json:"is_tax_exempt"`
}

// CreateInventoryItemRequest represents the request to create an inventory item
type CreateInventoryItemRequest struct {
	TipoItem     string  `json:"tipo_item" binding:"required"`
	SKU          *string `json:"sku"` // ← Optional (pointer)
	CodigoBarras *string `json:"codigo_barras"`

	Name         string  `json:"name" binding:"required"`
	Description  *string `json:"description"`
	Manufacturer *string `json:"manufacturer"`
	ImageURL     *string `json:"image_url"`

	CostPrice     *float64 `json:"cost_price"`
	UnitPrice     float64  `json:"unit_price" binding:"required"`
	UnitOfMeasure string   `json:"unit_of_measure" binding:"required"`
	Color         *string  `json:"color"`

	// Taxes to associate with the item (optional - will use defaults if empty)
	Taxes       []AddItemTaxRequest `json:"taxes"`
	IsTaxExempt *bool               `json:"is_tax_exempt"`
}

// UpdateInventoryItemRequest represents the request to update an inventory item
type UpdateInventoryItemRequest struct {
	Name         *string `json:"name"`
	Description  *string `json:"description"`
	Manufacturer *string `json:"manufacturer"`
	ImageURL     *string `json:"image_url"`

	CostPrice     *float64 `json:"cost_price"`
	UnitPrice     *float64 `json:"unit_price"`
	UnitOfMeasure *string  `json:"unit_of_measure"`
	Color         *string  `json:"color"`
	IsTaxExempt   *bool    `json:"is_tax_exempt"`
}

// Valid units of measure
var validUnitsOfMeasure = []string{
	"unidad", "servicio", "hora", "kilogramo", "gramo",
	"litro", "mililitro", "metro", "centimetro", "caja",
}

// SKU validation pattern: alphanumeric with dashes, 3-50 chars
var skuPattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9-]{1,48}[A-Z0-9]$`)

// Validate validates the create inventory item request
func (r *CreateInventoryItemRequest) Validate() error {
	// Validate tipo_item (only 1 and 2 for now)
	if r.TipoItem != "1" && r.TipoItem != "2" {
		return fmt.Errorf("tipo_item must be 1 (Bienes) or 2 (Servicios)")
	}

	// Validate and normalize SKU if provided
	if r.SKU != nil {
		normalized := strings.ToUpper(strings.TrimSpace(*r.SKU))
		if normalized != "" {
			if !skuPattern.MatchString(normalized) {
				return fmt.Errorf("sku must be 3-50 alphanumeric characters with dashes, starting and ending with alphanumeric (e.g., PROD-001)")
			}
			r.SKU = &normalized
		} else {
			// Empty string provided, treat as nil
			r.SKU = nil
		}
	}
	// If r.SKU is nil, service will auto-generate it

	// Validate name
	if strings.TrimSpace(r.Name) == "" {
		return fmt.Errorf("name is required")
	}

	// Validate unit_of_measure
	r.UnitOfMeasure = strings.ToLower(strings.TrimSpace(r.UnitOfMeasure))
	if !contains(validUnitsOfMeasure, r.UnitOfMeasure) {
		return fmt.Errorf("invalid unit_of_measure: must be one of %v", validUnitsOfMeasure)
	}

	// Validate prices
	if r.UnitPrice < 0 {
		return fmt.Errorf("unit_price cannot be negative")
	}
	if r.CostPrice != nil && *r.CostPrice < 0 {
		return fmt.Errorf("cost_price cannot be negative")
	}

	// Validate taxes if provided
	for i, tax := range r.Taxes {
		if err := tax.Validate(); err != nil {
			return fmt.Errorf("tax %d: %w", i+1, err)
		}
	}

	return nil
}

// Validate validates the update inventory item request
func (r *UpdateInventoryItemRequest) Validate() error {
	// Validate unit_of_measure if provided
	if r.UnitOfMeasure != nil {
		normalized := strings.ToLower(strings.TrimSpace(*r.UnitOfMeasure))
		if !contains(validUnitsOfMeasure, normalized) {
			return fmt.Errorf("invalid unit_of_measure: must be one of %v", validUnitsOfMeasure)
		}
		r.UnitOfMeasure = &normalized
	}

	// Validate prices if provided
	if r.UnitPrice != nil && *r.UnitPrice < 0 {
		return fmt.Errorf("unit_price cannot be negative")
	}
	if r.CostPrice != nil && *r.CostPrice < 0 {
		return fmt.Errorf("cost_price cannot be negative")
	}

	return nil
}

// Helper function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
