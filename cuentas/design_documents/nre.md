# Nota de Remisión Electrónica (NRE - Type 04) - Plan of Attack

**Date:** November 10, 2025  
**Document Type:** Type 04 - Nota de Remisión (Delivery Note)  
**Purpose:** Document movement of goods without immediate sale

---

## 📋 What is a Nota de Remisión?

A **delivery note** used to document the physical movement of goods:
- Between warehouses
- From warehouse to customer (before invoicing)
- To consignees or temporary locations
- **NOT a sale** - just goods movement tracking

Later, an actual invoice (Type 01 or 03) can reference this NRE.

---

## 🎯 Key Differences from Other DTE Types

| Feature | Type 01/03 (Invoice) | Type 11 (Export) | **Type 04 (NRE)** |
|---------|---------------------|------------------|-------------------|
| Purpose | Sale transaction | Export sale | **Goods movement** |
| IVA? | Yes (13%) | No (0% with C3) | **No IVA at all** |
| Payment | Required | Required | **Not applicable** |
| Receptor | Required (customer) | Required (foreign) | **Can be null!** |
| Related Docs | Optional | Optional | **Often references invoice** |
| Resumen | With IVA totals | With seguro/flete | **Simple totals only** |

---

## 📊 Schema Analysis - Type 04 Structure

### 1. Identificacion
```json
{
  "version": 3,           // NRE is version 3
  "tipoDte": "04",        // Type 04
  "numeroControl": "DTE-04-M038P019-000000000000001",
  "codigoGeneracion": "UUID",
  "ambiente": "00",
  "fecEmi": "2025-11-10",
  "horEmi": "14:30:00",
  "tipoMoneda": "USD"
}
```

### 2. DocumentoRelacionado (OPTIONAL)
```json
{
  "documentoRelacionado": [
    {
      "tipoDocumento": "01",      // Only "01" or "03" allowed
      "tipoGeneracion": 2,         // 1=physical, 2=electronic
      "numeroDocumento": "UUID",   // UUID if electronic
      "fechaEmision": "2025-11-10"
    }
  ]
  // OR null if no related docs
}
```

**Rules:**
- Up to 50 documents
- All same type ("01" OR "03", not mixed)
- Cannot reference Type 11 (exports) or Type 05/06 (credit/debit notes)

### 3. Emisor
Same as other DTEs - no special fields.

### 4. Receptor (CAN BE NULL!)
```json
{
  "receptor": null  // OK for internal warehouse transfers
  // OR full receptor object for external deliveries
}
```

### 5. CuerpoDocumento
```json
[
  {
    "numItem": 1,
    "tipoItem": 1,              // 1=goods, 2=service
    "cantidad": 100.0,
    "codigo": "PROD-001",
    "uniMedida": 59,
    "descripcion": "Product Name",
    "precioUni": 10.00,
    "montoDescu": 0.00,
    "ventaNoSuj": 1000.00,      // ← IMPORTANT: Use ventaNoSuj!
    "ventaExenta": 0.00,
    "ventaGravada": 0.00,
    "tributos": null            // ← NO tributos
  }
]
```

**Key Point:** NRE typically uses `ventaNoSuj` (not subject to tax) since it's just goods movement, not a sale.

### 6. Resumen (SIMPLER than invoices!)
```json
{
  "totalNoSuj": 1000.00,         // Sum of ventaNoSuj
  "totalExenta": 0.00,
  "totalGravada": 0.00,
  "subTotalVentas": 1000.00,     // = totalNoSuj + totalExenta + totalGravada
  "descuNoSuj": 0.00,
  "descuExenta": 0.00,
  "descuGravada": 0.00,
  "porcentajeDescuento": null,   // Can be null
  "totalDescu": 0.00,
  "tributos": null,              // Usually null (no IVA)
  "subTotal": 1000.00,           // = subTotalVentas - totalDescu
  "montoTotalOperacion": 1000.00,// = subTotal (no IVA to add)
  "totalLetras": "MIL DÓLARES"
}
```

