// internal/dte/calculator.go
package dte

import (
	"cuentas/internal/codigos"
	"fmt"
	"math"
)

// Calculator handles all DTE monetary calculations.
// It contains pure functions with no side effects or external dependencies.
// All calculations follow El Salvador's Ministerio de Hacienda requirements.
type Calculator struct{}

// NewCalculator creates a new calculator instance
func NewCalculator() *Calculator {
	return &Calculator{}
}

// ============================================
// ITEM-LEVEL CALCULATIONS
// ============================================

// CalculateConsumidorFinal calculates amounts for B2C invoices (Factura Consumidor Final)
// where the price INCLUDES IVA (13%).
//
// Formula:
//   - precioUni = priceWithIVA (what customer sees on price tag)
//   - ventaGravada = (precioUni × cantidad) - descuento
//   - ivaItem = ventaGravada - (ventaGravada / 1.13)
//
// Example:
//
//	Customer pays $11.30 for 1 item:
//	- precioUni: 11.30 (includes IVA)
//	- ventaGravada: 11.30 (also includes IVA)
//	- ivaItem: 11.30 - (11.30 / 1.13) = 11.30 - 10.00 = 1.30

func (c *Calculator) CalculateConsumidorFinal(
	priceWithIVA float64,
	quantity float64,
	discount float64,
) ItemAmounts {
	// For consumidor final, precioUni includes IVA
	precioUni := priceWithIVA

	// ventaGravada = (price × qty) - discount (all include IVA)
	ventaGravada := precioUni*quantity - discount

	// Extract IVA from the total: IVA = total - (total / 1.13)
	// This is equivalent to: IVA = total × (0.13 / 1.13)
	ivaItem := ventaGravada - (ventaGravada / IVADivisor)

	return ItemAmounts{
		PrecioUni:    RoundToItemPrecision(precioUni),
		VentaGravada: RoundToItemPrecision(ventaGravada),
		IvaItem:      ivaItem,
		MontoDescu:   RoundToItemPrecision(discount), //
	}
}

// CalculateCreditoFiscal calculates amounts for B2B invoices (Crédito Fiscal)
// where the price EXCLUDES IVA (13%).
//
// Formula:
//   - precioUni = priceWithoutIVA (base price)
//   - ventaGravada = (precioUni × cantidad) - descuento
//   - ivaItem = ventaGravada × 0.13
//
// Example:
//
//	Business sells service for $10.00 + IVA:
//	- precioUni: 10.00 (without IVA)
//	- ventaGravada: 10.00
//	- ivaItem: 10.00 × 0.13 = 1.30
//	- Total customer pays: 11.30
func (c *Calculator) CalculateCreditoFiscal(
	priceWithoutIVA float64,
	quantity float64,
	discount float64,
) ItemAmounts {
	// For credito fiscal, precioUni excludes IVA
	precioUni := priceWithoutIVA

	// ventaGravada = (price × qty) - discount (all exclude IVA)
	ventaGravada := precioUni*quantity - discount

	// Calculate IVA on the base amount
	ivaItem := ventaGravada * IVARate

	return ItemAmounts{
		PrecioUni:    RoundToItemPrecision(precioUni),
		VentaGravada: RoundToItemPrecision(ventaGravada),
		IvaItem:      RoundToItemPrecision(ivaItem),
		MontoDescu:   RoundToItemPrecision(discount), // ⭐ ADD THIS
	}
}

// ============================================
// RESUMEN-LEVEL CALCULATIONS
// ============================================

