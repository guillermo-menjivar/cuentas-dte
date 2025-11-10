# Type 11 Export Invoice (Factura de Exportación) - Complete Solution Guide

**Date:** November 10, 2025  
**Status:** ✅ RESOLVED - Successfully submitting to Ministerio de Hacienda  
**Final Result:** PROCESADO (Accepted)

---

## Executive Summary

Export invoices (Type 11) were being rejected by Hacienda with error code 020: `[resumen.montoTotalOperacion] CALCULO INCORRECTO`. After extensive testing and analysis, we discovered that the `montoTotalOperacion` field must include freight and insurance costs, not just the goods value.

**Root Cause:** Incorrect calculation formula for `montoTotalOperacion` in export invoices.

**Solution:** Include `seguro` (insurance) and `flete` (freight) in the `montoTotalOperacion` calculation.

---

## The Problem

### Initial Error
```
Estado: RECHAZADO
Código: 020
Descripción: [resumen.montoTotalOperacion] CALCULO INCORRECTO
```

### What We Were Doing Wrong
```json
{
  "totalGravada": 28439.07,
  "seguro": 284.39,
  "flete": 568.78,
  "montoTotalOperacion": 28439.07,  // ❌ WRONG - Only goods value
  "totalPagar": 29292.24             // ✅ Correct
}
```

The math was correct for `totalPagar`, but `montoTotalOperacion` was missing the freight and insurance costs.

---

## The Solution

### Correct Formula for Type 11 Exports

According to the official Hacienda manual and validated with real export invoice examples:

```
montoTotalOperacion = totalGravada + seguro + flete - descuentos
totalPagar = montoTotalOperacion + totalNoGravado
```

For most export invoices where `totalNoGravado = 0`:
```
montoTotalOperacion = totalPagar
```

### Corrected Code

**File:** `internal/dte/calculator.go`  
**Function:** `CalculateResumenExportacion`

```go
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
    
    // ✅ CRITICAL: Include seguro + flete in montoTotalOperacion
    montoTotalOperacion := RoundToResumenPrecision(totalGravada + seguro + flete - totalDescu)
    
    // totalPagar = montoTotalOperacion when totalNoGravado = 0
    totalPagar := RoundToResumenPrecision(montoTotalOperacion + 0)

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
```

**Key Change:** Line calculating `montoTotalOperacion` now includes `+ seguro + flete`

---

## Understanding Type 11 Export Invoices

### Key Characteristics

1. **Document Type:** `tipoDte: "11"` (Factura de Exportación)
2. **Tax Rate:** 0% IVA (exports are not taxed)
3. **Tributo Code:** `C3` - Identifies export transactions at 0% rate
4. **Currency:** USD only
5. **Special Fields:** Requires `seguro`, `flete`, and Incoterms

### Field Structure

**Line Items (cuerpoDocumento):**
```json
{
  "ventaGravada": 9799.02,  // ← Use this field (not ventaNoSuj)
  "tributos": ["C3"],        // ← Required: marks 0% export rate
  "noGravado": 0,
  "montoDescu": 0
}
```

**Resumen (Summary):**
```json
{
  "totalGravada": 28439.07,        // Sum of all line items
  "seguro": 284.39,                // Insurance cost
  "flete": 568.78,                 // Freight cost
  "montoTotalOperacion": 29292.24, // totalGravada + seguro + flete
  "totalNoGravado": 0,             // Usually 0 for exports
  "totalPagar": 29292.24,          // montoTotalOperacion + totalNoGravado
  "codIncoterms": "02",            // Required Incoterm code
  "descIncoterms": "FCA-Libre transportista"
}
```

**Emisor (Issuer):**
```json
{
  "tipoItemExpor": 1,    // 1=Goods, 2=Services, 3=Both
  "recintoFiscal": "01", // Required fiscal precinct code
  "regimen": "EX-1.1000.000" // Required export regime code
}
```

---

## Common Misconceptions We Discovered

### ❌ Myth 1: Use `ventaNoSuj` field for exports
**Reality:** The Type 11 schema does NOT have a `ventaNoSuj` field. Use `ventaGravada` with C3 tributo to mark it as 0% export rate.

### ❌ Myth 2: Remove tributos for exports
**Reality:** The C3 tributo is REQUIRED by schema. It marks the transaction as a 0% export.

### ❌ Myth 3: `montoTotalOperacion` = only goods value
**Reality:** For Type 11, `montoTotalOperacion` MUST include `seguro + flete`.

### ❌ Myth 4: Move `tipoItemExpor` to resumen
**Reality:** `tipoItemExpor` belongs in the `emisor` section and is required there by schema.

---

## Testing Evidence

### Test Invoice Data
- Beer: 93 units @ $1.50 = $139.50
- Mouse: 43 units @ $99.99 = $4,299.57
- Laptop: 20 units @ $1,200.00 = $24,000.00
- **Total Goods:** $28,439.07
- **Insurance:** $284.39
- **Freight:** $568.78

### Final Accepted Values
```json
{
  "totalGravada": 28439.07,
  "seguro": 284.39,
  "flete": 568.78,
  "totalDescu": 0,
  "montoTotalOperacion": 29292.24,  // ✅ Includes seguro + flete
  "totalNoGravado": 0,
  "totalPagar": 29292.24
}
```

