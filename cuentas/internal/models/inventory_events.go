package models

import (
	"cuentas/internal/codigos"
	"cuentas/internal/tools"
	"encoding/json"
	"fmt"
	"regexp"
	"time"
)

type CompanyLegalInfo struct {
	LegalName string `json:"legal_name" db:"nombre_comercial"`
	NIT       string `json:"nit" db:"nit"`
	NRC       string `json:"nrc" db:"ncr"`
}

type InventoryEvent struct {
	EventID               int64     `json:"event_id"`
	CompanyID             string    `json:"company_id"`
	ItemID                string    `json:"item_id"`
	EventType             string    `json:"event_type"`
	EventTimestamp        time.Time `json:"event_timestamp"`
	AggregateVersion      int       `json:"aggregate_version"`
	Quantity              float64   `json:"quantity"`
	UnitCost              Money     `json:"unit_cost"`
	TotalCost             Money     `json:"total_cost"`
	BalanceQuantityAfter  float64   `json:"balance_quantity_after"`
	BalanceTotalCostAfter Money     `json:"balance_total_cost_after"`
	MovingAvgCostBefore   Money     `json:"moving_avg_cost_before"`
	MovingAvgCostAfter    Money     `json:"moving_avg_cost_after"`

	// Purchase/Document fields
	DocumentType        *string `json:"document_type,omitempty"`
	DocumentNumber      *string `json:"document_number,omitempty"`
	SupplierName        *string `json:"supplier_name,omitempty"`
	SupplierNIT         *string `json:"supplier_nit,omitempty"`
	SupplierNationality *string `json:"supplier_nationality,omitempty"`
	CostSourceRef       *string `json:"cost_source_ref,omitempty"`

	// Sales fields
	SalePrice         *Money   `json:"sale_price,omitempty"`
	DiscountAmount    *Money   `json:"discount_amount,omitempty"`
	NetSalePrice      *Money   `json:"net_sale_price,omitempty"`
	TaxExempt         *bool    `json:"tax_exempt,omitempty"`
	TaxRate           *float64 `json:"tax_rate,omitempty"`
	TaxAmount         *Money   `json:"tax_amount,omitempty"`
	InvoiceID         *string  `json:"invoice_id,omitempty"`
	InvoiceLineID     *string  `json:"invoice_line_id,omitempty"`
	CustomerName      *string  `json:"customer_name,omitempty"`
	CustomerNIT       *string  `json:"customer_nit,omitempty"`
	CustomerTaxExempt *bool    `json:"customer_tax_exempt,omitempty"`

	// Common fields
	ReferenceType   *string         `json:"reference_type,omitempty"`
	ReferenceID     *string         `json:"reference_id,omitempty"`
	CorrelationID   *string         `json:"correlation_id,omitempty"`
	EventData       json.RawMessage `json:"event_data,omitempty"`
	Notes           *string         `json:"notes,omitempty"`
	CreatedByUserID *string         `json:"created_by_user_id,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

// InventoryState represents the current state of inventory for an item
type InventoryState struct {
	CompanyID        string    `json:"company_id"`
	ItemID           string    `json:"item_id"`
	CurrentQuantity  float64   `json:"current_quantity"`
	CurrentTotalCost Money     `json:"current_total_cost"`
	CurrentAvgCost   Money     `json:"current_avg_cost"`
	LastEventID      *int64    `json:"last_event_id,omitempty"`
	AggregateVersion int       `json:"aggregate_version"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// InventoryStateWithItem includes item details with the state
type InventoryStateWithItem struct {
	CompanyID        string    `json:"company_id"`
	ItemID           string    `json:"item_id"`
	SKU              string    `json:"sku"`
	ItemName         string    `json:"item_name"`
	TipoItem         string    `json:"tipo_item"`
	CurrentQuantity  float64   `json:"current_quantity"`
	CurrentTotalCost Money     `json:"current_total_cost"` // Changed from float64 to Money
	CurrentAvgCost   Money     `json:"current_avg_cost"`   // Changed from float64 to Money
	LastEventID      *int64    `json:"last_event_id,omitempty"`
	AggregateVersion int       `json:"aggregate_version"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// RecordPurchaseRequest represents a request to record a purchase
type RecordPurchaseRequest struct {
	Quantity float64 `json:"quantity" binding:"required,gt=0"`
	UnitCost Money   `json:"unit_cost" binding:"required,gt=0"`

	// Legal compliance fields (Article 142-A)
	DocumentType   string  `json:"document_type" binding:"required"`
	DocumentNumber string  `json:"document_number" binding:"required"`
	SupplierName   string  `json:"supplier_name" binding:"required"`
	SupplierNIT    *string `json:"supplier_nit"` // Required if DocumentType == CCF (03)
	CostSourceRef  *string `json:"cost_source_ref"`

	// Existing optional fields
	Notes         *string `json:"notes"`
	ReferenceType *string `json:"reference_type"`
	ReferenceID   *string `json:"reference_id"`
	CorrelationID *string `json:"correlation_id"`
}

// RecordAdjustmentRequest represents a request to record an inventory adjustment
type RecordAdjustmentRequest struct {
	Quantity      float64 `json:"quantity" binding:"required,ne=0"`
	UnitCost      *Money  `json:"unit_cost"`
	Reason        string  `json:"reason" binding:"required"`
	ReferenceType *string `json:"reference_type"`
	ReferenceID   *string `json:"reference_id"`
	CorrelationID *string `json:"correlation_id"`
}

func (r *RecordAdjustmentRequest) Validate() error {
	if r.Quantity == 0 {
		return fmt.Errorf("quantity cannot be zero")
	}

	// If adding inventory (positive quantity), unit cost is required
	if r.Quantity > 0 {
		if r.UnitCost == nil {
			return fmt.Errorf("unit_cost is required when adding inventory")
		}
		if *r.UnitCost < 0 {
			return fmt.Errorf("unit_cost cannot be negative")
		}
	}

	// If removing inventory (negative quantity), unit cost not needed
	// Will use current moving average cost

	if r.Reason == "" {
		return fmt.Errorf("reason is required for adjustments")
	}

	return nil
}

func (r *RecordPurchaseRequest) Validate() error {
	if r.Quantity <= 0 {
		return fmt.Errorf("quantity must be greater than 0")
	}

	if r.UnitCost < 0 {
		return fmt.Errorf("unit_cost cannot be negative")
	}

	// Validate document type using constants
	if r.DocumentType != codigos.DocTypeFactura && r.DocumentType != codigos.DocTypeComprobanteCredito {
		return fmt.Errorf("document_type must be %s (Factura) or %s (CCF)",
			codigos.DocTypeFactura, codigos.DocTypeComprobanteCredito)
	}

	// CCF (03) requires supplier NIT
	if r.DocumentType == codigos.DocTypeComprobanteCredito {
		if r.SupplierNIT == nil || *r.SupplierNIT == "" {
			return fmt.Errorf("supplier_nit is required for document type %s (CCF)",
				codigos.DocTypeComprobanteCredito)
		}
		// Optional: Validate NIT format
		if !tools.ValidateNIT(*r.SupplierNIT) {
			return fmt.Errorf("invalid supplier_nit format, must be XXXX-XXXXXX-XXX-X")
		}
	}

	if !validateNumeroControl(r.DocumentNumber) {
		return fmt.Errorf("document_number must be a valid DTE numero de control format (e.g., DTE-03-M001P001-000000000000001)")
	}

	if r.DocumentNumber == "" {
		return fmt.Errorf("document_number is required")
	}

	if r.SupplierName == "" {
		return fmt.Errorf("supplier_name is required")
	}

	return nil
}

type RecordSaleRequest struct {
	// Inventory
	Quantity float64 `json:"quantity" binding:"required"`

	// Pricing (all amounts are per unit, tax-exclusive unless noted)
	UnitSalePrice  Money  `json:"unit_sale_price" binding:"required"`
	DiscountAmount *Money `json:"discount_amount"`
	NetUnitPrice   Money  `json:"net_unit_price" binding:"required"`

	// Tax
	TaxExempt bool    `json:"tax_exempt"`
	TaxRate   float64 `json:"tax_rate"`
	TaxAmount Money   `json:"tax_amount"`

	// Document (reuses existing columns)
	DocumentType   string `json:"document_type" binding:"required"`   // "01" or "03"
	DocumentNumber string `json:"document_number" binding:"required"` // DTE numero de control

	// References
	InvoiceID     string `json:"invoice_id" binding:"required"`
	InvoiceLineID string `json:"invoice_line_id" binding:"required"`

	// Customer
	CustomerName      string  `json:"customer_name" binding:"required"`
	CustomerNIT       *string `json:"customer_nit"`
	CustomerTaxExempt bool    `json:"customer_tax_exempt"`

	// Optional
	Notes *string `json:"notes"`
}

// Validate validates the record sale request
func (r *RecordSaleRequest) Validate() error {
	if r.Quantity <= 0 {
		return fmt.Errorf("la cantidad debe ser mayor que 0")
	}

	if r.UnitSalePrice.Float64() < 0 {
		return fmt.Errorf("el precio de venta no puede ser negativo")
	}

	if r.NetUnitPrice.Float64() < 0 {
		return fmt.Errorf("el precio neto no puede ser negativo")
	}

	// Validate document type
	if r.DocumentType != codigos.DocTypeFactura && r.DocumentType != codigos.DocTypeComprobanteCredito {
		return fmt.Errorf("document_type debe ser %s (Factura) o %s (CCF)",
			codigos.DocTypeFactura, codigos.DocTypeComprobanteCredito)
	}

	if r.DocumentNumber == "" {
		return fmt.Errorf("el número de documento es requerido")
	}

	if r.InvoiceID == "" {
		return fmt.Errorf("el ID de factura es requerido")
	}

	if r.InvoiceLineID == "" {
		return fmt.Errorf("el ID de línea de factura es requerido")
	}

	if r.CustomerName == "" {
		return fmt.Errorf("el nombre del cliente es requerido")
	}

	// Validate tax logic
	if !r.TaxExempt && r.TaxRate <= 0 {
		return fmt.Errorf("la tasa de impuesto debe ser mayor que 0 para ventas gravadas")
	}

	if r.TaxExempt && r.TaxAmount.Float64() != 0 {
		return fmt.Errorf("el monto de impuesto debe ser 0 para ventas exentas")
	}

	if !validateNumeroControl(r.DocumentNumber) {
		return fmt.Errorf("document_number must be a valid DTE numero de control format (e.g., DTE-03-M001P001-000000000000001)")
	}

	return nil
}

// GetCostHistoryRequest represents query parameters for cost history
type GetCostHistoryRequest struct {
	Limit int    `form:"limit"` // Default will be 50
	Sort  string `form:"sort"`
}

// InventoryEventWithItem includes item details with the event
type InventoryEventWithItem struct {
	InventoryEvent
	SKU      string `json:"sku"`
	ItemName string `json:"item_name"`
}

// InventoryValuation represents inventory value at a specific point in time
type InventoryValuation struct {
	AsOfDate      time.Time       `json:"as_of_date"`
	CompanyID     string          `json:"company_id"`
	TotalValue    Money           `json:"total_value"`
	TotalQuantity float64         `json:"total_quantity"`
	ItemCount     int             `json:"item_count"`
	ItemValues    []ItemValuation `json:"item_values"`
}

// ItemValuation represents a single item's valuation at a point in time
type ItemValuation struct {
	ItemID      string    `json:"item_id"`
	SKU         string    `json:"sku"`
	ItemName    string    `json:"item_name"`
	Quantity    float64   `json:"quantity"`
	AvgCost     Money     `json:"avg_cost"`
	TotalValue  Money     `json:"total_value"`
	LastEventID int64     `json:"last_event_id"`
	LastEventAt time.Time `json:"last_event_at"`
}

func validateNumeroControl(numeroControl string) bool {
	var numeroControlRegex = regexp.MustCompile(`^DTE-[0-9]{2}-[MSBP][0-9]{3}P[0-9]{3}-[0-9]{15}$`)
	if numeroControl == "" {
		return false
	}
	return numeroControlRegex.MatchString(numeroControl)
}