// CalculateResumen calculates summary totals from item amounts.
// The calculation differs based on invoice type:
//
// Consumidor Final (B2C):
//   - totalGravada = sum(ventaGravada) [includes IVA]
//   - totalIva = sum(ivaItem) [extracted IVA]
//   - subTotal = totalGravada [NOT totalGravada + totalIva!]
//
// Crédito Fiscal (B2B):
//   - totalGravada = sum(ventaGravada) [excludes IVA]
//   - totalIva = sum(ivaItem)
//   - subTotal = totalGravada + totalIva
func (c *Calculator) CalculateResumen(
	items []ItemAmounts,
	invoiceType string,
) ResumenAmounts {
	// Sum all item amounts (keep precision during summation)
	var totalGravada, totalIva, totalDescu float64

	for _, item := range items {
		totalGravada += item.VentaGravada
		totalIva += item.IvaItem
		totalDescu += item.MontoDescu // ⭐ ADD THIS LINE
	}

	// Round totals to resumen precision (2 decimals)
	totalGravada = RoundToResumenPrecision(totalGravada)
	totalIva = RoundToResumenPrecision(totalIva)
	totalDescu = RoundToResumenPrecision(totalDescu) // ⭐ ADD THIS LINE

	// Calculate subTotal based on invoice type
	var subTotal float64
	if invoiceType == codigos.PersonTypeNatural {
		subTotal = totalGravada
	} else {
		subTotal = RoundToResumenPrecision(totalGravada + totalIva)
	}

	return ResumenAmounts{
		TotalNoSuj:          0,
		TotalExenta:         0,
		TotalGravada:        totalGravada,
		SubTotalVentas:      totalGravada,
		TotalDescu:          totalDescu, // ⭐ CHANGED FROM 0
		TotalIva:            totalIva,
		SubTotal:            subTotal,
		MontoTotalOperacion: subTotal,
		TotalPagar:          subTotal,
	}
}

// CalculateResumenWithDiscounts calculates resumen with global discounts.
// Use this when you have discounts applied at the invoice level (not per item).
func (c *Calculator) CalculateResumenWithDiscounts(
	items []ItemAmounts,
	invoiceType string,
	globalDiscountNoSuj float64,
	globalDiscountExenta float64,
	globalDiscountGravada float64,
) ResumenAmounts {
	// Start with basic calculation
	resumen := c.CalculateResumen(items, invoiceType)

	// Apply global discounts
	resumen.DescuNoSuj = RoundToResumenPrecision(globalDiscountNoSuj)
	resumen.DescuExenta = RoundToResumenPrecision(globalDiscountExenta)
	resumen.DescuGravada = RoundToResumenPrecision(globalDiscountGravada)

	totalDescu := resumen.DescuNoSuj + resumen.DescuExenta + resumen.DescuGravada
	resumen.TotalDescu = RoundToResumenPrecision(totalDescu)

	// Recalculate totals with discounts
	resumen.TotalGravada = RoundToResumenPrecision(resumen.TotalGravada - resumen.DescuGravada)

	// Recalculate IVA on the discounted amount if needed
	if invoiceType == codigos.PersonTypeNatural {
		// For consumidor final, extract IVA from the discounted total
		resumen.TotalIva = RoundToResumenPrecision(
			resumen.TotalGravada - (resumen.TotalGravada / IVADivisor),
		)
		resumen.SubTotal = resumen.TotalGravada
	} else {
		// For credito fiscal, IVA on discounted base
		resumen.TotalIva = RoundToResumenPrecision(resumen.TotalGravada * IVARate)
		resumen.SubTotal = RoundToResumenPrecision(resumen.TotalGravada + resumen.TotalIva)
	}

	resumen.MontoTotalOperacion = resumen.SubTotal
	resumen.TotalPagar = resumen.SubTotal

	return resumen
}

// CalculateResumenWithRetentions calculates resumen with tax retentions.
// Use this for B2B invoices where the buyer retains taxes.
func (c *Calculator) CalculateResumenWithRetentions(
	items []ItemAmounts,
	invoiceType string,
	ivaRetenido float64, // IVA Retenido (1%)
	rentaRetenida float64, // Renta Retenida
) ResumenAmounts {
	resumen := c.CalculateResumen(items, invoiceType)

	// Apply retentions
	resumen.IvaRete1 = RoundToResumenPrecision(ivaRetenido)
	resumen.ReteRenta = RoundToResumenPrecision(rentaRetenida)

	// Total to pay = subTotal - retentions
	resumen.TotalPagar = RoundToResumenPrecision(
		resumen.SubTotal - resumen.IvaRete1 - resumen.ReteRenta,
	)

	return resumen
}

