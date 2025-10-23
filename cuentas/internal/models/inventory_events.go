package models

import (
	"encoding/json"
	"fmt"
	"time"
)

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
	CurrentTotalCost float64   `json:"current_total_cost"`
	CurrentAvgCost   float64   `json:"current_avg_cost"`
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
	Limit int `form:"limit"` // Default will be 50
}