**Formula for NRE:**
```
subTotalVentas = totalNoSuj + totalExenta + totalGravada
subTotal = subTotalVentas - totalDescu
montoTotalOperacion = subTotal (no IVA!)
```

Much simpler than invoices!

---

## 🏗️ Implementation Plan

### Phase 1: Database & Models (1-2 hours)

#### 1.1 Database Schema
Check if you need new tables/fields:
- [ ] `invoices` table: Add `is_remision` flag?
- [ ] Or create `remisiones` table?
- [ ] Track related documents (documentoRelacionado)
- [ ] Extension fields: delivery person, recipient

**Decision Point:** Can you reuse existing invoice tables or need separate tables?

#### 1.2 Go Models
- [ ] Create `NotaRemisionElectronica` struct (similar to `DTE`)
- [ ] `NotaRemisionCuerpoItem` struct
- [ ] `NotaRemisionResumen` struct
- [ ] Related document tracking

### Phase 2: Calculator Functions (1 hour)

#### 2.1 New Calculator Method
```go
func (c *Calculator) CalculateNotaRemision(
    pricePerUnit float64,
    quantity float64,
    discount float64,
    itemType string, // "noSuj", "exenta", "gravada"
) ItemAmounts {
    // Calculate based on item type
    // Most common: ventaNoSuj (goods movement)
}
```

#### 2.2 Resumen Calculator
```go
func (c *Calculator) CalculateResumenNotaRemision(
    items []ItemAmounts,
) ResumenAmounts {
    // Sum totals
    // NO IVA calculations needed!
    // NO seguro/flete like exports
}
```

### Phase 3: Builder Functions (2-3 hours)

#### 3.1 Main Builder
```go
func (b *Builder) BuildNotaRemision(
    ctx context.Context,
    remision *models.Remision,
) ([]byte, error) {
    // Build NRE structure
    // Handle optional receptor (can be null!)
    // Handle optional documentoRelacionado
}
```

#### 3.2 Helper Functions
- `buildNotaRemisionIdentificacion()`
- `buildNotaRemisionCuerpoDocumento()`
- `buildNotaRemisionResumen()`
- `buildDocumentoRelacionado()` - NEW!

### Phase 4: Submission & Validation (1-2 hours)

#### 4.1 Schema Validation
- [ ] Load `fe-nr-v3.json` schema
- [ ] Validate before submission
- [ ] Handle version 3 (not 1 like Type 01)

#### 4.2 Hacienda Submission
Reuse existing submission logic:
- Endpoint: Same (`/fesv/recepciondte`)
- Format: Same (JSON with signed documento)
- `tipoDte`: "04"

### Phase 5: API Endpoints (1 hour)

#### 5.1 REST API
```
POST /v1/remisiones
POST /v1/remisiones/{id}/finalize
GET  /v1/remisiones/{id}
```

OR reuse invoice endpoints with type flag:
```
POST /v1/invoices (with type: "remision")
```

### Phase 6: Testing (2-3 hours)

#### 6.1 Test Cases
1. **Simple remision** - No related docs, no receptor
2. **With receptor** - External delivery
3. **With related invoice** - References Type 01
4. **With discounts** - Item and global discounts
5. **Multiple items** - Mixed ventaNoSuj/exenta

---

## 🎯 Critical Decisions Needed

### Decision 1: Database Structure
**Options:**
A. Reuse `invoices` table with `document_type` = "remision"
B. Create separate `remisiones` table
C. Hybrid: invoices table + remision-specific joins

**Recommendation:** Option A - Reuse invoices table
- Add `document_type` enum: 'invoice', 'export', 'remision', 'credit_note'
- Most logic overlaps with invoices
- Simpler to maintain

### Decision 2: Related Documents Storage
**How to store documentoRelacionado?**
A. JSON field in invoices table
B. Separate `invoice_related_documents` table
C. Both (JSON for DTE, table for queries)

**Recommendation:** Option C
- Store in JSON for DTE generation
- Also in related table for querying