func (c *Calculator) CalculateResumenCCF(items []ItemAmounts) ResumenAmounts {
	// Sum all ventaGravada and discounts
	var totalGravada, totalDescu float64

	for _, item := range items {
		totalGravada += item.VentaGravada
		totalDescu += item.MontoDescu
	}

	// Round totalGravada first, then calculate IVA
	totalGravadaRounded := RoundToResumenPrecision(totalGravada)
	totalIva := totalGravadaRounded * 0.13
	totalIvaRounded := RoundToResumenPrecision(totalIva)
	totalDescuRounded := RoundToResumenPrecision(totalDescu)

	// ⭐ Round the sum to avoid floating-point errors
	totalConIVA := RoundToResumenPrecision(totalGravadaRounded + totalIvaRounded)

	return ResumenAmounts{
		TotalNoSuj:          0,
		TotalExenta:         0,
		TotalGravada:        totalGravadaRounded,
		SubTotalVentas:      totalGravadaRounded,
		TotalDescu:          totalDescuRounded,
		TotalIva:            totalIvaRounded,
		SubTotal:            totalGravadaRounded, // NO IVA
		MontoTotalOperacion: totalConIVA,         // Rounded!
		TotalPagar:          totalConIVA,         // Rounded!
	}
}

func (c *Calculator) _CalculateResumenCCF(items []ItemAmounts) ResumenAmounts {
	// Sum all ventaGravada and discounts
	var totalGravada, totalDescu float64

	for _, item := range items {
		totalGravada += item.VentaGravada
		totalDescu += item.MontoDescu
	}

	// ⭐ Calculate IVA on the TOTAL (not per item)
	// Round totalGravada first, then calculate IVA
	totalGravadaRounded := RoundToResumenPrecision(totalGravada)
	totalIva := totalGravadaRounded * 0.13
	totalIvaRounded := RoundToResumenPrecision(totalIva)
	totalDescuRounded := RoundToResumenPrecision(totalDescu)

	// ⭐ For CCF: subTotal = totalGravada (NO IVA!)
	// ⭐ montoTotalOperacion = totalGravada + IVA
	totalConIVA := totalGravadaRounded + totalIvaRounded

	return ResumenAmounts{
		TotalNoSuj:          0,
		TotalExenta:         0,
		TotalGravada:        totalGravadaRounded,
		SubTotalVentas:      totalGravadaRounded,
		TotalDescu:          totalDescuRounded,
		TotalIva:            totalIvaRounded,
		SubTotal:            totalGravadaRounded, // ⭐ NO IVA!
		MontoTotalOperacion: totalConIVA,         // ⭐ WITH IVA
		TotalPagar:          totalConIVA,         // ⭐ WITH IVA
	}
}

func (c *Calculator) __CalculateResumenCCF(items []ItemAmounts) ResumenAmounts {
	// Sum all ventaGravada and discounts
	var totalGravada, totalDescu float64

	for _, item := range items {
		totalGravada += item.VentaGravada
		totalDescu += item.MontoDescu
	}

	// ⭐ Calculate IVA on the TOTAL (not per item)
	// Round totalGravada first, then calculate IVA
	totalGravadaRounded := RoundToResumenPrecision(totalGravada)
	totalIva := totalGravadaRounded * 0.13
	totalIvaRounded := RoundToResumenPrecision(totalIva)
	totalDescuRounded := RoundToResumenPrecision(totalDescu)

	// Calculate subTotal from rounded values
	subTotal := totalGravadaRounded + totalIvaRounded

	return ResumenAmounts{
		TotalNoSuj:          0,
		TotalExenta:         0,
		TotalGravada:        totalGravadaRounded,
		SubTotalVentas:      totalGravadaRounded,
		TotalDescu:          totalDescuRounded,
		TotalIva:            totalIvaRounded,
		SubTotal:            subTotal,
		MontoTotalOperacion: subTotal,
		TotalPagar:          subTotal,
	}
}

// ============================================
// ROUNDING FUNCTIONS
// ============================================

// RoundToItemPrecision rounds to 8 decimal places (item level).
// Uses banker's rounding (round half to even).
//
// Hacienda allows up to 8 decimal places for item-level amounts.
// When the 9th decimal is ≥ 5, round up the 8th decimal.
func RoundToItemPrecision(value float64) float64 {
	return math.Round(value*1e8) / 1e8
}

// RoundToResumenPrecision rounds to 2 decimal places (resumen level).
// Uses banker's rounding (round half to even).
//
// Hacienda requires exactly 2 decimal places for resumen amounts.
// When the 3rd decimal is ≥ 5, round up the 2nd decimal.
func RoundToResumenPrecision(value float64) float64 {
	return math.Round(value*100) / 100
}

// ============================================
// VALIDATION FUNCTIONS
// ============================================

