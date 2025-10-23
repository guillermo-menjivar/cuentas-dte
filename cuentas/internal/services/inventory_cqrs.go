package services

import (
	"context"
	"cuentas/internal/models"
)

// RecordPurchase adds inventory with cost tracking
func (s *InventoryService) RecordPurchase(
	ctx context.Context,
	companyID, itemID string,
	req *models.RecordPurchaseRequest,
) (*models.InventoryEvent, error)

// RecordAdjustment corrects inventory quantities
func (s *InventoryService) RecordAdjustment(
	ctx context.Context,
	companyID, itemID string,
	req *models.RecordAdjustmentRequest,
) (*models.InventoryEvent, error)

// GetInventoryState gets current quantity and moving average cost
func (s *InventoryService) GetInventoryState(
	ctx context.Context,
	companyID, itemID string,
) (*models.InventoryState, error)

// ListInventoryStates gets all inventory states for a company
func (s *InventoryService) ListInventoryStates(
	ctx context.Context,
	companyID string,
	inStockOnly bool,
) ([]models.InventoryState, error)

// GetCostHistory shows how cost changed over time
func (s *InventoryService) GetCostHistory(
	ctx context.Context,
	companyID, itemID string,
	limit int,
) ([]models.InventoryEvent, error)

// Helper: calculate moving average
func calculateMovingAverage(
	currentQty, currentTotal, addQty, addCost float64,
) (newQty, newTotal, newAvg float64)
