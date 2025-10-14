package dte

import "math"

// Calculator handles all DTE monetary calculations
// It contains pure functions with no side effects or external dependencies
type Calculator struct{}

// CalculateConsumidorFinal calculates amounts for B2C invoices
// where the price includes IVA
func (c *Calculator) CalculateConsumidorFinal(
	priceWithIVA float64,
	quantity float64,
	discount float64,
) ItemAmounts {
	precioUni := priceWithIVA
	ventaGravada := precioUni*quantity - discount
	ivaItem := ventaGravada - (ventaGravada / IVADivisor)

	return ItemAmounts{
		PrecioUni:    RoundToItemPrecision(precioUni),
		VentaGravada: RoundToItemPrecision(ventaGravada),
		IvaItem:      RoundToItemPrecision(ivaItem),
	}
}

// CalculateCreditoFiscal calculates amounts for B2B invoices
// where the price excludes IVA
func (c *Calculator) CalculateCreditoFiscal(
	priceWithoutIVA float64,
	quantity float64,
	discount float64,
) ItemAmounts {
	precioUni := priceWithoutIVA
	ventaGravada := precioUni*quantity - discount
	ivaItem := ventaGravada * IVARate

	return ItemAmounts{
		PrecioUni:    RoundToItemPrecision(precioUni),
		VentaGravada: RoundToItemPrecision(ventaGravada),
		IvaItem:      RoundToItemPrecision(ivaItem),
	}
}

// CalculateResumen calculates summary totals
func (c *Calculator) CalculateResumen(
	items []ItemAmounts,
	invoiceType InvoiceType,
) ResumenAmounts {
	var totalGravada, totalIva float64

	for _, item := range items {
		totalGravada += item.VentaGravada
		totalIva += item.IvaItem
	}

	// Round to resumen precision (2 decimals)
	totalGravada = RoundToResumenPrecision(totalGravada)
	totalIva = RoundToResumenPrecision(totalIva)

	var subTotal float64
	if invoiceType == InvoiceTypeConsumidorFinal {
		// For consumidor final, subTotal = totalGravada (already includes IVA)
		subTotal = totalGravada
	} else {
		// For credito fiscal, subTotal = totalGravada + totalIva
		subTotal = RoundToResumenPrecision(totalGravada + totalIva)
	}

	return ResumenAmounts{
		TotalNoSuj:          0,
		TotalExenta:         0,
		TotalGravada:        totalGravada,
		SubTotalVentas:      totalGravada,
		TotalDescu:          0,
		TotalIva:            totalIva,
		SubTotal:            subTotal,
		MontoTotalOperacion: subTotal,
		TotalPagar:          subTotal,
	}
}

// Helper functions for rounding
func RoundToItemPrecision(value float64) float64 {
	// Items: up to 8 decimal places
	return math.Round(value*1e8) / 1e8
}

func RoundToResumenPrecision(value float64) float64 {
	// Resumen: exactly 2 decimal places
	return math.Round(value*100) / 100
}