### Decision 3: Receptor Nullability
**How to handle optional receptor?**
- UI: Make receptor optional for remisiones
- DB: Allow NULL for receptor_id
- Validation: Skip receptor validation if null

### Decision 4: Use Cases Priority
**Which scenarios to implement first?**
1. ✅ **Internal warehouse transfer** (no receptor, no related docs)
2. ✅ **Delivery before invoice** (with receptor, no related docs)
3. ⏸️ **Related to invoice** (with documentoRelacionado) - Phase 2
4. ⏸️ **With discounts** - Phase 2

---

## ⚠️ Lessons from Export Invoice (Type 11)

### What We Learned:
1. **Schema is authoritative** - Don't assume field meanings
2. **Read official examples** - PDFs show real structure
3. **Test early and often** - Submit to Hacienda test env
4. **Formula differences** - Each DTE type has unique calculations
5. **Field naming misleads** - "ventaGravada" used differently per type

### Apply to NRE:
- ✅ Read schema thoroughly BEFORE coding
- ✅ Create test with simple case first
- ✅ Validate resumen calculation formula
- ✅ Don't assume fields work like invoices

---

## 📝 Implementation Checklist

### Prerequisites
- [x] Type 11 (Export) working ✅
- [x] Schema file (fe-nr-v3.json) ✅
- [x] Manual documentation ✅
- [ ] Database migration plan
- [ ] UI mockups (if needed)

### Phase 1: Foundation
- [ ] Design database schema
- [ ] Create Go structs
- [ ] Write calculator functions
- [ ] Unit tests for calculations

### Phase 2: Builder
- [ ] Implement BuildNotaRemision
- [ ] Handle optional fields (receptor, relacionado)
- [ ] Schema validation
- [ ] Integration tests

### Phase 3: Submission
- [ ] Hacienda submission logic
- [ ] Error handling
- [ ] Response processing
- [ ] Store sello de recepción

### Phase 4: API
- [ ] REST endpoints
- [ ] Request validation
- [ ] Response formatting
- [ ] API documentation

### Phase 5: Testing
- [ ] Test in Hacienda test environment
- [ ] Verify PROCESADO status
- [ ] Edge case testing
- [ ] Load testing

---

## 🚀 Recommended Approach

### Week 1: MVP - Simple Remision
**Goal:** Get ONE simple remision accepted by Hacienda

**Scope:**
- Internal warehouse transfer
- No receptor (null)
- No related documents (null)
- Single line item
- ventaNoSuj only (no taxes)

**Success Criteria:**
```
Estado: PROCESADO
Sello Recibido: [seal]
```

### Week 2: Full Features
- Add receptor support
- Add documentoRelacionado
- Handle discounts
- Multiple item types
- UI integration

### Week 3: Polish
- Error handling
- Validation messages
- Documentation
- Training

---

## 🔑 Key Formulas - Type 04 (NRE)

### Line Item:
```
ventaNoSuj = (precioUni × cantidad) - montoDescu
```

### Resumen:
```
subTotalVentas = totalNoSuj + totalExenta + totalGravada
subTotal = subTotalVentas - totalDescu
montoTotalOperacion = subTotal
```

**NO IVA! NO seguro/flete!** Much simpler than exports!

---

## 📚 Next Steps

1. **Review this plan** with team
2. **Decide on database approach** (reuse invoices table?)
3. **Create test data** for first simple remision
4. **Implement Phase 1** (calculator functions)
5. **Test calculation logic** before building DTE
6. **Build simple NRE** and submit to test environment
7. **Iterate** based on Hacienda response

---

## 💡 Questions to Answer

Before starting implementation:

1. **Use case:** Why do you need remisiones? (warehouse transfers? deliveries?)
2. **UI:** Do users create remisiones manually or auto-generated?
3. **Workflow:** Remision → Invoice flow? Or standalone?
4. **Inventory:** Does remision affect inventory? (probably yes)
5. **Permissions:** Same as invoices or separate permissions?

---

**Ready to start?** Let's begin with Phase 1: Calculator functions! 🚀
