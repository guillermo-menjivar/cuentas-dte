# Nota de Remisión Electrónica - Production Implementation Plan
## Based on Real Client Scenarios

**Date:** November 10, 2025  
**Goal:** Production-grade NRE system supporting 3 core use cases

---

## 🎯 Client Use Cases to Support

### Use Case 1: Pre-Invoice Delivery (Factory → Customer)
**Scenario:** Industrias López sends 500 plastic containers to customer before invoice is ready
- **Emisor:** Factory (your client)
- **Receptor:** Customer (known entity)
- **Amount:** $0.00 (no sale yet)
- **Related Docs:** null initially, later invoice references this NRE
- **Key Field:** `observaciones` = "Entrega previa a facturación"

### Use Case 2: Inter-Branch Transfer (Internal Movement)
**Scenario:** Supermercado El Faro transfers 200 boxes rice + 100 water pallets between branches
- **Emisor:** Branch A (San Miguel)
- **Receptor:** Branch B (Santa Ana) - SAME company
- **Amount:** $0.00 (internal transfer)
- **Related Docs:** null
- **Key Field:** `observaciones` = "Traslado de mercancía entre sucursales"

### Use Case 3: Route Sales with Extra Inventory (Tortillería)
**Scenario:** Delivery truck carries confirmed orders + extra product for opportunistic sales
- **Emisor:** Tortillería El Buen Maíz
- **Receptor:** "CONSUMIDOR FINAL" (unknown at departure)
- **Amount:** $0.00 (no sale yet)
- **Related Docs:** null initially
- **Extension:** Driver name and vehicle plate
- **Workflow:** NRE → Multiple invoices during route → Each invoice references NRE

---

## 🏗️ Production System Architecture

### Database Design

#### Option A: Extend Invoices Table (RECOMMENDED)
```sql
ALTER TABLE invoices ADD COLUMN document_type VARCHAR(20) DEFAULT 'invoice';
-- Values: 'invoice', 'ccf', 'export', 'remision', 'credit_note', 'debit_note'

ALTER TABLE invoices ADD COLUMN remision_type VARCHAR(50);
-- Values: 'pre_invoice_delivery', 'inter_branch_transfer', 'route_sales', 'other'

ALTER TABLE invoices ADD COLUMN receptor_id UUID NULL;
-- Allow NULL for internal transfers or unknown recipients

ALTER TABLE invoices ADD COLUMN delivery_person VARCHAR(200);
ALTER TABLE invoices ADD COLUMN vehicle_plate VARCHAR(20);
ALTER TABLE invoices ADD COLUMN delivery_notes TEXT;

-- For tracking which invoices reference this remision
CREATE TABLE remision_invoice_links (
    id UUID PRIMARY KEY,
    remision_id UUID REFERENCES invoices(id),
    invoice_id UUID REFERENCES invoices(id),
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(remision_id, invoice_id)
);
```

**Why this approach:**
- ✅ Reuses 90% of existing invoice logic
- ✅ Same line items table
- ✅ Same establishment/POS tracking
- ✅ Easier to query "all documents"
- ✅ Simpler codebase maintenance

#### Related Documents Table
```sql
CREATE TABLE invoice_related_documents (
    id UUID PRIMARY KEY,
    invoice_id UUID REFERENCES invoices(id),
    related_document_type VARCHAR(2),  -- "01", "03", etc
    related_generation_type INT,       -- 1=physical, 2=electronic
    related_document_number VARCHAR(36), -- UUID or physical number
    related_emission_date DATE,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_invoice_related ON invoice_related_documents(invoice_id);
CREATE INDEX idx_related_doc ON invoice_related_documents(related_document_number);
```

---

## 📋 Implementation Phases

### Phase 1: Core NRE Functionality (Week 1)
**Goal:** Support Use Case 1 & 2 (known receptor scenarios)

#### 1.1 Database Migration
```bash
# Create migration file
db/migrations/YYYYMMDD_add_remision_support.sql
```

