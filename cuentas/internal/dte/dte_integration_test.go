package test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"cuentas/internal/dte"
	"cuentas/internal/models"
)

func TestDTEBuilderProducesCorrectJSON(t *testing.T) {
	// Skip if no database connection
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	// Create test invoice
	codigoGen := "B8AF117B-D8B1-0E60-E6D0-1B0CA6645D68"
	numeroControl := "DTE-01-M001P001-000000000000007"
	notes := "Test invoice"

	invoice := &models.Invoice{
		CompanyID:       "your-company-uuid-here",
		EstablishmentID: "your-establishment-uuid-here",
		PointOfSaleID:   "your-pos-uuid-here",
		ClientID:        "your-client-uuid-here",

		DteCodigoGeneracion: &codigoGen,
		DteNumeroControl:    &numeroControl,

		PaymentTerms: "cash",
		Notes:        &notes,

		LineItems: []models.InvoiceLineItem{
			{
				LineNumber:     1,
				ItemTipoItem:   "2", // Servicio
				ItemSku:        "PROD-001",
				ItemName:       "Servicio de consultoria",
				Quantity:       1,
				UnitPrice:      11.30, // WITH IVA for consumidor final
				DiscountAmount: 0,
				UnitOfMeasure:  "servicio",
				TaxableAmount:  10.00, // This will be IGNORED by calculator
				TotalTaxes:     1.30,  // This will be IGNORED by calculator
			},
		},
	}

	// Build DTE
	db := getTestDB(t) // You'll need to implement this
	builder := dte.NewBuilder(db)

	factura, err := builder.BuildFromInvoice(context.Background(), invoice)
	if err != nil {
		t.Fatalf("Failed to build DTE: %v", err)
	}

	// Print generated JSON
	jsonBytes, err := json.MarshalIndent(factura, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	fmt.Println("Generated DTE JSON:")
	fmt.Println(string(jsonBytes))

	// Verify cuerpoDocumento
	if len(factura.CuerpoDocumento) != 1 {
		t.Fatalf("Expected 1 line item, got %d", len(factura.CuerpoDocumento))
	}

	item := factura.CuerpoDocumento[0]

	// These should match your working test_dte.json
	assertEqual(t, item.PrecioUni, 11.30, "precioUni")
	assertEqual(t, item.VentaGravada, 11.30, "ventaGravada")
	assertEqual(t, item.IvaItem, 1.30, "ivaItem")

	// Verify resumen
	assertEqual(t, factura.Resumen.TotalGravada, 11.30, "resumen.totalGravada")
	assertEqual(t, factura.Resumen.SubTotal, 11.30, "resumen.subTotal")
	assertEqual(t, factura.Resumen.TotalIva, 1.30, "resumen.totalIva")
	assertEqual(t, factura.Resumen.TotalPagar, 11.30, "resumen.totalPagar")

	fmt.Println("\n✅ All assertions passed! DTE matches expected format.")
}

func assertEqual(t *testing.T, got, want float64, field string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %.2f, want %.2f", field, got, want)
	}
}
