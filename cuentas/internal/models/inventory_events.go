package models

import (
	"encoding/json"
	"fmt"
	"time"
)

type CompanyLegalInfo struct {
	LegalName string `json:"legal_name" db:"nombre_comercial"`
	NIT       string `json:"nit" db:"nit"`
	NRC       string `json:"nrc" db:"ncr"`
}

type InventoryEvent struct {
	EventID               int64           `json:"event_id"`
	CompanyID             string          `json:"company_id"`
	ItemID                string          `json:"item_id"`
	EventType             string          `json:"event_type"`
	EventTimestamp        time.Time       `json:"event_timestamp"`
	AggregateVersion      int             `json:"aggregate_version"`
	Quantity              float64         `json:"quantity"`
	UnitCost              Money           `json:"unit_cost"`
	TotalCost             Money           `json:"total_cost"`
	BalanceQuantityAfter  float64         `json:"balance_quantity_after"`
	BalanceTotalCostAfter Money           `json:"balance_total_cost_after"`
	MovingAvgCostBefore   Money           `json:"moving_avg_cost_before"`
	MovingAvgCostAfter    Money           `json:"moving_avg_cost_after"`
	ReferenceType         *string         `json:"reference_type,omitempty"`
	ReferenceID           *string         `json:"reference_id,omitempty"`
	CorrelationID         *string         `json:"correlation_id,omitempty"`
	EventData             json.RawMessage `json:"event_data,omitempty"`
	Notes                 *string         `json:"notes,omitempty"`
	CreatedByUserID       *string         `json:"created_by_user_id,omitempty"`
	CreatedAt             time.Time       `json:"created_at"`
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
	Quantity      float64 `json:"quantity" binding:"required,gt=0"`
	UnitCost      Money   `json:"unit_cost" binding:"required,gt=0"`
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