- [ ] Add document_type enum
- [ ] Add remision-specific fields
- [ ] Create related_documents table
- [ ] Add indexes
- [ ] Migrate existing data (all = 'invoice')

#### 1.2 Models Update
```go
// internal/models/invoice.go

type DocumentType string

const (
    DocumentTypeInvoice      DocumentType = "invoice"
    DocumentTypeCCF          DocumentType = "ccf"
    DocumentTypeExport       DocumentType = "export"
    DocumentTypeRemision     DocumentType = "remision"      // ← NEW
    DocumentTypeCreditNote   DocumentType = "credit_note"
    DocumentTypeDebitNote    DocumentType = "debit_note"
)

type RemisionType string

const (
    RemisionPreInvoice    RemisionType = "pre_invoice_delivery"
    RemisionInterBranch   RemisionType = "inter_branch_transfer"
    RemisionRouteSales    RemisionType = "route_sales"
    RemisionOther         RemisionType = "other"
)

type Invoice struct {
    // ... existing fields ...
    DocumentType    DocumentType  `json:"document_type"`
    RemisionType    *RemisionType `json:"remision_type,omitempty"`
    DeliveryPerson  *string       `json:"delivery_person,omitempty"`
    VehiclePlate    *string       `json:"vehicle_plate,omitempty"`
    DeliveryNotes   *string       `json:"delivery_notes,omitempty"`
    ReceptorID      *string       `json:"receptor_id"` // NULL for internal
}

type RelatedDocument struct {
    ID                    string    `json:"id"`
    InvoiceID             string    `json:"invoice_id"`
    RelatedDocumentType   string    `json:"related_document_type"`   // "01", "03"
    RelatedGenerationType int       `json:"related_generation_type"` // 1 or 2
    RelatedDocumentNumber string    `json:"related_document_number"` // UUID
    RelatedEmissionDate   time.Time `json:"related_emission_date"`
}
```

#### 1.3 Calculator Functions
```go
// internal/dte/calculator.go

// For remision: amounts are typically $0.00
func (c *Calculator) CalculateRemisionItem(
    quantity float64,
    unitPrice float64, // For tracking, not for charging
    discount float64,
) ItemAmounts {
    // Most remisiones have $0 totals
    // But track quantities for inventory
    
    return ItemAmounts{
        PrecioUni:    RoundToItemPrecision(unitPrice),
        VentaGravada: 0, // No sale
        VentaNoSuj:   0, // Movement only
        VentaExenta:  0,
        IvaItem:      0,
        MontoDescu:   0,
    }
}

func (c *Calculator) CalculateResumenRemision(
    items []ItemAmounts,
) ResumenAmounts {
    // For remision: everything is typically zero
    
    return ResumenAmounts{
        TotalNoSuj:          0,
        TotalExenta:         0,
        TotalGravada:        0,
        SubTotalVentas:      0,
        TotalDescu:          0,
        TotalIva:            0,
        SubTotal:            0,
        MontoTotalOperacion: 0,
        TotalPagar:          0,
    }
}
```

**Key Insight:** Remisiones are almost always $0.00 because they document movement, not sales!

