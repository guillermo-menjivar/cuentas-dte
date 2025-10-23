package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"cuentas/internal/models"
)

// RecordPurchase adds inventory with cost tracking via event sourcing
func (s *InventoryService) RecordPurchase(
	ctx context.Context,
	companyID, itemID string,
	req *models.RecordPurchaseRequest,
) (*models.InventoryEvent, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Verify item exists and belongs to company
	_, err := s.GetItemByID(ctx, companyID, itemID)
	if err != nil {
		return nil, fmt.Errorf("item not found: %w", err)
	}

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get or create current state
	currentState, err := s.getOrCreateInventoryStateTx(ctx, tx, companyID, itemID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current state: %w", err)
	}

	// Calculate new values
	newQuantity := currentState.CurrentQuantity + req.Quantity
	newTotalCost := currentState.CurrentTotalCost + (req.Quantity * req.UnitCost)
	newAvgCost := float64(0)
	if newQuantity > 0 {
		newAvgCost = newTotalCost / newQuantity
	}

	nextVersion := currentState.AggregateVersion + 1

	// Build event data
	eventData := map[string]interface{}{
		"quantity":   req.Quantity,
		"unit_cost":  req.UnitCost,
		"total_cost": req.Quantity * req.UnitCost,
	}
	if req.ReferenceType != nil {
		eventData["reference_type"] = *req.ReferenceType
	}
	if req.ReferenceID != nil {
		eventData["reference_id"] = *req.ReferenceID
	}
	if req.Notes != nil {
		eventData["notes"] = *req.Notes
	}

	eventDataJSON, err := json.Marshal(eventData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event data: %w", err)
	}

	// Insert event
	eventQuery := `
		INSERT INTO inventory_events (
			company_id, item_id, event_type, event_timestamp,
			aggregate_version, quantity, unit_cost, total_cost,
			balance_quantity_after, balance_total_cost_after,
			moving_avg_cost_before, moving_avg_cost_after,
			reference_type, reference_id, correlation_id,
			event_data, notes, created_by_user_id, created_at
		) VALUES (
			$1, $2, $3, NOW(),
			$4, $5, $6, $7,
			$8, $9,
			$10, $11,
			$12, $13, $14,
			$15, $16, $17, NOW()
		)
		RETURNING event_id, company_id, item_id, event_type, event_timestamp,
				  aggregate_version, quantity, unit_cost, total_cost,
				  balance_quantity_after, balance_total_cost_after,
				  moving_avg_cost_before, moving_avg_cost_after,
				  reference_type, reference_id, correlation_id,
				  event_data, notes, created_by_user_id, created_at
	`

	var event models.InventoryEvent
	err = tx.QueryRowContext(ctx, eventQuery,
		companyID, itemID, "PURCHASE",
		nextVersion, req.Quantity, req.UnitCost, req.Quantity*req.UnitCost,
		newQuantity, newTotalCost,
		currentState.CurrentAvgCost, newAvgCost,
		req.ReferenceType, req.ReferenceID, nil, // correlation_id nil for now
		eventDataJSON, req.Notes, nil, // created_by_user_id nil (placeholder)
	).Scan(
		&event.EventID, &event.CompanyID, &event.ItemID, &event.EventType, &event.EventTimestamp,
		&event.AggregateVersion, &event.Quantity, &event.UnitCost, &event.TotalCost,
		&event.BalanceQuantityAfter, &event.BalanceTotalCostAfter,
		&event.MovingAvgCostBefore, &event.MovingAvgCostAfter,
		&event.ReferenceType, &event.ReferenceID, &event.CorrelationID,
		&event.EventData, &event.Notes, &event.CreatedByUserID, &event.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert event: %w", err)
	}

	// Update state
	err = s.updateInventoryStateTx(ctx, tx, companyID, itemID, newQuantity, newTotalCost, event.EventID, nextVersion, currentState.AggregateVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to update state: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &event, nil
}

// RecordAdjustment corrects inventory quantities (add or remove)
func (s *InventoryService) RecordAdjustment(
	ctx context.Context,
	companyID, itemID string,
	req *models.RecordAdjustmentRequest,
) (*models.InventoryEvent, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Verify item exists
	_, err := s.GetItemByID(ctx, companyID, itemID)
	if err != nil {
		return nil, fmt.Errorf("item not found: %w", err)
	}

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get current state
	currentState, err := s.getOrCreateInventoryStateTx(ctx, tx, companyID, itemID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current state: %w", err)
	}

	// Determine unit cost
	var unitCost float64
	if req.Quantity > 0 {
		// Adding inventory - use provided cost
		if req.UnitCost == nil {
			return nil, fmt.Errorf("unit_cost required when adding inventory")
		}
		unitCost = *req.UnitCost
	} else {
		// Removing inventory - use current moving average
		unitCost = currentState.CurrentAvgCost
	}

	// Calculate new values
	newQuantity := currentState.CurrentQuantity + req.Quantity
	if newQuantity < 0 {
		return nil, fmt.Errorf("adjustment would result in negative quantity (current: %.2f, adjustment: %.2f)", currentState.CurrentQuantity, req.Quantity)
	}

	newTotalCost := currentState.CurrentTotalCost + (req.Quantity * unitCost)
	if newTotalCost < 0 {
		newTotalCost = 0 // Safety check
	}

	newAvgCost := float64(0)
	if newQuantity > 0 {
		newAvgCost = newTotalCost / newQuantity
	}

	nextVersion := currentState.AggregateVersion + 1

	// Build event data
	eventData := map[string]interface{}{
		"quantity":   req.Quantity,
		"unit_cost":  unitCost,
		"total_cost": req.Quantity * unitCost,
		"reason":     req.Reason,
	}
	if req.ReferenceType != nil {
		eventData["reference_type"] = *req.ReferenceType
	}
	if req.ReferenceID != nil {
		eventData["reference_id"] = *req.ReferenceID
	}

	eventDataJSON, err := json.Marshal(eventData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event data: %w", err)
	}

	// Insert event
	eventQuery := `
		INSERT INTO inventory_events (
			company_id, item_id, event_type, event_timestamp,
			aggregate_version, quantity, unit_cost, total_cost,
			balance_quantity_after, balance_total_cost_after,
			moving_avg_cost_before, moving_avg_cost_after,
			reference_type, reference_id, correlation_id,
			event_data, notes, created_by_user_id, created_at
		) VALUES (
			$1, $2, $3, NOW(),
			$4, $5, $6, $7,
			$8, $9,
			$10, $11,
			$12, $13, $14,
			$15, $16, $17, NOW()
		)
		RETURNING event_id, company_id, item_id, event_type, event_timestamp,
				  aggregate_version, quantity, unit_cost, total_cost,
				  balance_quantity_after, balance_total_cost_after,
				  moving_avg_cost_before, moving_avg_cost_after,
				  reference_type, reference_id, correlation_id,
				  event_data, notes, created_by_user_id, created_at
	`

	var event models.InventoryEvent
	err = tx.QueryRowContext(ctx, eventQuery,
		companyID, itemID, "ADJUSTMENT",
		nextVersion, req.Quantity, unitCost, req.Quantity*unitCost,
		newQuantity, newTotalCost,
		currentState.CurrentAvgCost, newAvgCost,
		req.ReferenceType, req.ReferenceID, nil,
		eventDataJSON, req.Reason, nil,
	).Scan(
		&event.EventID, &event.CompanyID, &event.ItemID, &event.EventType, &event.EventTimestamp,
		&event.AggregateVersion, &event.Quantity, &event.UnitCost, &event.TotalCost,
		&event.BalanceQuantityAfter, &event.BalanceTotalCostAfter,
		&event.MovingAvgCostBefore, &event.MovingAvgCostAfter,
		&event.ReferenceType, &event.ReferenceID, &event.CorrelationID,
		&event.EventData, &event.Notes, &event.CreatedByUserID, &event.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert event: %w", err)
	}

	// Update state
	err = s.updateInventoryStateTx(ctx, tx, companyID, itemID, newQuantity, newTotalCost, event.EventID, nextVersion, currentState.AggregateVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to update state: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &event, nil
}

// GetInventoryState retrieves the current inventory state for an item
func (s *InventoryService) GetInventoryState(
	ctx context.Context,
	companyID, itemID string,
) (*models.InventoryState, error) {
	// Verify item exists
	_, err := s.GetItemByID(ctx, companyID, itemID)
	if err != nil {
		return nil, fmt.Errorf("item not found: %w", err)
	}

	query := `
		SELECT company_id, item_id, current_quantity, current_total_cost,
			   current_avg_cost, last_event_id, aggregate_version, updated_at
		FROM inventory_state
		WHERE company_id = $1 AND item_id = $2
	`

	var state models.InventoryState
	err = s.db.QueryRowContext(ctx, query, companyID, itemID).Scan(
		&state.CompanyID, &state.ItemID, &state.CurrentQuantity, &state.CurrentTotalCost,
		&state.CurrentAvgCost, &state.LastEventID, &state.AggregateVersion, &state.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			// No state yet - return zero state
			return &models.InventoryState{
				CompanyID:        companyID,
				ItemID:           itemID,
				CurrentQuantity:  0,
				CurrentTotalCost: 0,
				CurrentAvgCost:   0,
				AggregateVersion: 0,
			}, nil
		}
		return nil, fmt.Errorf("failed to get inventory state: %w", err)
	}

	return &state, nil
}

// ListInventoryStates retrieves all inventory states for a company
func (s *InventoryService) ListInventoryStates(
	ctx context.Context,
	companyID string,
	inStockOnly bool,
) ([]models.InventoryState, error) {
	query := `
		SELECT company_id, item_id, current_quantity, current_total_cost,
			   current_avg_cost, last_event_id, aggregate_version, updated_at
		FROM inventory_state
		WHERE company_id = $1
	`

	if inStockOnly {
		query += " AND current_quantity > 0"
	}

	query += " ORDER BY updated_at DESC"

	rows, err := s.db.QueryContext(ctx, query, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to list inventory states: %w", err)
	}
	defer rows.Close()

	var states []models.InventoryState
	for rows.Next() {
		var state models.InventoryState
		err := rows.Scan(
			&state.CompanyID, &state.ItemID, &state.CurrentQuantity, &state.CurrentTotalCost,
			&state.CurrentAvgCost, &state.LastEventID, &state.AggregateVersion, &state.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan state: %w", err)
		}
		states = append(states, state)
	}

	if states == nil {
		states = []models.InventoryState{}
	}

	return states, nil
}

// GetCostHistory retrieves the event history showing cost changes over time
func (s *InventoryService) GetCostHistory(
	ctx context.Context,
	companyID, itemID string,
	limit int,
) ([]models.InventoryEvent, error) {
	// Verify item exists
	_, err := s.GetItemByID(ctx, companyID, itemID)
	if err != nil {
		return nil, fmt.Errorf("item not found: %w", err)
	}

	if limit <= 0 {
		limit = 50 // Default limit
	}
	if limit > 1000 {
		limit = 1000 // Max limit
	}

	query := `
		SELECT event_id, company_id, item_id, event_type, event_timestamp,
			   aggregate_version, quantity, unit_cost, total_cost,
			   balance_quantity_after, balance_total_cost_after,
			   moving_avg_cost_before, moving_avg_cost_after,
			   reference_type, reference_id, correlation_id,
			   event_data, notes, created_by_user_id, created_at
		FROM inventory_events
		WHERE company_id = $1 AND item_id = $2
		ORDER BY created_at DESC
		LIMIT $3
	`

	rows, err := s.db.QueryContext(ctx, query, companyID, itemID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get cost history: %w", err)
	}
	defer rows.Close()

	var events []models.InventoryEvent
	for rows.Next() {
		var event models.InventoryEvent
		err := rows.Scan(
			&event.EventID, &event.CompanyID, &event.ItemID, &event.EventType, &event.EventTimestamp,
			&event.AggregateVersion, &event.Quantity, &event.UnitCost, &event.TotalCost,
			&event.BalanceQuantityAfter, &event.BalanceTotalCostAfter,
			&event.MovingAvgCostBefore, &event.MovingAvgCostAfter,
			&event.ReferenceType, &event.ReferenceID, &event.CorrelationID,
			&event.EventData, &event.Notes, &event.CreatedByUserID, &event.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, event)
	}

	if events == nil {
		events = []models.InventoryEvent{}
	}

	return events, nil
}

// ========================================
// Helper Functions
// ========================================

// getOrCreateInventoryStateTx gets existing state or creates initial zero state within transaction
func (s *InventoryService) getOrCreateInventoryStateTx(
	ctx context.Context,
	tx *sql.Tx,
	companyID, itemID string,
) (*models.InventoryState, error) {
	query := `
		SELECT company_id, item_id, current_quantity, current_total_cost,
			   current_avg_cost, last_event_id, aggregate_version, updated_at
		FROM inventory_state
		WHERE company_id = $1 AND item_id = $2
		FOR UPDATE
	`

	var state models.InventoryState
	err := tx.QueryRowContext(ctx, query, companyID, itemID).Scan(
		&state.CompanyID, &state.ItemID, &state.CurrentQuantity, &state.CurrentTotalCost,
		&state.CurrentAvgCost, &state.LastEventID, &state.AggregateVersion, &state.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// Create initial state
		insertQuery := `
			INSERT INTO inventory_state (
				company_id, item_id, current_quantity, current_total_cost,
				aggregate_version, updated_at
			) VALUES ($1, $2, 0, 0, 0, NOW())
			RETURNING company_id, item_id, current_quantity, current_total_cost,
					  current_avg_cost, last_event_id, aggregate_version, updated_at
		`

		err = tx.QueryRowContext(ctx, insertQuery, companyID, itemID).Scan(
			&state.CompanyID, &state.ItemID, &state.CurrentQuantity, &state.CurrentTotalCost,
			&state.CurrentAvgCost, &state.LastEventID, &state.AggregateVersion, &state.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create initial state: %w", err)
		}

		return &state, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get inventory state: %w", err)
	}

	return &state, nil
}

// updateInventoryStateTx updates the inventory state within a transaction with optimistic locking
func (s *InventoryService) updateInventoryStateTx(
	ctx context.Context,
	tx *sql.Tx,
	companyID, itemID string,
	newQuantity, newTotalCost float64,
	eventID int64,
	newVersion, expectedVersion int,
) error {
	query := `
		UPDATE inventory_state
		SET current_quantity = $1,
			current_total_cost = $2,
			last_event_id = $3,
			aggregate_version = $4,
			updated_at = NOW()
		WHERE company_id = $5 
		  AND item_id = $6
		  AND aggregate_version = $7
	`

	result, err := tx.ExecContext(ctx, query,
		newQuantity, newTotalCost, eventID, newVersion,
		companyID, itemID, expectedVersion,
	)
	if err != nil {
		return fmt.Errorf("failed to update state: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("optimistic lock failed: state was modified by another transaction")
	}

	return nil
}