### Hacienda Response
```
Estado: PROCESADO
Código de Generación: D7A65D5F-F004-40DD-87B3-80063CBBA2A4
Sello Recibido: 20254C05484414EF449DB193BEFB619BDBADJY9D
Fecha Procesamiento: 10/11/2025 13:33:04
```

---

## Key Learnings

### 1. Schema is King
The JSON schema (`fe-fex-v1.json`) is the authoritative source. Official examples validate interpretation.

### 2. Field Naming Can Be Misleading
`ventaGravada` (taxable sales) is used for 0% exports because the C3 tributo indicates the special rate.

### 3. Formula Varies by Document Type
The calculation for `montoTotalOperacion` differs between:
- **Type 01 (Domestic):** Usually `totalGravada + totalIva`
- **Type 11 (Export):** `totalGravada + seguro + flete`

### 4. Rounding Matters
Always use 2-decimal precision for resumen amounts. Hacienda allows ±$0.01 tolerance.

### 5. Official Examples Are Gold
The PDF example from Hacienda clarified the field ordering and formula expectations.

---

## Troubleshooting Guide

### Error 020: [resumen.montoTotalOperacion] CALCULO INCORRECTO

**Cause:** The `montoTotalOperacion` value doesn't match Hacienda's expected formula.

**Solution for Type 11:**
```go
montoTotalOperacion = totalGravada + seguro + flete - totalDescu
```

### Error 020: [cuerpoDocumento.item.X] CALCULO INCORRECTO

**Cause:** Line item amounts don't match expected values or have invalid fields.

**Common Issues:**
- Using `ventaNoSuj` (not in schema)
- Missing required `tributos: ["C3"]`
- Incorrect `ventaGravada` calculation

### Schema Validation Failed: Additional property ventaNoSuj

**Cause:** Trying to use a field that doesn't exist in the Type 11 schema.

**Solution:** Use `ventaGravada` instead with C3 tributo.

---

## Code Changes Summary

### Files Modified
1. `internal/dte/calculator.go` - Updated `CalculateResumenExportacion`

### Changes Made
**Before:**
```go
montoTotalOperacion := RoundToResumenPrecision(totalGravada - totalDescu)
totalPagar := RoundToResumenPrecision(totalGravada + seguro + flete - totalDescu)
```

**After:**
```go
montoTotalOperacion := RoundToResumenPrecision(totalGravada + seguro + flete - totalDescu)
totalPagar := RoundToResumenPrecision(montoTotalOperacion + 0)
```

---

## Reference Documents

1. **Schema:** `fe-fex-v1.json` - Type 11 Export Invoice JSON Schema
2. **Manual:** "02 Manual funcional del Sistema DTE" - Official Hacienda documentation
3. **Example:** Real export invoice PDF showing correct field structure
4. **Validation:** Hacienda test environment responses

---

## Additional Notes

### Municipality Code Warning
You may see: `Warning: Invalid municipality code for emisor: dept=06, mun=14`

This is a validation warning but doesn't block submission. However, you should fix it:
- Department 06 = San Salvador
- Valid municipality codes: Use "01" for San Salvador Centro
- Check official MH catalog for other valid codes

### Other Document Types Supported
The same codebase also handles:
- Type 01: Factura Consumidor Final (B2C)
- Type 03: Comprobante de Crédito Fiscal (B2B)
- Type 05: Nota de Crédito
- Type 06: Nota de Débito

Each type may have different calculation rules for `montoTotalOperacion`.

---

## Success Metrics

- ✅ Schema validation passes
- ✅ Hacienda accepts DTE with "PROCESADO" status
- ✅ No error code 020
- ✅ Sello Recibido generated
- ✅ Invoice can be legally used for export transactions

---

## Future Considerations

### When Adding New Export Features
1. Always check the schema first
2. Validate with official examples
3. Test in Hacienda's test environment
4. Verify all resumen calculations match expected formulas

### When Handling Discounts
If you add global discounts (`descuento`, `porcentajeDescuento`):
- Subtract from `totalGravada` first
- Then add `seguro + flete` to get `montoTotalOperacion`
- Formula remains: `montoTotalOperacion = (totalGravada - descuentos) + seguro + flete`

### When Using totalNoGravado
If you ever have non-taxable charges beyond freight/insurance:
- Put them in `totalNoGravado`
- Formula: `totalPagar = montoTotalOperacion + totalNoGravado`
- Currently always 0 for your use case

---

## Team Acknowledgments

**Resolved by:** AI Assistant (Claude) + Development Team  
**Key Contributors:**
- Accountant feedback on Hacienda regulations
- Official schema documentation
- Real export invoice examples
- Multiple test iterations

**Time to Resolution:** ~4 hours of debugging and testing  
**Attempts Before Success:** 15+ iterations

---

## Conclusion

The key to resolving Type 11 export invoice rejections was understanding that `montoTotalOperacion` must include ALL operational costs (goods + insurance + freight), not just the goods value. This differs from our initial interpretation but aligns with Hacienda's official documentation and real-world examples.

The formula is simple but critical:
```
montoTotalOperacion = totalGravada + seguro + flete - descuentos
```

With this fix, export invoices now submit successfully to Hacienda with "PROCESADO" status.

---

**Document Version:** 1.0  
**Last Updated:** November 10, 2025  
**Status:** Production-Ready ✅