#### 1.4 DTE Builder
```go
// internal/dte/builder_remision.go

type NotaRemisionElectronica struct {
    Identificacion       Identificacion           `json:"identificacion"`
    DocumentoRelacionado *[]DocumentoRelacionado  `json:"documentoRelacionado"`
    Emisor               Emisor                   `json:"emisor"`
    Receptor             *Receptor                `json:"receptor"` // Can be null!
    OtrosDocumentos      *[]OtroDocumento         `json:"otrosDocumentos"`
    VentaTercero         *VentaTercero            `json:"ventaTercero"`
    CuerpoDocumento      []CuerpoDocumentoItem    `json:"cuerpoDocumento"`
    Resumen              ResumenRemision          `json:"resumen"`
    Extension            *ExtensionRemision       `json:"extension"`
    Apendice             *[]Apendice              `json:"apendice"`
}

type ResumenRemision struct {
    TotalNoSuj           float64    `json:"totalNoSuj"`
    TotalExenta          float64    `json:"totalExenta"`
    TotalGravada         float64    `json:"totalGravada"`
    SubTotalVentas       float64    `json:"subTotalVentas"`
    DescuNoSuj           float64    `json:"descuNoSuj"`
    DescuExenta          float64    `json:"descuExenta"`
    DescuGravada         float64    `json:"descuGravada"`
    PorcentajeDescuento  *float64   `json:"porcentajeDescuento"` // Can be null
    TotalDescu           float64    `json:"totalDescu"`
    Tributos             *[]Tributo `json:"tributos"` // Usually null
    SubTotal             float64    `json:"subTotal"`
    MontoTotalOperacion  float64    `json:"montoTotalOperacion"`
    TotalLetras          string     `json:"totalLetras"`
}

type ExtensionRemision struct {
    NombEntrega   *string `json:"nombEntrega"`   // Delivery person
    DocuEntrega   *string `json:"docuEntrega"`   // DUI of delivery person
    NombRecibe    *string `json:"nombRecibe"`    // Recipient name
    DocuRecibe    *string `json:"docuRecibe"`    // DUI of recipient
    Observaciones *string `json:"observaciones"` // Notes
    PlacaVehiculo *string `json:"placaVehiculo"` // Vehicle plate
}

func (b *Builder) BuildNotaRemision(
    ctx context.Context,
    invoice *models.Invoice,
) ([]byte, error) {
    // Validate it's a remision
    if invoice.DocumentType != models.DocumentTypeRemision {
        return nil, fmt.Errorf("not a remision document")
    }
    
    // Load company & establishment
    company, err := b.loadCompany(ctx, invoice.CompanyID)
    if err != nil {
        return nil, fmt.Errorf("load company: %w", err)
    }
    
    establishment, err := b.loadEstablishmentAndPOS(ctx, invoice.EstablishmentID, invoice.PointOfSaleID)
    if err != nil {
        return nil, fmt.Errorf("load establishment: %w", err)
    }
    
    // Load receptor (if any)
    var receptor *Receptor
    if invoice.ReceptorID != nil {
        client, err := b.loadClient(ctx, *invoice.ReceptorID)
        if err != nil {
            return nil, fmt.Errorf("load client: %w", err)
        }
        receptor = b.buildReceptor(client)
    }
    // If null, it's an internal transfer!
    
    // Load related documents (if any)
    var documentoRelacionado *[]DocumentoRelacionado
    relatedDocs, err := b.loadRelatedDocuments(ctx, invoice.ID)
    if err != nil {
        return nil, fmt.Errorf("load related docs: %w", err)
    }
    if len(relatedDocs) > 0 {
        docs := b.buildDocumentosRelacionados(relatedDocs)
        documentoRelacionado = &docs
    }
    
    // Build line items (typically $0 amounts)
    cuerpoDocumento, _ := b.buildRemisionCuerpoDocumento(invoice)
    
    // Build resumen (typically all zeros)
    resumen := b.buildRemisionResumen(invoice)
    
    // Build extension (delivery info)
    extension := b.buildRemisionExtension(invoice)
    
    nre := &NotaRemisionElectronica{
        Identificacion:       b.buildRemisionIdentificacion(invoice, company),
        DocumentoRelacionado: documentoRelacionado,
        Emisor:               b.buildEmisor(company, establishment),
        Receptor:             receptor, // Can be null!
        OtrosDocumentos:      nil,
        VentaTercero:         nil,
        CuerpoDocumento:      cuerpoDocumento,
        Resumen:              resumen,
        Extension:            extension,
        Apendice:             nil,
    }
    
    // Marshal and validate
    jsonBytes, err := json.Marshal(nre)
    if err != nil {
        return nil, err
    }
    
    // Validate against schema
    if err := dte_schemas.Validate("04", jsonBytes); err != nil {
        return nil, fmt.Errorf("schema validation: %w", err)
    }
    
    return jsonBytes, nil
}

func (b *Builder) buildRemisionIdentificacion(
    invoice *models.Invoice,
    company *CompanyData,
) Identificacion {
    return Identificacion{
        Version:          3, // NRE is version 3
        Ambiente:         company.DTEAmbiente,
        TipoDte:          "04", // Nota de Remisión
        NumeroControl:    invoice.NumeroControl, // DTE-04-M038P019-...
        CodigoGeneracion: invoice.ID,
        TipoModelo:       1,
        TipoOperacion:    1,
        TipoContingencia: nil,
        MotivoContin:     nil,
        FecEmi:           invoice.IssueDate.Format("2006-01-02"),
        HorEmi:           invoice.IssueDate.Format("15:04:05"),
        TipoMoneda:       "USD",
    }
}

func (b *Builder) buildRemisionCuerpoDocumento(
    invoice *models.Invoice,
) ([]CuerpoDocumentoItem, []ItemAmounts) {
    items := make([]CuerpoDocumentoItem, len(invoice.LineItems))
    
    for i, lineItem := range invoice.LineItems {
        items[i] = CuerpoDocumentoItem{
            NumItem:      lineItem.LineNumber,
            TipoItem:     1, // Goods
            Cantidad:     lineItem.Quantity,
            Codigo:       &lineItem.ItemSku,
            UniMedida:    b.parseUniMedida(lineItem.UnitOfMeasure),
            Descripcion:  lineItem.ItemName,
            PrecioUni:    0, // Remision: no sale price
            MontoDescu:   0,
            VentaNoSuj:   0, // Not a sale
            VentaExenta:  0,
            VentaGravada: 0,
            Tributos:     nil, // No tributos
            NoGravado:    0,
        }
    }
    
    return items, nil
}

func (b *Builder) buildRemisionResumen(
    invoice *models.Invoice,
) ResumenRemision {
    return ResumenRemision{
        TotalNoSuj:          0,
        TotalExenta:         0,
        TotalGravada:        0,
        SubTotalVentas:      0,
        DescuNoSuj:          0,
        DescuExenta:         0,
        DescuGravada:        0,
        PorcentajeDescuento: nil,
        TotalDescu:          0,
        Tributos:            nil,
        SubTotal:            0,
        MontoTotalOperacion: 0,
        TotalLetras:         "CERO DÓLARES",
    }
}

func (b *Builder) buildRemisionExtension(
    invoice *models.Invoice,
) *ExtensionRemision {
    if invoice.DeliveryPerson == nil && invoice.VehiclePlate == nil && invoice.DeliveryNotes == nil {
        return nil // No extension needed
    }
    
    return &ExtensionRemision{
        NombEntrega:   invoice.DeliveryPerson,
        PlacaVehiculo: invoice.VehiclePlate,
        Observaciones: invoice.DeliveryNotes,
    }
}

func (b *Builder) buildDocumentosRelacionados(
    docs []models.RelatedDocument,
) []DocumentoRelacionado {
    result := make([]DocumentoRelacionado, len(docs))
    
    for i, doc := range docs {
        result[i] = DocumentoRelacionado{
            TipoDocumento:   doc.RelatedDocumentType,
            TipoGeneracion:  doc.RelatedGenerationType,
            NumeroDocumento: doc.RelatedDocumentNumber,
            FechaEmision:    doc.RelatedEmissionDate.Format("2006-01-02"),
        }
    }
    
    return result
}
```

