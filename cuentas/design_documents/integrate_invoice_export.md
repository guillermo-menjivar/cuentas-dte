# Service Layer Updates - Factura de Exportación (Separate Methods)

## 📁 Files to Update

```
internal/services/
├── invoice_service.go (NO CHANGES - keep as-is)
└── invoice_service_export.go (NEW - add this file)

internal/dte/
└── calculator.go (UPDATE - add export calculation methods)
```

**Approach:** Keep export logic completely separate to avoid side effects. Refactor later in a dedicated milestone.

---

## ✅ Step 1: Update Calculator

**File:** `internal/dte/calculator.go`

**Add these two methods** (at the end of the file, in the item calculations section):

```go
// Copy from: calculator_export_methods.go
```

This adds:
- ✅ `CalculateExportacion()` - 0% IVA item calculation
- ✅ `CalculateResumenExportacion()` - Export invoice summary with seguro & flete

---

## ✅ Step 2: Create New Export Service File

**Create:** `internal/services/invoice_service_export.go`

**Copy ALL content from:** `invoice_service_export_methods.go`

This adds **separate** methods with **no conflicts**:
- ✅ `insertExportDocuments()` - Save export documents
- ✅ `getExportDocuments()` - Load export documents  
- ✅ `processLineItemsExport()` - Process items with 0% IVA
- ✅ `insertInvoiceExport()` - Insert with export fields (separate from insertInvoice)
- ✅ `getInvoiceHeaderExport()` - Query with export fields (separate from getInvoiceHeader)
- ✅ `GetInvoiceExport()` - Get complete export invoice (separate from GetInvoice)

**Key point:** All methods have **Export** suffix or different names - NO conflicts!

---

## ✅ Step 3: Update CreateInvoice to Route to Export Methods

**File:** `internal/services/invoice_service.go`

**Replace the entire `CreateInvoice()` method** with:

```go
// Copy from: create_invoice_updated.go
```

**Key changes:**
```go
// Uses separate methods based on invoice type:
if req.IsExportInvoice() {
    invoiceID, err = s.insertInvoiceExport(ctx, tx, invoice)  // Export version
} else {
    invoiceID, err = s.insertInvoice(ctx, tx, invoice)        // Regular version
}
```

---

## 📋 Summary of Changes

### NO Changes to Existing Methods
- ✅ `insertInvoice()` - **unchanged**
- ✅ `getInvoiceHeader()` - **unchanged**
- ✅ `GetInvoice()` - **unchanged**

### New Separate Methods (invoice_service_export.go)
- ✅ `insertInvoiceExport()` - handles export fields
- ✅ `getInvoiceHeaderExport()` - loads export fields
- ✅ `GetInvoiceExport()` - complete export invoice
- ✅ `processLineItemsExport()` - 0% IVA processing
- ✅ `insertExportDocuments()` - export docs
- ✅ `getExportDocuments()` - export docs

### Updated Method (invoice_service.go)
- ✅ `CreateInvoice()` - routes to export methods when needed

---

## 🧪 Testing Changes

After making these changes:

```bash
# Test compilation
cd internal/services
go build

cd internal/dte
go build
```

---

## 🎯 Benefits of This Approach

✅ **Zero side effects** - Regular invoices unchanged  
✅ **Easy to test** - Test export separately  
✅ **Clear separation** - Easy to find export code  
✅ **Refactor later** - Clean milestone for DRY refactor  

---

## 📝 Refactor Milestone (Later)

In a future milestone, you can:
1. Create `insertInvoiceUnified()` that handles both
2. Deprecate separate methods
3. Add feature flags if needed
4. Test thoroughly before switching

**For now: Keep separate, ship fast!** ✅
