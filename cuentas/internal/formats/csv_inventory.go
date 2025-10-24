package formats

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"time"

	"cuentas/internal/i18n"
	"cuentas/internal/models"
)

// WriteEventsCSV writes inventory events to CSV format with translations
func WriteEventsCSV(events []models.InventoryEventWithItem, lang string) ([]byte, error) {
	buf := new(bytes.Buffer)
	writer := csv.NewWriter(buf)

	// Get translations
	t := i18n.New(lang)

	// Write header
	if err := writer.Write(t.InventoryEventsHeaders()); err != nil {
		return nil, err
	}

	// Write data rows
	for _, event := range events {
		notes := ""
		if event.Notes != nil {
			notes = *event.Notes
		}

		row := []string{
			fmt.Sprintf("%d", event.EventID),
			event.EventTimestamp.Format(time.RFC3339),
			event.SKU,
			event.ItemName,
			t.EventType(event.EventType), // Translate event type
			fmt.Sprintf("%.2f", event.Quantity),
			fmt.Sprintf("%.2f", event.UnitCost.Float64()),
			fmt.Sprintf("%.2f", event.TotalCost.Float64()),
			fmt.Sprintf("%.2f", event.MovingAvgCostBefore.Float64()),
			fmt.Sprintf("%.2f", event.MovingAvgCostAfter.Float64()),
			fmt.Sprintf("%.2f", event.BalanceQuantityAfter),
			fmt.Sprintf("%.2f", event.BalanceTotalCostAfter.Float64()),
			notes,
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

// WriteValuationCSV writes inventory valuation to CSV format with translations
func WriteValuationCSV(valuation *models.InventoryValuation, lang string) ([]byte, error) {
	buf := new(bytes.Buffer)
	writer := csv.NewWriter(buf)

	// Get translations
	t := i18n.New(lang)

	// Summary section
	summaryHeader := t.ValuationSummaryHeaders()
	if err := writer.Write(summaryHeader); err != nil {
		return nil, err
	}

	summary := [][]string{
		{t.ValuationSummaryRow("As of Date"), valuation.AsOfDate.Format("2006-01-02")},
		{t.ValuationSummaryRow("Total Value"), fmt.Sprintf("%.2f", valuation.TotalValue.Float64())},
		{t.ValuationSummaryRow("Total Quantity"), fmt.Sprintf("%.2f", valuation.TotalQuantity)},
		{t.ValuationSummaryRow("Item Count"), fmt.Sprintf("%d", valuation.ItemCount)},
		{}, // Blank row
	}

	for _, row := range summary {
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	// Detail header
	if err := writer.Write(t.ValuationDetailHeaders()); err != nil {
		return nil, err
	}

	// Detail rows
	for _, item := range valuation.ItemValues {
		row := []string{
			item.SKU,
			item.ItemName,
			fmt.Sprintf("%.2f", item.Quantity),
			fmt.Sprintf("%.2f", item.AvgCost.Float64()),
			fmt.Sprintf("%.2f", item.TotalValue.Float64()),
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

// WriteInventoryStatesCSV writes inventory states to CSV format with translations
func WriteInventoryStatesCSV(states []models.InventoryStateWithItem, lang string) ([]byte, error) {
	buf := new(bytes.Buffer)
	writer := csv.NewWriter(buf)

	// Get translations
	t := i18n.New(lang)

	// Write header
	if err := writer.Write(t.InventoryStatesHeaders()); err != nil {
		return nil, err
	}

	// Write data rows
	for _, state := range states {
		row := []string{
			state.SKU,
			state.ItemName,
			t.ItemType(state.TipoItem), // Translate item type
			fmt.Sprintf("%.2f", state.CurrentQuantity),
			fmt.Sprintf("%.2f", state.CurrentAvgCost.Float64()),
			fmt.Sprintf("%.2f", state.CurrentTotalCost.Float64()),
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

// DetermineLanguage determines language from query param (default: Spanish)
func DetermineLanguage(langParam string) string {
	if langParam == "en" {
		return "en"
	}
	// Default to Spanish
	return "es"
}