---

### Phase 2: API Endpoints (Week 1)

#### 2.1 Create Remision
```go
// POST /v1/remisiones
type CreateRemisionRequest struct {
    CompanyID        string                  `json:"company_id"`
    EstablishmentID  string                  `json:"establishment_id"`
    PointOfSaleID    string                  `json:"point_of_sale_id"`
    RemisionType     string                  `json:"remision_type"` // "pre_invoice_delivery", etc
    ReceptorID       *string                 `json:"receptor_id"` // Null for internal
    LineItems        []LineItemRequest       `json:"line_items"`
    DeliveryPerson   *string                 `json:"delivery_person,omitempty"`
    VehiclePlate     *string                 `json:"vehicle_plate,omitempty"`
    DeliveryNotes    *string                 `json:"delivery_notes,omitempty"`
    RelatedDocuments []RelatedDocumentInput  `json:"related_documents,omitempty"`
}

// Response includes invoice ID and status
```

#### 2.2 Finalize & Submit
```go
// POST /v1/remisiones/{id}/finalize
// Same flow as invoices:
// 1. Build DTE
// 2. Sign
// 3. Submit to Hacienda
// 4. Store sello
```

#### 2.3 Link Invoice to Remision
```go
// POST /v1/invoices/{invoiceId}/link-remision
type LinkRemisionRequest struct {
    RemisionID string `json:"remision_id"`
}

// Creates entry in remision_invoice_links table
// Updates invoice's documentoRelacionado when finalizing
```