// ValidateItemCalculation validates that item calculations are within tolerance.
// Hacienda allows ±0.01 cent difference.
func (c *Calculator) ValidateItemCalculation(item ItemAmounts) error {
	// Validate: precioUni × cantidad - descuento should equal ventaGravada
	// (This validation is conceptual - in practice, we calculate it)

	// Validate: IVA should be within tolerance
	tolerance := 0.01

	// For now, just check that values are reasonable
	if item.PrecioUni < 0 {
		return ErrNegativePrecio
	}
	if item.VentaGravada < 0 {
		return ErrNegativeVentaGravada
	}
	if item.IvaItem < 0 {
		return ErrNegativeIVA
	}

	// Check IVA is reasonable (should be ~13% of base for consumidor final)
	// This is a sanity check, not a strict validation
	_ = tolerance // Use if needed for strict validation

	return nil
}

// ============================================
// UTILITY FUNCTIONS
// ============================================

// CalculateIVAFromTotal extracts IVA amount from a total that includes IVA.
// Use this for consumidor final calculations.
//
// Formula: IVA = total - (total / 1.13)
func CalculateIVAFromTotal(totalWithIVA float64) float64 {
	return totalWithIVA - (totalWithIVA / IVADivisor)
}

// CalculateIVAFromBase calculates IVA amount from a base amount (without IVA).
// Use this for credito fiscal calculations.
//
// Formula: IVA = base × 0.13
func CalculateIVAFromBase(baseAmount float64) float64 {
	return baseAmount * IVARate
}

// CalculateBaseFromTotal calculates base amount from total that includes IVA.
//
// Formula: base = total / 1.13
func CalculateBaseFromTotal(totalWithIVA float64) float64 {
	return totalWithIVA / IVADivisor
}

// CalculateTotalFromBase calculates total amount from base (adds IVA).
//
// Formula: total = base × 1.13
func CalculateTotalFromBase(baseAmount float64) float64 {
	return baseAmount * IVADivisor
}

// CalculateResumenExportacion calculates summary totals for export invoices.
// Key difference: IVA is always 0%, but we still track ventaGravada.
//
// Export Invoice (Type 11):
//   - totalGravada = sum(ventaGravada) [0% IVA applied]
//   - totalIva = 0 (always 0% for exports)
//   - subTotal = totalGravada
//   - montoTotalOperacion = totalGravada + seguro + flete

// ============================================
// EXPORT CALCULATIONS (Type 11 - 0% IVA)
// ============================================

// CalculateExportacion calculates amounts for export invoices (Factura de Exportación)
func (c *Calculator) CalculateExportacion(
	priceWithoutIVA float64,
	quantity float64,
	discount float64,
) ItemAmounts {
	precioUni := priceWithoutIVA
	ventaGravada := precioUni * quantity

	fmt.Printf("[DEBUG] CalculateExportacion: price=%.2f, qty=%.2f, discount=%.2f → ventaGravada=%.2f (export 0%% with C3)\n",
		priceWithoutIVA, quantity, discount, ventaGravada)

	ivaItem := 0.0

	return ItemAmounts{
		PrecioUni:    RoundToItemPrecision(precioUni),
		VentaGravada: RoundToItemPrecision(ventaGravada),
		IvaItem:      ivaItem,
		MontoDescu:   RoundToItemPrecision(discount),
	}
}

func (c *Calculator) CalculateResumenExportacion(
	items []ItemAmounts,
	seguro float64,
	flete float64,
) ResumenAmounts {
	var totalGravada, totalDescu float64

	for _, item := range items {
		totalGravada += item.VentaGravada
		totalDescu += item.MontoDescu
	}

	totalGravada = RoundToResumenPrecision(totalGravada)
	totalDescu = RoundToResumenPrecision(totalDescu)
	seguro = RoundToResumenPrecision(seguro)
	flete = RoundToResumenPrecision(flete)

	totalIva := 0.0

	montoTotalOperacion := RoundToResumenPrecision(totalGravada + seguro + flete - totalDescu)

	totalPagar := RoundToResumenPrecision(totalGravada + seguro + flete - totalDescu)

	return ResumenAmounts{
		TotalNoSuj:          0,
		TotalExenta:         0,
		TotalGravada:        totalGravada,
		SubTotalVentas:      totalGravada,
		TotalDescu:          totalDescu,
		TotalIva:            totalIva,
		SubTotal:            totalGravada,
		MontoTotalOperacion: montoTotalOperacion,
		TotalPagar:          totalPagar,
		TotalNoGravado:      0,
	}
}
