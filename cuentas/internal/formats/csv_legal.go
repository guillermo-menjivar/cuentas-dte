package formats

import (
	"bytes"
	"cuentas/internal/i18n"
	"cuentas/internal/models"
	"encoding/csv"
	"fmt"
	"strings"
)

// WriteLegalInventoryRegisterCSV writes a legal inventory register per item (Article 142-A compliant)
func WriteLegalInventoryRegisterCSV(
	companyInfo *models.CompanyLegalInfo,
	item *models.InventoryItem,
	events []models.InventoryEventWithItem,
	startDate, endDate string,
	lang string,
) ([]byte, error) {
	buf := new(bytes.Buffer)
	writer := csv.NewWriter(buf)
	t := i18n.New(lang)

	// Header section with additional legal requirements
	header := [][]string{
		{t.FormatRegisterHeader()},
		{t.FormatCompanyLabel(), companyInfo.LegalName},
		{"NIT", companyInfo.NIT},
		{"NRC", companyInfo.NRC},
		{t.FormatPeriodLabel(), fmt.Sprintf("Del %s al %s", startDate, endDate)},
		{t.FormatItemLabel(), fmt.Sprintf("%s - %s", item.SKU, item.Name)},
		{"Unidad de Medida", item.UnitOfMeasure},            // NEW
		{"Método de Valuación", "Costo Promedio Ponderado"}, // NEW
		{}, // Blank row
	}
	for _, row := range header {
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	// Column headers (with new Observaciones column)
	if err := writer.Write(t.InventoryRegisterHeaders()); err != nil {
		return nil, err
	}

	// Data rows
	for i, event := range events {
		// Separate units in/out based on event type
		unitsIn := ""
		unitsOut := ""
		if event.EventType == "PURCHASE" || event.EventType == "ADJUSTMENT" {
			if event.Quantity > 0 {
				unitsIn = fmt.Sprintf("%.2f", event.Quantity)
			} else if event.Quantity < 0 {
				unitsOut = fmt.Sprintf("%.2f", -event.Quantity)
			}
		} else if event.EventType == "SALE" || event.EventType == "RETURN" {
			if event.Quantity > 0 {
				unitsOut = fmt.Sprintf("%.2f", event.Quantity)
			}
		}

		// Separate cost in/out
		costIn := ""
		costOut := ""
		if event.TotalCost.Float64() > 0 {
			costIn = fmt.Sprintf("%.2f", event.TotalCost.Float64())
		} else if event.TotalCost.Float64() < 0 {
			costOut = fmt.Sprintf("%.2f", -event.TotalCost.Float64())
		}

		// Document info
		docType := ""
		if event.DocumentType != nil {
			docType = *event.DocumentType
		}

		docNumber := ""
		if event.DocumentNumber != nil {
			docNumber = *event.DocumentNumber
		}

		// Supplier/Customer name (without NIT)
		supplierOrCustomer := ""
		if event.EventType == "PURCHASE" {
			if event.SupplierName != nil {
				supplierOrCustomer = *event.SupplierName
			}
		} else if event.EventType == "SALE" {
			if event.CustomerName != nil {
				supplierOrCustomer = *event.CustomerName
			}
		}

		// NIT (separate column) - NORMALIZED (no dashes)
		nit := ""
		if event.EventType == "PURCHASE" {
			if event.SupplierNIT != nil && *event.SupplierNIT != "" {
				nit = normalizeNIT(*event.SupplierNIT)
			}
		} else if event.EventType == "SALE" {
			if event.CustomerNIT != nil && *event.CustomerNIT != "" {
				nit = normalizeNIT(*event.CustomerNIT)
			} else {
				// Only set "CF" if customer NIT is actually empty/null
				nit = "CF"
			}
		}

		// Nationality
		nationality := ""
		if event.EventType == "PURCHASE" && event.SupplierNationality != nil {
			nationality = *event.SupplierNationality
		} else if event.EventType == "SALE" {
			nationality = "Nacional"
		}

		// Source reference - improved format
		sourceRef := ""
		if event.CostSourceRef != nil {
			sourceRef = *event.CostSourceRef
		} else if event.InvoiceID != nil {
			// For sales, should reference libro de ventas folio
			sourceRef = "Libro de Ventas"
		} else if event.Notes != nil {
			sourceRef = *event.Notes
		}

		// Calculate Costo Promedio (Moving Average Cost After the event)
		costoPromedio := ""
		if event.BalanceQuantityAfter > 0 {
			avgCost := event.BalanceTotalCostAfter.Float64() / event.BalanceQuantityAfter
			costoPromedio = fmt.Sprintf("%.4f", avgCost)
		} else {
			costoPromedio = "0.0000"
		}

		// Observaciones (notes/remarks for audit trail)
		observaciones := ""
		if event.Notes != nil {
			observaciones = *event.Notes
		}

		row := []string{
			fmt.Sprintf("%d", i+1),                             // Correlativo
			event.EventTimestamp.Format("2006-01-02 15:04:05"), // Fecha
			event.EventType,                                    // Tipo Evento (PURCHASE/SALE)
			docType,                                            // Tipo Doc
			docNumber,                                          // No. Documento
			supplierOrCustomer,                                 // Proveedor/Cliente
			nit,                                                // NIT (normalized)
			nationality,                                        // Nacionalidad
			sourceRef,                                          // Fuente/Referencia
			unitsIn,                                            // Unidades Entrada
			unitsOut,                                           // Unidades Salida
			fmt.Sprintf("%.2f", event.BalanceQuantityAfter), // Saldo Unidades
			costIn,  // Costo Entrada
			costOut, // Costo Salida
			fmt.Sprintf("%.2f", event.BalanceTotalCostAfter.Float64()), // Saldo Costo
			costoPromedio, // Costo Promedio
			observaciones, // Observaciones (NEW)
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

// normalizeNIT removes dashes and spaces from NIT for consistent formatting
func normalizeNIT(nit string) string {
	// Remove dashes and spaces
	normalized := strings.ReplaceAll(nit, "-", "")
	normalized = strings.ReplaceAll(normalized, " ", "")
	return normalized
}
