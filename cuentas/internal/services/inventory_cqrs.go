package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

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
	purchaseTotal := req.UnitCost.Mul(req.Quantity)
	newTotalCost := currentState.CurrentTotalCost.Add(purchaseTotal)
	newQuantity := currentState.CurrentQuantity + req.Quantity

	var newAvgCost models.Money
	if newQuantity > 0 {
		newAvgCost = newTotalCost.Div(newQuantity)
	}

	nextVersion := currentState.AggregateVersion + 1

	// Build event data
	eventData := map[string]interface{}{
		"quantity":   req.Quantity,
		"unit_cost":  req.UnitCost.Float64(),
		"total_cost": purchaseTotal.Float64(),
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
	err = tx.QueryRowContext(ctx, eventQuery, // Changed from tx.QueryRow to tx.QueryRowContext
		companyID, itemID, "PURCHASE",
		nextVersion, req.Quantity, req.UnitCost.Float64(), purchaseTotal.Float64(),
		newQuantity, newTotalCost.Float64(),
		currentState.CurrentAvgCost.Float64(), newAvgCost.Float64(),
		req.ReferenceType, req.ReferenceID, req.CorrelationID,
		eventDataJSON, req.Notes, nil,
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
	err = s.updateInventoryStateTx(ctx, tx, companyID, itemID, newQuantity, newTotalCost.Float64(), event.EventID, nextVersion, currentState.AggregateVersion) // Changed to .Float64()
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

	// Calculate new quantity first (needed for validation)
	newQuantity := currentState.CurrentQuantity + req.Quantity
	if newQuantity < 0 {
		return nil, fmt.Errorf("adjustment would result in negative quantity (current: %.2f, adjustment: %.2f)", currentState.CurrentQuantity, req.Quantity)
	}

	// Determine unit cost
	var unitCost models.Money
	if req.Quantity > 0 {
		if req.UnitCost == nil {
			return nil, fmt.Errorf("unit_cost required when adding inventory")
		}
		unitCost = *req.UnitCost
	} else {
		unitCost = currentState.CurrentAvgCost
	}

	// Calculate new values
	adjustmentTotal := unitCost.Mul(req.Quantity)
	newTotalCost := currentState.CurrentTotalCost.Add(adjustmentTotal)
	if newTotalCost.Float64() < 0 { // Fixed: added .Float64()
		newTotalCost = 0 // Safety check
	}

	var newAvgCost models.Money // Fixed: changed from float64
	if newQuantity > 0 {
		newAvgCost = newTotalCost.Div(newQuantity) // Fixed: use .Div()
	}

	nextVersion := currentState.AggregateVersion + 1

	// Build event data
	eventData := map[string]interface{}{
		"quantity":   req.Quantity,
		"unit_cost":  unitCost.Float64(),        // Fixed: added .Float64()
		"total_cost": adjustmentTotal.Float64(), // Fixed: use adjustmentTotal
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
		nextVersion, req.Quantity, unitCost.Float64(), adjustmentTotal.Float64(),
		newQuantity, newTotalCost.Float64(),
		currentState.CurrentAvgCost.Float64(), newAvgCost.Float64(),
		req.ReferenceType, req.ReferenceID, req.CorrelationID,
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
	err = s.updateInventoryStateTx(ctx, tx, companyID, itemID, newQuantity, newTotalCost.Float64(), event.EventID, nextVersion, currentState.AggregateVersion) // Fixed: added .Float64()
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
	fmt.Println("we are here... inside the state")
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
		fmt.Println("we failed to get inventory state", err)
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
// ListInventoryStates gets inventory states with item details
func (s *InventoryService) ListInventoryStates(
	ctx context.Context,
	companyID string,
	inStockOnly bool,
) ([]models.InventoryStateWithItem, error) {
	query := `
		SELECT 
			s.company_id, s.item_id, 
			i.sku, i.name as item_name, i.tipo_item,
			s.current_quantity, s.current_total_cost, s.current_avg_cost,
			s.last_event_id, s.aggregate_version, s.updated_at
		FROM inventory_states s
		JOIN inventory_items i ON s.item_id = i.id
		WHERE s.company_id = $1
	`

	if inStockOnly {
		query += " AND s.current_quantity > 0"
	}

	query += " ORDER BY i.sku"

	rows, err := s.db.QueryContext(ctx, query, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to list inventory states: %w", err)
	}
	defer rows.Close()

	var states []models.InventoryStateWithItem
	for rows.Next() {
		var state models.InventoryStateWithItem

		err := rows.Scan(
			&state.CompanyID,
			&state.ItemID,
			&state.SKU,
			&state.ItemName,
			&state.TipoItem,
			&state.CurrentQuantity,
			&state.CurrentTotalCost, // Money.Scan() handles conversion
			&state.CurrentAvgCost,   // Money.Scan() handles conversion
			&state.LastEventID,
			&state.AggregateVersion,
			&state.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory state: %w", err)
		}

		states = append(states, state)
	}

	return states, nil
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

// GetCostHistory gets the cost event history for an item with date filtering
func (s *InventoryService) GetCostHistory(
	ctx context.Context,
	companyID, itemID string,
	limit int,
	sortOrder string,
	startDate, endDate string,
) ([]models.InventoryEvent, error) {
	// Verify item exists and belongs to company
	_, err := s.GetItemByID(ctx, companyID, itemID)
	if err != nil {
		return nil, fmt.Errorf("item not found: %w", err)
	}

	if limit <= 0 {
		limit = 50 // Default limit
	}

	// Validate sort order
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	// Build query with optional date filters
	query := `
		SELECT 
			event_id, company_id, item_id, event_type, event_timestamp,
			aggregate_version, quantity, unit_cost, total_cost,
			balance_quantity_after, balance_total_cost_after,
			moving_avg_cost_before, moving_avg_cost_after,
			reference_type, reference_id, correlation_id,
			event_data, notes, created_by_user_id, created_at
		FROM inventory_events
		WHERE company_id = $1 AND item_id = $2
	`

	args := []interface{}{companyID, itemID}
	argCount := 2

	// Add date filters if provided
	if startDate != "" {
		// Validate date format
		if _, err := time.Parse("2006-01-02", startDate); err != nil {
			return nil, fmt.Errorf("invalid start_date format, use YYYY-MM-DD: %w", err)
		}
		argCount++
		query += fmt.Sprintf(" AND event_timestamp >= $%d", argCount)
		args = append(args, startDate+"T00:00:00Z")
	}

	if endDate != "" {
		// Validate date format
		if _, err := time.Parse("2006-01-02", endDate); err != nil {
			return nil, fmt.Errorf("invalid end_date format, use YYYY-MM-DD: %w", err)
		}
		argCount++
		query += fmt.Sprintf(" AND event_timestamp <= $%d", argCount)
		args = append(args, endDate+"T23:59:59Z")
	}

	// Add ordering and limit
	query += fmt.Sprintf(" ORDER BY event_timestamp %s, event_id %s LIMIT $%d", sortOrder, sortOrder, argCount+1)
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
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

	return events, nil
}

// GetAllEvents gets all inventory events across all items with filters
func (s *InventoryService) GetAllEvents(
	ctx context.Context,
	companyID string,
	startDate, endDate string,
	eventType string,
	sortOrder string,
) ([]models.InventoryEventWithItem, error) {
	// Validate sort order
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	// Build query
	query := `
		SELECT 
			e.event_id, e.company_id, e.item_id, e.event_type, e.event_timestamp,
			e.aggregate_version, e.quantity, e.unit_cost, e.total_cost,
			e.balance_quantity_after, e.balance_total_cost_after,
			e.moving_avg_cost_before, e.moving_avg_cost_after,
			e.reference_type, e.reference_id, e.correlation_id,
			e.event_data, e.notes, e.created_by_user_id, e.created_at,
			i.sku, i.name as item_name
		FROM inventory_events e
		JOIN inventory_items i ON e.item_id = i.id
		WHERE e.company_id = $1
	`

	args := []interface{}{companyID}
	argCount := 1

	// Add date filters
	if startDate != "" {
		if _, err := time.Parse("2006-01-02", startDate); err != nil {
			return nil, fmt.Errorf("invalid start_date format, use YYYY-MM-DD: %w", err)
		}
		argCount++
		query += fmt.Sprintf(" AND e.event_timestamp >= $%d", argCount)
		args = append(args, startDate+"T00:00:00Z")
	}

	if endDate != "" {
		if _, err := time.Parse("2006-01-02", endDate); err != nil {
			return nil, fmt.Errorf("invalid end_date format, use YYYY-MM-DD: %w", err)
		}
		argCount++
		query += fmt.Sprintf(" AND e.event_timestamp <= $%d", argCount)
		args = append(args, endDate+"T23:59:59Z")
	}

	// Add event type filter
	if eventType != "" {
		argCount++
		query += fmt.Sprintf(" AND e.event_type = $%d", argCount)
		args = append(args, eventType)
	}

	// Add ordering
	query += fmt.Sprintf(" ORDER BY e.event_timestamp %s, e.event_id %s", sortOrder, sortOrder)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}
	defer rows.Close()

	var events []models.InventoryEventWithItem
	for rows.Next() {
		var event models.InventoryEventWithItem
		err := rows.Scan(
			&event.EventID, &event.CompanyID, &event.ItemID, &event.EventType, &event.EventTimestamp,
			&event.AggregateVersion, &event.Quantity, &event.UnitCost, &event.TotalCost,
			&event.BalanceQuantityAfter, &event.BalanceTotalCostAfter,
			&event.MovingAvgCostBefore, &event.MovingAvgCostAfter,
			&event.ReferenceType, &event.ReferenceID, &event.CorrelationID,
			&event.EventData, &event.Notes, &event.CreatedByUserID, &event.CreatedAt,
			&event.SKU, &event.ItemName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, event)
	}

	return events, nil
}

// GetInventoryValuationAtDate calculates total inventory value at a specific date
func (s *InventoryService) GetInventoryValuationAtDate(
	ctx context.Context,
	companyID string,
	asOfDate string,
) (*models.InventoryValuation, error) {
	// Validate date format
	targetDate, err := time.Parse("2006-01-02", asOfDate)
	if err != nil {
		return nil, fmt.Errorf("invalid date format, use YYYY-MM-DD: %w", err)
	}

	// Get all product items
	items, err := s.ListItems(ctx, companyID, false, "1") // tipo_item = "1" (products only)
	if err != nil {
		return nil, fmt.Errorf("failed to list items: %w", err)
	}

	valuation := &models.InventoryValuation{
		AsOfDate:   targetDate,
		CompanyID:  companyID,
		ItemValues: []models.ItemValuation{},
	}

	var totalValue models.Money
	var totalQuantity float64

	// For each item, calculate its value at the target date
	for _, item := range items {
		// Get all events up to the target date
		events, err := s.GetCostHistory(ctx, companyID, item.ID, 10000, "asc", "", asOfDate)
		if err != nil {
			return nil, fmt.Errorf("failed to get history for item %s: %w", item.ID, err)
		}

		if len(events) == 0 {
			// No events before this date - item had zero inventory
			continue
		}

		// Get the last event before the target date
		lastEvent := events[len(events)-1]

		itemVal := models.ItemValuation{
			ItemID:      item.ID,
			SKU:         item.SKU,
			ItemName:    item.Name,
			Quantity:    lastEvent.BalanceQuantityAfter,
			AvgCost:     lastEvent.MovingAvgCostAfter,
			TotalValue:  lastEvent.BalanceTotalCostAfter,
			LastEventID: lastEvent.EventID,
			LastEventAt: lastEvent.EventTimestamp,
		}

		valuation.ItemValues = append(valuation.ItemValues, itemVal)
		totalValue = totalValue.Add(itemVal.TotalValue)
		totalQuantity += itemVal.Quantity
	}

	valuation.TotalValue = totalValue
	valuation.TotalQuantity = totalQuantity
	valuation.ItemCount = len(valuation.ItemValues)

	return valuation, nil
}