---

### Phase 3: Use Case-Specific Features (Week 2)

#### 3.1 Pre-Invoice Delivery Workflow
```
1. Create remision with known receptor
2. Submit to Hacienda → Get sello
3. Print/email remision with QR
4. Driver delivers goods with remision
5. Later: Create invoice referencing remision
```

**API Flow:**
```bash
# Step 1: Create remision
POST /v1/remisiones
{
  "remision_type": "pre_invoice_delivery",
  "receptor_id": "customer-uuid",
  "line_items": [...],
  "delivery_person": "Juan Pérez",
  "vehicle_plate": "P-123456"
}

# Step 2: Finalize
POST /v1/remisiones/{id}/finalize

# Step 3: Create invoice later
POST /v1/invoices
{
  "related_documents": [
    {
      "type": "05",
      "generation_type": 2,
      "number": "remision-uuid",
      "emission_date": "2025-11-10"
    }
  ]
}
```

#### 3.2 Inter-Branch Transfer
```
1. Create remision with receptor = other branch
2. Set observaciones = "Traslado entre sucursales"
3. Submit → Get sello
4. Update inventory (deduct from origin, add to destination)
```

**Special handling:**
- Receptor is another establishment of SAME company
- Need to handle this in client selection UI
- Inventory impact on both locations

#### 3.3 Route Sales (Tortillería)
```
1. Morning: Create remision with ALL product (confirmed + extra)
2. Set receptor = "CONSUMIDOR FINAL" or null
3. Submit → Driver gets QR
4. During route: Record actual sales
5. Evening: Create invoices for actual sales
6. Each invoice references morning remision
7. Update inventory for unsold items
```

**Complex workflow:**
- One remision → Multiple invoices
- Need UI to track "remision-derived invoices"
- Inventory reconciliation at end of day

---

## 🔐 Security & Validation

### Business Rules
```go
func ValidateRemision(remision *Invoice) error {
    // 1. Document type must be remision
    if remision.DocumentType != DocumentTypeRemision {
        return errors.New("not a remision")
    }
    
    // 2. Amounts typically zero
    if remision.TotalAmount != 0 {
        return errors.New("remision should have zero amount")
    }
    
    // 3. Receptor can be null for internal transfers
    if remision.RemisionType == RemisionInterBranch && remision.ReceptorID != nil {
        // Verify receptor is same company
    }
    
    // 4. Related docs must be type 01 or 03 only
    for _, doc := range remision.RelatedDocuments {
        if doc.Type != "01" && doc.Type != "03" {
            return errors.New("remision can only reference invoices (01/03)")
        }
    }
    
    // 5. Max 50 related docs
    if len(remision.RelatedDocuments) > 50 {
        return errors.New("max 50 related documents")
    }
    
    return nil
}
```

---

## 📊 Inventory Integration

### Key Decision: How Remisiones Affect Inventory

**Option A: Deduct on Remision Creation**
```
Remision created → Inventory reduced → Physical goods leave
If remision cancelled → Inventory restored
```

