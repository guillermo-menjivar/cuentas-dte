package dte

import (
	"testing"
)

func TestCalculateConsumidorFinal(t *testing.T) {
	calc := NewCalculator()

	tests := []struct {
		name         string
		priceWithIVA float64
		quantity     float64
		discount     float64
		wantPrecio   float64
		wantGravada  float64
		wantIVA      float64
	}{
		{
			name:         "simple $11.30 total",
			priceWithIVA: 11.30,
			quantity:     1,
			discount:     0,
			wantPrecio:   11.30,
			wantGravada:  11.30,
			wantIVA:      1.30,
		},
		{
			name:         "simple $10.00 total",
			priceWithIVA: 10.00,
			quantity:     1,
			discount:     0,
			wantPrecio:   10.00,
			wantGravada:  10.00,
			wantIVA:      1.15, // 10 - (10/1.13) = 1.1504... rounds to 1.15
		},
		{
			name:         "real example $79.90",
			priceWithIVA: 79.90,
			quantity:     1,
			discount:     0,
			wantPrecio:   79.90,
			wantGravada:  79.90,
			wantIVA:      9.19, // 79.90 - (79.90/1.13) = 9.1920... rounds to 9.19
		},
		{
			name:         "multiple quantity",
			priceWithIVA: 11.30,
			quantity:     2,
			discount:     0,
			wantPrecio:   11.30,
			wantGravada:  22.60,
			wantIVA:      2.60, // 22.60 - (22.60/1.13) = 2.6008... rounds to 2.60
		},
		{
			name:         "with discount",
			priceWithIVA: 11.30,
			quantity:     1,
			discount:     1.30,
			wantPrecio:   11.30,
			wantGravada:  10.00,
			wantIVA:      1.15, // 10 - (10/1.13) = 1.1504... rounds to 1.15
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calc.CalculateConsumidorFinal(tt.priceWithIVA, tt.quantity, tt.discount)

			if result.PrecioUni != tt.wantPrecio {
				t.Errorf("PrecioUni = %v, want %v", result.PrecioUni, tt.wantPrecio)
			}
			if result.VentaGravada != tt.wantGravada {
				t.Errorf("VentaGravada = %v, want %v", result.VentaGravada, tt.wantGravada)
			}
			if result.IvaItem != tt.wantIVA {
				t.Errorf("IvaItem = %v, want %v", result.IvaItem, tt.wantIVA)
			}
		})
	}
}

func TestCalculateCreditoFiscal(t *testing.T) {
	calc := NewCalculator()

	tests := []struct {
		name            string
		priceWithoutIVA float64
		quantity        float64
		discount        float64
		wantPrecio      float64
		wantGravada     float64
		wantIVA         float64
	}{
		{
			name:            "simple $10 base",
			priceWithoutIVA: 10.00,
			quantity:        1,
			discount:        0,
			wantPrecio:      10.00,
			wantGravada:     10.00,
			wantIVA:         1.30, // 10 × 0.13 = 1.30
		},
		{
			name:            "simple $100 base",
			priceWithoutIVA: 100.00,
			quantity:        1,
			discount:        0,
			wantPrecio:      100.00,
			wantGravada:     100.00,
			wantIVA:         13.00, // 100 × 0.13 = 13.00
		},
		{
			name:            "multiple quantity",
			priceWithoutIVA: 10.00,
			quantity:        3,
			discount:        0,
			wantPrecio:      10.00,
			wantGravada:     30.00,
			wantIVA:         3.90, // 30 × 0.13 = 3.90
		},
		{
			name:            "with discount",
			priceWithoutIVA: 10.00,
			quantity:        2,
			discount:        5.00,
			wantPrecio:      10.00,
			wantGravada:     15.00, // (10 × 2) - 5 = 15
			wantIVA:         1.95,  // 15 × 0.13 = 1.95
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calc.CalculateCreditoFiscal(tt.priceWithoutIVA, tt.quantity, tt.discount)

			if result.PrecioUni != tt.wantPrecio {
				t.Errorf("PrecioUni = %v, want %v", result.PrecioUni, tt.wantPrecio)
			}
			if result.VentaGravada != tt.wantGravada {
				t.Errorf("VentaGravada = %v, want %v", result.VentaGravada, tt.wantGravada)
			}
			if result.IvaItem != tt.wantIVA {
				t.Errorf("IvaItem = %v, want %v", result.IvaItem, tt.wantIVA)
			}
		})
	}
}

