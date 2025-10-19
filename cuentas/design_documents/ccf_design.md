## feat: Add support for CCF (Crédito Fiscal) invoice type

### Summary
Implemented complete support for CCF (Comprobante de Crédito Fiscal) invoices to comply with El Salvador's Ministerio de Hacienda DTE v3 requirements. CCF invoices follow different calculation rules than consumer invoices (Factura) and are used for B2B transactions.

### Changes

#### New Calculator Function
- **Added `CalculateResumenCCF()`** in `internal/dte/calculator.go`
  - Dedicated calculation logic for CCF resumen (summary) totals
  - Calculates IVA on total `ventaGravada` rather than per-item to avoid rounding errors
  - Returns `subTotal` equal to `totalGravada` (excluding IVA)
  - Returns `montoTotalOperacion` and `totalPagar` with IVA included
  - Applies proper rounding (`RoundToResumenPrecision`) after all calculations to prevent floating-point precision errors

#### Updated Builder Logic
- **Modified `buildResumen()`** in `internal/dte/builder.go`
  - Added switch statement to route CCF invoices to `CalculateResumenCCF()`
  - Consumer invoices (Factura) continue to use `CalculateResumen()`
  - Added CCF-specific `tributos` array with IVA breakdown:
```json
    {
      "codigo": "20",
      "descripcion": "Impuesto al Valor Agregado 13",
      "valor": <calculated_iva>
    }
```
  - Excluded `TotalIva` field from JSON output for CCF (uses `omitempty`)

### Key Differences: CCF vs Factura

| Field | CCF (Jurídica) | Factura (Natural) |
|-------|----------------|-------------------|
| `subTotal` | = `totalGravada` (no IVA) | = `totalGravada + IVA` |
| `montoTotalOperacion` | = `totalGravada + IVA` | = `subTotal` |
| `totalPagar` | = `totalGravada + IVA` | = `subTotal` |
| `TotalIva` field | Omitted from JSON | Included |
| `tributos` array | Required (shows IVA breakdown) | Null/optional |

### Technical Implementation Details

1. **Isolated CCF Logic**: Created separate calculator to avoid introducing breaking changes to existing Factura implementation
2. **Rounding Strategy**: Applied rounding at critical points to maintain 2-decimal precision:
   - Round `totalGravada` before calculating IVA
   - Round IVA calculation result
   - Round final sum of `totalGravada + IVA` to eliminate floating-point errors
3. **IVA Calculation**: IVA computed on rounded total (not sum of per-item IVA) per Hacienda requirements

### Testing
- ✅ Successfully validated against Hacienda test environment (`ambiente: "00"`)
- ✅ Multiple test invoices submitted and accepted
- ✅ Verified calculation accuracy and JSON schema compliance

### Files Modified
- `internal/dte/calculator.go` - Added `CalculateResumenCCF()`
- `internal/dte/builder.go` - Updated `buildResumen()` with CCF logic
- `internal/dte/builder.go` - Added tributos array population for CCF

### Breaking Changes
None. Changes are additive and isolated to CCF invoice type.

### Related Issues
Fixes validation errors from Hacienda API:
- `[resumen.subTotal] CALCULO INCORRECTO`
- `montoTotalOperacion is not a multiple of 0.01`
- Missing tributos array requirement for CCF
