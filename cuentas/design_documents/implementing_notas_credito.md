Perfect! Let's Build Clean 3-Layer Architecture
Architecture Overview
┌─────────────────────────────────────────────────────────────┐
│ LAYER 1: API Handler (HTTP/Gin)                            │
│ • Parse requests                                            │
│ • HTTP validation (binding)                                 │
│ • Call business service                                     │
│ • Return responses                                          │
└────────────────┬────────────────────────────────────────────┘
                 │
                 ↓
┌─────────────────────────────────────────────────────────────┐
│ LAYER 2: Business Logic Service                            │
│ • Validate business rules                                   │
│ • Calculate totals (reuse CCF calculator)                  │
│ • Detect full annulment                                     │
│ • Orchestrate database operations                          │
│ • Call DTE service when finalizing                         │
└────────────────┬────────────────────────────────────────────┘
                 │
                 ↓
┌─────────────────────────────────────────────────────────────┐
│ LAYER 3: DTE Service (Hacienda Integration)                │
│ • Build DTE JSON                                            │
│ • Sign with Firmador                                        │
│ • Submit to Hacienda                                        │
│ • Save response                                             │
│ • Log to commit log                                         │
└─────────────────────────────────────────────────────────────┘

Implementation Plan
Let's build in this order:
Phase 1: Business Logic Service ✅ START HERE
File: internal/services/nota_credito_service.go
Responsibilities:

ValidateCreditRequest() - Business rule validation
ValidateCreditLineItem() - Line item validation
CheckForExistingCredits() - Prevent over-crediting
CalculateCreditTotals() - Calculate financial totals
IsFullAnnulment() - Detect 100% voiding
CreateNotaCredito() - Orchestrate DB operations
FinalizeNotaCredito() - Finalize and trigger DTE

Phase 2: DTE Service
File: internal/dte/service_notas_credito.go
Responsibilities:

ProcessNotaCredito() - Main DTE flow
saveNotaCreditoHaciendaResponse() - Save response
logNotaCreditoToCommitLog() - Commit log (one per CCF)

Phase 3: DTE Builder
File: internal/dte/builder_nota_credito.go
Responsibilities:

BuildNotaCredito() - Build complete DTE
buildNotaCreditoIdentificacion() - Identificacion section
buildNotaCreditoEmisor() - Emisor section
buildNotaCreditoReceptor() - Receptor section
buildNotaCreditoCuerpoDocumento() - Line items
buildNotaCreditoResumen() - Financial summary
buildNotaCreditoExtension() - Extension
buildNotaCreditoDocumentosRelacionados() - CCF references

Phase 4: API Handler
File: internal/handlers/notas_credito.go
Responsibilities:

CreateNotaCredito() - POST /v1/notas/credito
GetNotaCredito() - GET /v1/notas/credito/:id
FinalizeNotaCredito() - POST /v1/notas/credito/:id/finalize
ListNotasCredito() - GET /v1/notas/credito (optional)

Phase 5: Routes & Integration
Files:

cmd/api/routes.go - Register routes
cmd/api/main.go - Wire up services


Let's Start with Phase 1: Business Logic Service
Ready to see the code? I'll create a production-ready, clean service with:

✅ Proper error handling
✅ Transaction management
✅ Validation logic
✅ Calculation logic
✅ Full documentation
✅ Ready for testing

Should I proceed with creating internal/services/nota_credito_service.go
