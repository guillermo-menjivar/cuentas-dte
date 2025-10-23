package models

import (
	"encoding/json"
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

func (r *RecordPurchaseRequest) Validate() error
func (r *RecordAdjustmentRequest) Validate() error
