# Integration Guide: Adding Nota de Crédito to Existing NotaHandler

## Step 1: Update NotaHandler Struct

**File:** `internal/handlers/nota_handler.go`

**BEFORE:**
```go
type NotaHandler struct {
	notaService    *services.NotaService
	invoiceService *services.InvoiceService
}
```

**AFTER:**
```go
type NotaHandler struct {
	notaService         *services.NotaService
	notaCreditoService  *services.NotaCreditoService  // ← ADD THIS
	invoiceService      *services.InvoiceService
}
```

---

## Step 2: Update Constructor

**BEFORE:**
```go
func NewNotaHandler(
	notaService *services.NotaService,
	invoiceService *services.InvoiceService,
) *NotaHandler {
	return &NotaHandler{
		notaService:    notaService,
		invoiceService: invoiceService,
	}
}
```

**AFTER:**
```go
func NewNotaHandler(
	notaService *services.NotaService,
	notaCreditoService *services.NotaCreditoService,  // ← ADD THIS
	invoiceService *services.InvoiceService,
) *NotaHandler {
	return &NotaHandler{
		notaService:        notaService,
		notaCreditoService: notaCreditoService,  // ← ADD THIS
		invoiceService:     invoiceService,
	}
}
```

---

## Step 3: Add Three New Methods

**Copy these three methods to the END of your `nota_handler.go` file:**

1. `CreateNotaCredito(c *gin.Context)`
2. `GetNotaCredito(c *gin.Context)`
3. `FinalizeNotaCredito(c *gin.Context)`

*(See nota_credito_handlers_to_add.go for the complete method implementations)*

---

## Step 4: Update Route Registration

**File:** Where you register your routes (e.g., `cmd/server/routes.go`)

**ADD these imports:**
```go
import (
	"cuentas/internal/services"
)
```

**UPDATE handler initialization:**

**BEFORE:**
```go
notaHandler := handlers.NewNotaHandler(
	notaService,
	invoiceService,
)
```

**AFTER:**
```go
// Initialize Nota de Crédito service
notaCreditoService := services.NewNotaCreditoService()

// Initialize handler with BOTH services
notaHandler := handlers.NewNotaHandler(
	notaService,
	notaCreditoService,  // ← ADD THIS
	invoiceService,
)
```

**ADD new routes:**
```go
// Existing routes
notasDebito := v1.Group("/notas/debito")
{
	notasDebito.POST("", notaHandler.CreateNotaDebito)
	notasDebito.GET("/:id", notaHandler.GetNotaDebito)
	notasDebito.POST("/:id/finalize", notaHandler.FinalizeNotaDebito)
}

// NEW: Nota de Crédito routes
notasCredito := v1.Group("/notas/credito")
{
	notasCredito.POST("", notaHandler.CreateNotaCredito)
	notasCredito.GET("/:id", notaHandler.GetNotaCredito)
	notasCredito.POST("/:id/finalize", notaHandler.FinalizeNotaCredito)
}
```

---

## Final File Structure

```
internal/
├── handlers/
│   └── nota_handler.go          ← Updated (added notaCreditoService + 3 methods)
├── services/
│   ├── notas_debito.go          ← Existing
│   └── notas_credito.go         ← NEW (nota_credito_service_fixed.go)
└── dte/
    ├── builder_notas_debito.go  ← Existing
    ├── builder_notas_credito.go ← NEW
    ├── service_notas_debito.go  ← Existing
    └── service_notas_credito.go ← NEW
```

---

## API Endpoints Created

### Nota de Crédito Endpoints:

1. **Create (Draft)**
   - `POST /v1/notas/credito`
   - Body: `CreateNotaCreditoRequest`
   - Returns: Nota in draft status

2. **Get by ID**
   - `GET /v1/notas/credito/:id`
   - Returns: Nota with line items and CCF references

3. **Finalize & Submit**
   - `POST /v1/notas/credito/:id/finalize`
   - Generates numero control
   - Processes DTE (builds, signs, submits to Hacienda)
   - Logs to commit log (ONE ENTRY PER CCF)

---

## Testing

```bash
# 1. Create a nota de crédito (draft)
curl -X POST http://localhost:8080/v1/notas/credito \
  -H "Content-Type: application/json" \
  -d '{
    "ccf_ids": ["ccf-uuid-1", "ccf-uuid-2"],
    "credit_reason": "Producto defectuoso",
    "credit_description": "Cliente reportó defecto de fábrica",
    "payment_terms": "contado",
    "line_items": [
      {
        "related_ccf_id": "ccf-uuid-1",
        "ccf_line_item_id": "line-item-uuid-1",
        "quantity_credited": 5.0,
        "credit_amount": 10.50,
        "credit_reason": "Defecto en producto"
      }
    ]
  }'

# 2. Get the nota
curl http://localhost:8080/v1/notas/credito/{nota_id}

# 3. Finalize and submit to Hacienda
curl -X POST http://localhost:8080/v1/notas/credito/{nota_id}/finalize
```

---

## ✅ Complete Integration Checklist

- [ ] Update `NotaHandler` struct (add `notaCreditoService`)
- [ ] Update `NewNotaHandler` constructor
- [ ] Add three new methods to `nota_handler.go`
- [ ] Initialize `NotaCreditoService` in route setup
- [ ] Update handler initialization to pass `notaCreditoService`
- [ ] Register new routes for `/notas/credito`
- [ ] Test all three endpoints

---

## Files to Copy to Your Project

1. **Service Layer**
   - Copy `nota_credito_service_fixed.go` → `internal/services/notas_credito.go`

2. **DTE Layer**
   - Copy `builder_notas_credito.go` → `internal/dte/builder_notas_credito.go`
   - Copy `service_notas_credito.go` → `internal/dte/service_notas_credito.go`

3. **Handler Layer**
   - Modify existing `internal/handlers/nota_handler.go` using the changes above

That's it! 🚀