**Option B: Deduct on Related Invoice**
```
Remision created → Inventory "reserved" (not deducted)
Invoice created → Inventory actually deducted
```

**Recommendation:** Option B
- Remision = intention to move
- Invoice = actual sale
- Cleaner for route sales scenario

---

## 🧪 Testing Strategy

### Test Case 1: Simple Pre-Invoice Delivery
```json
{
  "remision_type": "pre_invoice_delivery",
  "receptor_id": "known-customer",
  "line_items": [
    {
      "item_id": "product-1",
      "quantity": 100,
      "unit_price": 0  // Remision: no sale price
    }
  ],
  "delivery_person": "Juan Pérez",
  "vehicle_plate": "P-123456",
  "delivery_notes": "Entrega previa a facturación"
}
```

**Expected:**
- ✅ DTE created with tipoDte="04"
- ✅ All amounts = $0.00
- ✅ Hacienda accepts with PROCESADO
- ✅ Can create invoice later referencing this remision

### Test Case 2: Internal Transfer (No Receptor)
```json
{
  "remision_type": "inter_branch_transfer",
  "receptor_id": null,  // No external receptor
  "line_items": [...],
  "delivery_notes": "Traslado de San Miguel a Santa Ana"
}
```

**Expected:**
- ✅ Receptor field = null in JSON
- ✅ Extension has observaciones
- ✅ Hacienda accepts

### Test Case 3: Route Sales
```json
{
  "remision_type": "route_sales",
  "receptor_id": null,  // Unknown at creation
  "line_items": [
    {"quantity": 500},  // Confirmed orders
    {"quantity": 100}   // Extra inventory
  ],
  "delivery_person": "Juan Pérez",
  "vehicle_plate": "P-789"
}
```

**Expected:**
- ✅ One remision covers all goods
- ✅ Multiple invoices can reference it
- ✅ Each invoice has documentoRelacionado pointing to remision

---

## 📱 UI Considerations

### Create Remision Screen
```
[Tipo de Remisión]
○ Entrega previa a facturación
○ Traslado entre sucursales
○ Ventas en ruta
○ Otro

[Receptor] (Optional for internal transfers)
[ Seleccionar cliente ▼ ]

[Artículos]
+ Agregar artículo

[Información de entrega]
Persona que entrega: [ _________ ]
Placa del vehículo:  [ _________ ]
Observaciones:       [ _________ ]

[Documentos relacionados] (Optional)
+ Agregar documento relacionado

[Crear Remisión]
```

### List View Enhancement
```
Documentos
Filter: [Todos ▼] [Facturas] [CCF] [Exportación] [Remisiones ✓] [Notas]

# INV-2025-001   Factura          $1,250.00  PROCESADO
# REM-2025-001   Remisión         $0.00      PROCESADO  🚚
# INV-2025-002   Factura          $890.00    PROCESADO
# REM-2025-002   Remisión         $0.00      PROCESADO  🚚
```

---

## ✅ Production Readiness Checklist

### Week 1 - MVP
- [ ] Database migration
- [ ] Models updated
- [ ] Calculator functions
- [ ] DTE builder
- [ ] Schema validation
- [ ] API endpoints
- [ ] Basic UI
- [ ] Test with Hacienda (get PROCESADO)

### Week 2 - Full Features
- [ ] Related documents support
- [ ] Invoice linking workflow
- [ ] Inventory integration
- [ ] All 3 use cases working
- [ ] Comprehensive tests
- [ ] Error handling
- [ ] Documentation

### Week 3 - Polish
- [ ] User training materials
- [ ] Admin dashboard
- [ ] Reporting (remisiones by type, etc)
- [ ] Performance optimization
- [ ] Production deployment
- [ ] Client onboarding

---

## 🚀 Next Steps

1. **Confirm approach** with team
2. **Create database migration** (extend invoices table)
3. **Implement calculator** (simple: all zeros)
4. **Build DTE structure** for Type 04
5. **Test submission** to Hacienda test environment
6. **Iterate** based on first PROCESADO response

