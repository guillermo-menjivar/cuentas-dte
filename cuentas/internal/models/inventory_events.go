package models

import (
	"encoding/json"
	"fmt"
	"time"
)

type InventoryEvent struct {
	EventID               int64
	CompanyID             string
	ItemID                string
	EventType             string
	EventTimestamp        time.Time
	AggregateVersion      int
	Quantity              float64
	UnitCost              float64
	TotalCost             float64
	BalanceQuantityAfter  float64
	BalanceTotalCostAfter float64
	MovingAvgCostBefore   float64
	MovingAvgCostAfter    float64
	ReferenceType         *string
	ReferenceID           *string
	CorrelationID         *string
	EventData             json.RawMessage
	Notes                 *string
	CreatedByUserID       *string
	CreatedAt             time.Time
}

type InventoryState struct {
	CompanyID        string
	ItemID           string
	CurrentQuantity  float64
	CurrentTotalCost float64
	CurrentAvgCost   float64
	LastEventID      *int64
	AggregateVersion int
	UpdatedAt        time.Time
}

type RecordPurchaseRequest struct {
	Quantity      float64 `json:"quantity" binding:"required"`
	UnitCost      float64 `json:"unit_cost" binding:"required"`
	ReferenceType *string `json:"reference_type"`
	ReferenceID   *string `json:"reference_id"`
	Notes         *string `json:"notes"`
}

type RecordAdjustmentRequest struct {
	Quantity      float64  `json:"quantity" binding:"required"` // Can be + or -
	Reason        string   `json:"reason" binding:"required"`
	UnitCost      *float64 `json:"unit_cost"` // Required if adding qty
	ReferenceType *string  `json:"reference_type"`
	ReferenceID   *string  `json:"reference_id"`
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