func TestCalculateResumenConsumidorFinal(t *testing.T) {
	calc := NewCalculator()

	items := []ItemAmounts{
		{PrecioUni: 11.30, VentaGravada: 11.30, IvaItem: 1.30},
	}

	result := calc.CalculateResumen(items, InvoiceTypeConsumidorFinal)

	// For consumidor final: subTotal = totalGravada (NOT totalGravada + IVA!)
	if result.TotalGravada != 11.30 {
		t.Errorf("TotalGravada = %v, want 11.30", result.TotalGravada)
	}
	if result.TotalIva != 1.30 {
		t.Errorf("TotalIva = %v, want 1.30", result.TotalIva)
	}
	if result.SubTotal != 11.30 {
		t.Errorf("SubTotal = %v, want 11.30 (should equal TotalGravada, NOT TotalGravada + IVA!)", result.SubTotal)
	}
	if result.TotalPagar != 11.30 {
		t.Errorf("TotalPagar = %v, want 11.30", result.TotalPagar)
	}
}

func TestCalculateResumenCreditoFiscal(t *testing.T) {
	calc := NewCalculator()

	items := []ItemAmounts{
		{PrecioUni: 10.00, VentaGravada: 10.00, IvaItem: 1.30},
	}

	result := calc.CalculateResumen(items, InvoiceTypeCreditoFiscal)

	// For credito fiscal: subTotal = totalGravada + totalIva
	if result.TotalGravada != 10.00 {
		t.Errorf("TotalGravada = %v, want 10.00", result.TotalGravada)
	}
	if result.TotalIva != 1.30 {
		t.Errorf("TotalIva = %v, want 1.30", result.TotalIva)
	}
	if result.SubTotal != 11.30 {
		t.Errorf("SubTotal = %v, want 11.30 (should be TotalGravada + IVA)", result.SubTotal)
	}
	if result.TotalPagar != 11.30 {
		t.Errorf("TotalPagar = %v, want 11.30", result.TotalPagar)
	}
}

func TestRoundingFunctions(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		want8 float64 // 8 decimals
		want2 float64 // 2 decimals
	}{
		{
			name:  "simple",
			value: 1.12345678901,
			want8: 1.12345679, // 9th decimal (9) rounds up
			want2: 1.12,
		},
		{
			name:  "edge case .5",
			value: 1.125,
			want8: 1.125,
			want2: 1.12, // Banker's rounding: round to even
		},
		{
			name:  "IVA calculation",
			value: 1.1504424778761, // 10 / 1.13 = 8.849557522, * 0.13 = 1.1504424778761
			want8: 1.15044248,
			want2: 1.15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got8 := RoundToItemPrecision(tt.value)
			if got8 != tt.want8 {
				t.Errorf("RoundToItemPrecision(%v) = %v, want %v", tt.value, got8, tt.want8)
			}

			got2 := RoundToResumenPrecision(tt.value)
			if got2 != tt.want2 {
				t.Errorf("RoundToResumenPrecision(%v) = %v, want %v", tt.value, got2, tt.want2)
			}
		})
	}
}

func TestUtilityFunctions(t *testing.T) {
	// Test IVA extraction from total
	totalWithIVA := 11.30
	iva := CalculateIVAFromTotal(totalWithIVA)
	expected := 1.30
	if RoundToResumenPrecision(iva) != expected {
		t.Errorf("CalculateIVAFromTotal(%v) = %v, want %v", totalWithIVA, iva, expected)
	}

	// Test base extraction from total
	base := CalculateBaseFromTotal(totalWithIVA)
	expectedBase := 10.00
	if RoundToResumenPrecision(base) != expectedBase {
		t.Errorf("CalculateBaseFromTotal(%v) = %v, want %v", totalWithIVA, base, expectedBase)
	}

	// Test IVA calculation from base
	baseAmount := 10.00
	ivaFromBase := CalculateIVAFromBase(baseAmount)
	expectedIVA := 1.30
	if ivaFromBase != expectedIVA {
		t.Errorf("CalculateIVAFromBase(%v) = %v, want %v", baseAmount, ivaFromBase, expectedIVA)
	}
}
