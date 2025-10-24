package formats

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"time"

	"cuentas/internal/models"
)

// WriteEventsCSV writes inventory events to CSV format
func WriteEventsCSV(events []models.InventoryEventWithItem) ([]byte, error) {
	buf := new(bytes.Buffer)
	writer := csv.NewWriter(buf)

	// Write header
	header := []string{
		"Event ID", "Timestamp", "SKU", "Item Name", "Event Type",
		"Quantity", "Unit Cost", "Total Cost",
		"Avg Cost Before", "Avg Cost After",
		"Balance Qty", "Balance Value", "Notes",
	}
	if err := writer.Write(header); err != nil {
		return nil, err
	}

	// Write data rows
	for _, event := range events {
		row := []string{
			fmt.Sprintf("%d", event.EventID),
			event.EventTimestamp.Format(time.RFC3339),
			event.SKU,
			event.ItemName,
			event.EventType,
			fmt.Sprintf("%.2f", event.Quantity),
			fmt.Sprintf("%.2f", event.UnitCost),
			fmt.Sprintf("%.2f", event.TotalCost),
			fmt.Sprintf("%.2f", event.MovingAvgCostBefore),
			fmt.Sprintf("%.2f", event.MovingAvgCostAfter),
			fmt.Sprintf("%.2f", event.BalanceQuantityAfter),
			fmt.Sprintf("%.2f", event.BalanceTotalCostAfter),
		}

		if event.Notes != nil {
			row = append(row, *event.Notes)
		} else {
			row = append(row, "")
		}

		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// WriteValuationCSV writes inventory valuation to CSV format
func WriteValuationCSV(valuation *models.InventoryValuation) ([]byte, error) {
	buf := new(bytes.Buffer)
	writer := csv.NewWriter(buf)

	// Summary section
	summary := [][]string{
		{"INVENTORY VALUATION SUMMARY"},
		{"As of Date", valuation.AsOfDate.Format("2006-01-02")},
		{"Total Value", fmt.Sprintf("%.2f", valuation.TotalValue)},
		{"Total Quantity", fmt.Sprintf("%.2f", valuation.TotalQuantity)},
		{"Item Count", fmt.Sprintf("%d", valuation.ItemCount)},
		{}, // Blank row
	}

	for _, row := range summary {
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	// Detail header
	header := []string{"SKU", "Item Name", "Quantity", "Avg Cost", "Total Value", "Last Event Date"}
	if err := writer.Write(header); err != nil {
		return nil, err
	}

	// Detail rows
	for _, item := range valuation.ItemValues {
		row := []string{
			item.SKU,
			item.ItemName,
			fmt.Sprintf("%.2f", item.Quantity),
			fmt.Sprintf("%.2f", item.AvgCost),
			fmt.Sprintf("%.2f", item.TotalValue),
			item.LastEventAt.Format(time.RFC3339),
		}
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// WriteInventoryStatesCSV writes inventory states to CSV format
func WriteInventoryStatesCSV(states []models.InventoryStateWithItem) ([]byte, error) {
	buf := new(bytes.Buffer)
	writer := csv.NewWriter(buf)

	// Write header
	header := []string{
		"SKU", "Item Name", "Type", "Quantity", "Avg Cost", "Total Value", "Last Updated",
	}
	if err := writer.Write(header); err != nil {
		return nil, err
	}

	// Write data rows
	for _, state := range states {
		itemType := "Service"
		if state.TipoItem == "1" {
			itemType = "Product"
		}

		row := []string{
			state.SKU,
			state.ItemName,
			itemType,
			fmt.Sprintf("%.2f", state.CurrentQuantity),
			fmt.Sprintf("%.2f", state.CurrentAvgCost),
			fmt.Sprintf("%.2f", state.CurrentTotalCost),
			state.UpdatedAt.Format(time.RFC3339),
		}

		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// DetermineFormat determines output format from Accept header or query param
func DetermineFormat(acceptHeader, formatParam string) string {
	// Query param takes precedence
	if formatParam == "csv" {
		return "csv"
	}
	if formatParam == "json" || formatParam == "" {
		return "json"
	}

	// Check Accept header
	if acceptHeader == "text/csv" || acceptHeader == "application/csv" {
		return "csv"
	}

	// Default to JSON
	return "json"
}
