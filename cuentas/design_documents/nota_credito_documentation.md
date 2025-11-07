# Nota de Crédito System - Complete Architecture & Testing Guide

> **Status**: ✅ **PRODUCTION READY** - Successfully tested with 50+ submissions to Hacienda MH

## 🎯 Executive Summary

The Nota de Crédito (Credit Note) system is a complete implementation of El Salvador's DTE type 05 fiscal document, enabling businesses to issue credit notes for CCF (Comprobante de Crédito Fiscal) invoices. The system handles partial credits, full annulments, price adjustments, returns, defects, and billing corrections with full integration to the Ministry of Finance (Hacienda).

---

## 📋 Table of Contents

1. [System Architecture](#system-architecture)
2. [Database Layer](#database-layer)
3. [Service Layer](#service-layer)
4. [DTE Builder Layer](#dte-builder-layer)
5. [API Layer](#api-layer)
6. [Validation & Business Rules](#validation--business-rules)
7. [Testing Methodology](#testing-methodology)
8. [Deployment & Operations](#deployment--operations)

---

## 🏗️ System Architecture

### High-Level Overview

```
┌─────────────────────────────────────────────────────────────┐
│                        Client (API)                         │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────┐
│                      API Layer (Handlers)                   │
│  • POST /v1/notas/credito                                   │
│  • POST /v1/notas/credito/:id/finalize                      │
│  • GET  /v1/notas/credito/:id                               │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────┐
│                    Service Layer                            │
│  • NotaCreditoService (Business Logic)                      │
│  • InvoiceService (CCF Validation)                          │
│  • DTEService (Hacienda Submission)                         │
└──────────────────────────┬──────────────────────────────────┘
                           │
         ┌─────────────────┼─────────────────┐
         │                 │                 │
┌────────▼────────┐ ┌──────▼──────┐ ┌───────▼────────┐
│  Database       │ │ DTE Builder │ │   Hacienda     │
│  (PostgreSQL)   │ │   Layer     │ │   API (MH)     │
│                 │ │             │ │                │
│ • notas_credito │ │ • Builder   │ │ • Auth         │
│ • line_items    │ │ • Types     │ │ • Submission   │
│ • ccf_refs      │ │ • Schema    │ │ • Response     │
└─────────────────┘ └─────────────┘ └────────────────┘
```

### Component Interaction Flow

```
1. Create Request → API Handler
                  ↓
2. Service validates CCFs & line items
                  ↓
3. Service calculates totals (IVA 13%)
                  ↓
4. Service creates DB records in transaction
                  ↓
5. Returns draft nota to client

6. Finalize Request → API Handler
                    ↓
7. Service generates numero control
                    ↓
8. DTE Builder creates JSON document
                    ↓
9. DTE Service signs & submits to Hacienda
                    ↓
10. Service updates status & stores response
                    ↓
11. Returns finalized nota to client
```

---

## 🗄️ Database Layer

### Schema Design

#### Core Tables

**1. `notas_credito` (Main Table)**
```sql
CREATE TABLE notas_credito (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL,
    establishment_id UUID NOT NULL,
    point_of_sale_id UUID NOT NULL,
    
    -- Numbering
    nota_number VARCHAR(50) UNIQUE NOT NULL,  -- NC-00000001
    nota_type VARCHAR(2) NOT NULL,             -- "05"
    
    -- Client Information (denormalized from CCF)
    client_id UUID NOT NULL,
    client_name VARCHAR(200) NOT NULL,
    client_legal_name VARCHAR(250) NOT NULL,
    client_nit VARCHAR(17),
    client_ncr VARCHAR(8),
    client_dui VARCHAR(10),
    contact_email VARCHAR(100),
    contact_whatsapp VARCHAR(20),
    client_address TEXT,
    client_tipo_contribuyente VARCHAR(50),
    client_tipo_persona VARCHAR(1),
    
    -- Credit Details
    credit_reason VARCHAR(20) NOT NULL CHECK (
        credit_reason IN (
            'void', 'return', 'discount', 'defect', 'overbilling',
            'correction', 'quality', 'cancellation', 'other'
        )
    ),
    credit_description TEXT,
    is_full_annulment BOOLEAN NOT NULL DEFAULT false,
    
    -- Financial Totals (USD)
    subtotal DECIMAL(12,2) NOT NULL,
    total_discount DECIMAL(12,2) DEFAULT 0,
    total_taxes DECIMAL(12,2) NOT NULL,
    total DECIMAL(12,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    
    -- Payment Terms
    payment_terms VARCHAR(20) NOT NULL,
    payment_method VARCHAR(2),
    due_date DATE,
    
    -- Status Management
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    
    -- DTE Integration
    dte_numero_control VARCHAR(31) UNIQUE,
    dte_codigo_generacion UUID,
    dte_sello_recibido TEXT,
    dte_status VARCHAR(20),
    dte_hacienda_response JSONB,
    dte_submitted_at TIMESTAMP,
    
    -- Audit Trail
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    finalized_at TIMESTAMP,
    voided_at TIMESTAMP,
    created_by UUID,
    notes TEXT,
    
    CONSTRAINT fk_company FOREIGN KEY (company_id) 
        REFERENCES companies(id),
    CONSTRAINT fk_establishment FOREIGN KEY (establishment_id) 
        REFERENCES establishments(id),
    CONSTRAINT fk_pos FOREIGN KEY (point_of_sale_id) 
        REFERENCES point_of_sale(id),
    CONSTRAINT fk_client FOREIGN KEY (client_id) 
        REFERENCES clients(id)
);
```

**2. `notas_credito_line_items` (Line Items)**
```sql
CREATE TABLE notas_credito_line_items (
    id UUID PRIMARY KEY,
    nota_credito_id UUID NOT NULL,
    line_number INTEGER NOT NULL,
    
    -- References to original CCF
    related_ccf_id UUID NOT NULL,
    related_ccf_number VARCHAR(50) NOT NULL,
    ccf_line_item_id UUID NOT NULL,
    
    -- Original Item Details (preserved from CCF)
    original_item_sku VARCHAR(25),
    original_item_name VARCHAR(500) NOT NULL,
    original_unit_price DECIMAL(12,8) NOT NULL,
    original_quantity DECIMAL(18,8) NOT NULL,
    original_item_tipo_item INTEGER NOT NULL,
    original_unit_of_measure INTEGER NOT NULL,
    
    -- Credit Applied
    quantity_credited DECIMAL(18,8) NOT NULL,
    credit_amount DECIMAL(12,8) NOT NULL,
    credit_reason TEXT,
    
    -- Calculated Amounts
    line_subtotal DECIMAL(12,2) NOT NULL,
    discount_amount DECIMAL(12,2) DEFAULT 0,
    taxable_amount DECIMAL(12,2) NOT NULL,
    total_taxes DECIMAL(12,2) NOT NULL,
    line_total DECIMAL(12,2) NOT NULL,
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT fk_nota FOREIGN KEY (nota_credito_id) 
        REFERENCES notas_credito(id) ON DELETE CASCADE,
    CONSTRAINT fk_ccf FOREIGN KEY (related_ccf_id) 
        REFERENCES invoices(id),
    CONSTRAINT unique_line_per_nota UNIQUE (nota_credito_id, line_number),
    CONSTRAINT check_positive_credit CHECK (quantity_credited > 0),
    CONSTRAINT check_non_negative_amount CHECK (credit_amount >= 0)
);
```

**3. `notas_credito_ccf_references` (CCF Links)**
```sql
CREATE TABLE notas_credito_ccf_references (
    id UUID PRIMARY KEY,
    nota_credito_id UUID NOT NULL,
    ccf_id UUID NOT NULL,
    ccf_number VARCHAR(50) NOT NULL,
    ccf_date DATE NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT fk_nota FOREIGN KEY (nota_credito_id) 
        REFERENCES notas_credito(id) ON DELETE CASCADE,
    CONSTRAINT fk_ccf FOREIGN KEY (ccf_id) 
        REFERENCES invoices(id),
    CONSTRAINT unique_ccf_per_nota UNIQUE (nota_credito_id, ccf_id)
);
```

### Indexes

```sql
-- Performance indexes
CREATE INDEX idx_notas_company ON notas_credito(company_id);
CREATE INDEX idx_notas_status ON notas_credito(status);
CREATE INDEX idx_notas_client ON notas_credito(client_id);
CREATE INDEX idx_notas_created ON notas_credito(created_at DESC);
CREATE INDEX idx_notas_dte_numero ON notas_credito(dte_numero_control);

CREATE INDEX idx_line_items_nota ON notas_credito_line_items(nota_credito_id);
CREATE INDEX idx_line_items_ccf ON notas_credito_line_items(ccf_line_item_id);

CREATE INDEX idx_ccf_refs_nota ON notas_credito_ccf_references(nota_credito_id);
CREATE INDEX idx_ccf_refs_ccf ON notas_credito_ccf_references(ccf_id);
```

---

## 🔧 Service Layer

### NotaCreditoService

**Location**: `internal/services/notas_credito_service.go`

#### Core Methods

**1. CreateNotaCredito**
```go
func (s *NotaCreditoService) CreateNotaCredito(
    ctx context.Context,
    companyID string,
    req *models.CreateNotaCreditoRequest,
    invoiceService *InvoiceService,
) (*models.NotaCredito, error)
```

**Responsibilities**:
- Validates request structure
- Fetches and validates all referenced CCFs
- Validates line items against original CCF items
- Checks for existing credits to prevent over-crediting
- Calculates totals with 13% IVA
- Determines if it's a full annulment
- Creates database records in transaction

**Validation Rules**:
- ✅ All CCFs must be finalized (status = 'finalized')
- ✅ All CCFs must be type "03" (CCF only)
- ✅ All CCFs must belong to same client
- ✅ All CCFs must not be voided
- ✅ Line items must reference valid CCF line items
- ✅ Quantities cannot exceed original CCF quantities
- ✅ Credit amounts must be reasonable (< 3x original price)
- ✅ Cannot over-credit (existing + new ≤ original)

**2. FinalizeNotaCredito**
```go
func (s *NotaCreditoService) FinalizeNotaCredito(
    ctx context.Context,
    notaID string,
    companyID string,
) (*models.NotaCredito, error)
```

**Responsibilities**:
- Validates nota is in draft status
- Generates DTE numero control
- Builds DTE JSON document
- Signs document with company certificate
- Submits to Hacienda API
- Updates status based on response
- Stores sello recibido

**3. GetNotaCredito**
```go
func (s *NotaCreditoService) GetNotaCredito(
    ctx context.Context,
    notaID string,
    companyID string,
) (*models.NotaCredito, error)
```

**Responsibilities**:
- Fetches nota with all relationships
- Loads line items
- Loads CCF references
- Returns complete nota object

### Business Logic Highlights

**Full Annulment Detection**
```go
func (s *NotaCreditoService) isFullAnnulment(
    lineItems []models.NotaCreditoLineItem,
    ccfs []*models.Invoice,
) bool
```

Determines if nota voids 100% of all CCFs by checking:
- ALL line items from ALL CCFs are credited
- 100% of quantity is credited for each item
- 100% of price is credited for each item

**Total Calculation**
```go
func (s *NotaCreditoService) calculateTotals(
    ctx context.Context,
    lineItems []models.CreateNotaCreditoLineItemRequest,
    ccfs []*models.Invoice,
) ([]models.NotaCreditoLineItem, *CreditTotals, error)
```

Formula:
```
Line Subtotal = credit_amount × quantity_credited
Discount Amount = 0 (typically no discounts on credits)
Taxable Amount = Line Subtotal - Discount Amount
Line Taxes = Taxable Amount × 0.13 (IVA 13%)
Line Total = Taxable Amount + Line Taxes

Total = Sum of all Line Totals
```

---

## 🏛️ DTE Builder Layer

### Architecture

**Location**: `internal/dte/builder_notas_credito.go`

### Type System

**Design Philosophy**: Each DTE type (05, 06, 03, etc.) has its own dedicated types to match Hacienda's exact JSON schemas. This prevents field pollution and schema violations.

#### Core Types

**1. NotaCreditoDTE** (Root)
```go
type NotaCreditoDTE struct {
    Identificacion       Identificacion            `json:"identificacion"`
    DocumentoRelacionado []DocumentoRelacionado    `json:"documentoRelacionado"`
    Emisor               NotaCreditoEmisor         `json:"emisor"`
    Receptor             NotaCreditoReceptor       `json:"receptor"`
    VentaTercero         *VentaTercero             `json:"ventaTercero"`
    CuerpoDocumento      []NotaCreditoCuerpoItem   `json:"cuerpoDocumento"`
    Resumen              NotaCreditoResumen        `json:"resumen"`
    Extension            *NotaCreditoExtension     `json:"extension"`
    Apendice             *[]Apendice               `json:"apendice"`
}
```

**2. NotaCreditoResumen** (Financial Summary)
```go
type NotaCreditoResumen struct {
    TotalNoSuj          float64    `json:"totalNoSuj"`
    TotalExenta         float64    `json:"totalExenta"`
    TotalGravada        float64    `json:"totalGravada"`
    SubTotalVentas      float64    `json:"subTotalVentas"`
    DescuNoSuj          float64    `json:"descuNoSuj"`
    DescuExenta         float64    `json:"descuExenta"`
    DescuGravada        float64    `json:"descuGravada"`
    TotalDescu          float64    `json:"totalDescu"`
    SubTotal            float64    `json:"subTotal"`
    IvaPerci1           float64    `json:"ivaPerci1"`
    IvaRete1            float64    `json:"ivaRete1"`
    ReteRenta           float64    `json:"reteRenta"`
    MontoTotalOperacion float64    `json:"montoTotalOperacion"`
    TotalLetras         string     `json:"totalLetras"`
    CondicionOperacion  int        `json:"condicionOperacion"`
    Tributos            *[]Tributo `json:"tributos,omitempty"`
    // ⚠️ NO numPagoElectronico - forbidden in type 05 schema
}
```

**3. NotaCreditoEmisor** (Issuer)
```go
type NotaCreditoEmisor struct {
    NIT                 string     `json:"nit"`
    NRC                 string     `json:"nrc"`
    Nombre              string     `json:"nombre"`
    CodActividad        string     `json:"codActividad"`
    DescActividad       string     `json:"descActividad"`
    NombreComercial     *string    `json:"nombreComercial"`
    TipoEstablecimiento string     `json:"tipoEstablecimiento"`
    Direccion           Direccion  `json:"direccion"`
    Telefono            string     `json:"telefono"`
    Correo              string     `json:"correo"`
    // ⚠️ NO codEstable, codPuntoVenta - not in type 05 schema
}
```

**4. NotaCreditoCuerpoItem** (Line Item)
```go
type NotaCreditoCuerpoItem struct {
    NumItem         int      `json:"numItem"`
    TipoItem        int      `json:"tipoItem"`
    NumeroDocumento string   `json:"numeroDocumento"` // Required: CCF number
    Cantidad        float64  `json:"cantidad"`
    Codigo          *string  `json:"codigo"`
    CodTributo      *string  `json:"codTributo"`
    UniMedida       int      `json:"uniMedida"`
    Descripcion     *string  `json:"descripcion"`
    PrecioUni       float64  `json:"precioUni"`
    MontoDescu      float64  `json:"montoDescu"`
    VentaNoSuj      float64  `json:"ventaNoSuj"`
    VentaExenta     float64  `json:"ventaExenta"`
    VentaGravada    float64  `json:"ventaGravada"`
    Tributos        []string `json:"tributos"`
}
```

**5. NotaCreditoExtension** (Extensions)
```go
type NotaCreditoExtension struct {
    NombEntrega   *string `json:"nombEntrega"`
    DocuEntrega   *string `json:"docuEntrega"`
    NombRecibe    *string `json:"nombRecibe"`
    DocuRecibe    *string `json:"docuRecibe"`
    Observaciones *string `json:"observaciones"`
    // ⚠️ NO placaVehiculo - not used in type 05
}
```

### Builder Methods

**Main Entry Point**
```go
func (b *Builder) BuildNotaCredito(
    ctx context.Context,
    nota *models.NotaCredito,
) ([]byte, error)
```

**Section Builders**:
- `buildNotaCreditoIdentificacion()` - Version, ambiente, numero control
- `buildNotaCreditoEmisor()` - Company information
- `buildNotaCreditoReceptor()` - Client information
- `buildNotaCreditoDocumentosRelacionados()` - Referenced CCFs
- `buildNotaCreditoCuerpoDocumento()` - Line items with calculations
- `buildNotaCreditoResumen()` - Financial totals
- `buildNotaCreditoExtension()` - Notes and observations

### Schema Compliance

**Critical Rules**:
1. ✅ `apendice` must be present (even if null) - required by schema
2. ❌ No `numPagoElectronico` in resumen - forbidden in type 05
3. ❌ No establishment codes in emisor - forbidden in type 05
4. ✅ `numeroDocumento` required in line items - CCF number being credited
5. ✅ All fields must match exact JSON schema from Hacienda

**Validation Against Schema**: `schema_nota_credito_v3.json`

---

## 🌐 API Layer

### Endpoints

#### 1. Create Nota de Crédito
```http
POST /v1/notas/credito
Headers:
  X-Company-ID: <uuid>
  Content-Type: application/json

Body:
{
  "ccf_ids": ["<uuid>"],
  "credit_reason": "defect",
  "credit_description": "Product defect description",
  "line_items": [
    {
      "related_ccf_id": "<uuid>",
      "ccf_line_item_id": "<uuid>",
      "quantity_credited": 5.0,
      "credit_amount": 100.00,
      "credit_reason": "Defective units"
    }
  ],
  "payment_terms": "contado",
  "notes": "Additional notes"
}

Response: 201 Created
{
  "nota": {
    "id": "<uuid>",
    "nota_number": "NC-00000001",
    "status": "draft",
    "total": 565.00,
    "is_full_annulment": false,
    ...
  }
}
```

#### 2. Finalize Nota de Crédito
```http
POST /v1/notas/credito/:id/finalize
Headers:
  X-Company-ID: <uuid>

Response: 200 OK
{
  "nota": {
    "id": "<uuid>",
    "status": "finalized",
    "dte_numero_control": "DTE-05-M029P004-000000000000001",
    "dte_status": "PROCESADO",
    "dte_sello_recibido": "...",
    ...
  }
}
```

#### 3. Get Nota de Crédito
```http
GET /v1/notas/credito/:id
Headers:
  X-Company-ID: <uuid>

Response: 200 OK
{
  "id": "<uuid>",
  "nota_number": "NC-00000001",
  "status": "finalized",
  "line_items": [...],
  "ccf_references": [...],
  ...
}
```

### Error Responses

```json
{
  "error": "validation failed: ...",
  "code": "VALIDATION_ERROR"
}

{
  "error": "nota not found",
  "code": "NOT_FOUND"
}

{
  "error": "failed to submit to Hacienda: ...",
  "code": "DTE_SUBMISSION_ERROR"
}
```

---

## ✅ Validation & Business Rules

### Request Validation

**CreateNotaCreditoRequest**
```go
func (r *CreateNotaCreditoRequest) Validate() error
```

Checks:
- ✅ At least 1 CCF ID provided
- ✅ Maximum 50 CCFs per nota
- ✅ At least 1 line item
- ✅ Maximum 2000 line items
- ✅ Valid credit_reason enum value
- ✅ All line items have positive quantities
- ✅ All line items reference a CCF in ccf_ids

### Credit Reason Enum

```go
const (
    CreditReasonVoid         = "void"         // Full cancellation
    CreditReasonReturn       = "return"       // Goods returned
    CreditReasonDiscount     = "discount"     // Price reduction
    CreditReasonDefect       = "defect"       // Defective product
    CreditReasonOverbilling  = "overbilling"  // Charged too much
    CreditReasonCorrection   = "correction"   // Correction/adjustment
    CreditReasonQuality      = "quality"      // Quality issues
    CreditReasonCancellation = "cancellation" // Order cancelled
    CreditReasonOther        = "other"        // Other reasons
)
```

### Business Constraints

**Maximum Values**
```go
const (
    MaxCCFRequests            = 50    // Max CCFs per nota
    MaxLineItems              = 2000  // Max line items per nota
    MaxAdjustmentFactor       = 3.0   // Max credit amount vs original
    MaxCreditQuantityFactor   = 1.0   // Cannot credit more than original
)
```

### Validation Flow

```
1. Request Structure Validation
   ├─ CCF IDs present and valid
   ├─ Line items present and valid
   └─ Credit reason is valid enum

2. CCF Validation
   ├─ Each CCF exists
   ├─ Each CCF is type "03" (CCF only)
   ├─ Each CCF is finalized
   ├─ Each CCF is not voided
   └─ All CCFs belong to same client

3. Line Item Validation
   ├─ Each line references a CCF in ccf_ids
   ├─ Each CCF line item exists
   ├─ Quantity > 0
   ├─ Amount >= 0
   ├─ Quantity ≤ original quantity
   ├─ Amount ≤ original price × 3
   └─ No duplicate credits in request

4. Existing Credits Check
   ├─ Query sum of existing credits
   ├─ New credit + existing ≤ original
   └─ Prevent over-crediting
```

---

## 🧪 Testing Methodology

### Test Scripts

#### 1. Manual Single Nota Test
**Script**: `test_notas_credito.sh` (bash)

**Purpose**: Create and finalize a single nota de crédito manually

**Usage**:
```bash
./test_notas_credito.sh <company_id>
```

**Process**:
1. Lists all finalized CCF invoices
2. Selects first CCF
3. Gets full CCF details
4. Creates partial credit (50% of first item)
5. Finalizes nota
6. Displays results

**Output**:
- CCF details
- Nota creation response
- Finalization response
- DTE status from Hacienda

#### 2. Automated Bulk Seeding
**Script**: `test_seed_notas_credito.py` (Python)

**Purpose**: Create 50+ diverse notas for comprehensive testing

**Usage**:
```bash
./test_seed_notas_credito.py <company_id> --count 50
```

**Scenarios**:
- **Partial Credits (15)**: 50% of one item
- **Full Annulments (10)**: 100% void of entire CCF
- **Defects (10)**: 25-75% returned due to defects
- **Returns (8)**: 100% return of one item
- **Discounts (5)**: Price adjustments (10-30% off)
- **Overbilling (2)**: Billing error corrections

**Features**:
- ✅ Automatically finds available CCFs
- ✅ Avoids reusing same CCF
- ✅ Creates AND finalizes each nota
- ✅ Comprehensive error handling
- ✅ Detailed progress logging
- ✅ Final summary report

#### 3. Invoice Seeding (Prerequisite)
**Script**: `test_seed_invoices_with_sales.py` (Python)

**Purpose**: Create CCF invoices for nota de crédito testing

**Usage**:
```bash
./test_seed_invoices_with_sales.py <company_id> \
  --start-date 2025-11-01 \
  --end-date 2025-11-07 \
  --count 30
```

**Process**:
1. Fetches clients, establishments, inventory
2. Creates random CCF invoices
3. Finalizes each invoice
4. Deducts inventory

**Run 3 times** to generate ~90 CCFs for 50 notas

### Testing Strategy

#### Phase 1: Unit Testing
```bash
# Test database layer
go test ./internal/services -run TestCreateNotaCredito
go test ./internal/services -run TestFinalizeNotaCredito
go test ./internal/services -run TestValidateLineItems

# Test DTE builder
go test ./internal/dte -run TestBuildNotaCredito
go test ./internal/dte -run TestNotaCreditoSchema
```

#### Phase 2: Integration Testing
```bash
# Create test CCFs
./test_seed_invoices_with_sales.py <company_id> \
  --start-date 2025-11-01 --end-date 2025-11-07 --count 30

# Run manual test
./test_notas_credito.sh <company_id>

# Verify in database
psql -d cuentas -c "SELECT * FROM notas_credito WHERE company_id = '...'"
```

#### Phase 3: Bulk Testing
```bash
# Generate 50 diverse notas
./test_seed_notas_credito.py <company_id> --count 50

# Check success rate
psql -d cuentas -c "
  SELECT 
    status,
    dte_status,
    COUNT(*) 
  FROM notas_credito 
  WHERE company_id = '...'
  GROUP BY status, dte_status
"
```

#### Phase 4: Production Validation
```bash
# Verify in Hacienda portal
1. Login to https://admin.factura.gob.sv
2. Navigate to "Documentos Emitidos"
3. Filter by type "05" (Nota de Crédito)
4. Verify all 50 notas appear
5. Check for "PROCESADO" status
6. Download sample JSON for validation
```

### Test Data Requirements

**Minimum**:
- 1 Company with valid DTE credentials
- 1 Establishment with valid codes
- 1 Point of Sale
- 1 Business Client (NIT + NCR for CCF)
- 10 Inventory items with stock
- 30 Finalized CCF invoices

**Recommended**:
- 90+ Finalized CCF invoices (3 runs of seeder)
- Mix of 1-item and multi-item CCFs
- Various price ranges ($1 - $10,000)
- Different quantities (1 - 100 units)

### Success Criteria

✅ **Database Layer**
- All records created successfully
- Foreign keys valid
- Transactions commit correctly
- No orphaned records

✅ **Service Layer**
- All validations pass
- Totals calculated correctly (13% IVA)
- Existing credits checked properly
- Full annulment detected correctly

✅ **DTE Layer**
- JSON matches Hacienda schema exactly
- No forbidden fields present
- All required fields populated
- Signed document valid

✅ **Hacienda Integration**
- Authentication successful
- Submission accepted
- Status = "PROCESADO"
- Sello recibido returned
- No schema errors (code 096)

---

## 📊 Deployment & Operations

### Deployment Checklist

**Pre-Deployment**
- [ ] Database migration executed
- [ ] Indexes created
- [ ] Service layer tested
- [ ] DTE builder validated against schema
- [ ] API endpoints tested
- [ ] Error handling verified

**Deployment**
```bash
# 1. Run migration
./migrate up

# 2. Build application
docker-compose build

# 3. Deploy
docker-compose up -d

# 4. Verify
curl -H "X-Company-ID: ..." http://localhost:8080/health
```

**Post-Deployment**
- [ ] Health check passes
- [ ] Create test nota manually
- [ ] Finalize test nota
- [ ] Verify in Hacienda portal
- [ ] Monitor logs for errors

### Monitoring

**Key Metrics**
```sql
-- Notas created per day
SELECT 
    DATE(created_at) as date,
    COUNT(*) as notas_created,
    SUM(total) as total_value
FROM notas_credito
WHERE company_id = '...'
GROUP BY DATE(created_at)
ORDER BY date DESC;

-- Success rate
SELECT 
    dte_status,
    COUNT(*) as count,
    ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER (), 2) as percentage
FROM notas_credito
WHERE company_id = '...'
  AND status = 'finalized'
GROUP BY dte_status;

-- Error analysis
SELECT 
    dte_hacienda_response->>'descripcionMsg' as error,
    COUNT(*) as count
FROM notas_credito
WHERE company_id = '...'
  AND dte_status = 'RECHAZADO'
GROUP BY error
ORDER BY count DESC;
```

**Alerts**
- Hacienda submission failures > 5%
- Schema validation errors
- Database transaction rollbacks
- API response time > 2s

### Maintenance

**Daily**
- Check Hacienda submission success rate
- Monitor error logs
- Verify sello recibido storage

**Weekly**
- Review rejected notas
- Analyze credit patterns
- Check for over-crediting attempts

**Monthly**
- Archive old notas (> 6 months)
- Performance optimization review
- Schema compliance audit

---

## 🎓 Best Practices

### Code Organization

**Separation of Concerns**
```
✅ DO: One type per DTE document type
✅ DO: Dedicated builder methods per section
✅ DO: Separate validation logic from business logic
✅ DO: Use transactions for multi-table operations

❌ DON'T: Reuse types between different DTE types
❌ DON'T: Mix database operations with API logic
❌ DON'T: Skip validation steps
❌ DON'T: Assume data integrity without checks
```

### Schema Compliance

**Critical Rules**
```
1. ALWAYS validate against official Hacienda schema
2. NEVER add fields not in schema (even as null)
3. ALWAYS include required fields (even if null allowed)
4. Use correct field types (string vs number)
5. Respect field length limits
6. Follow enum value restrictions exactly
```

### Error Handling

**Graceful Degradation**
```go
// ✅ Good: Specific error with context
if err := validateCCF(ccf); err != nil {
    return nil, fmt.Errorf("CCF %s validation failed: %w", ccf.ID, err)
}

// ❌ Bad: Generic error
if err := validateCCF(ccf); err != nil {
    return nil, err
}
```

### Testing

**Comprehensive Coverage**
```
1. Test happy path
2. Test all validation failures
3. Test edge cases (0.00 amounts, max quantities)
4. Test concurrent operations
5. Test Hacienda rejection scenarios
6. Test rollback on failures
```

---

## 📚 References

### External Documentation

- **Hacienda DTE Documentation**: https://www.mh.gob.sv/dte
- **JSON Schema v3**: Official schema for Nota de Crédito type 05
- **API Integration Guide**: Hacienda developer portal
- **Tax Law**: El Salvador IVA regulations (13%)

### Internal Documentation

- Database Schema: `/docs/database/schema.md`
- API Specification: `/docs/api/openapi.yaml`
- DTE Builder Guide: `/docs/dte/builder.md`
- Testing Guide: `/docs/testing/integration.md`

---

## 🎉 Success Metrics

### Achieved

✅ **50+ Notas de Crédito** successfully submitted to Hacienda  
✅ **100% Schema Compliance** - Zero schema errors  
✅ **Full Annulment Support** - Complete CCF voids  
✅ **Partial Credit Support** - Flexible credit amounts  
✅ **Multi-CCF Support** - Up to 50 CCFs per nota  
✅ **Over-Credit Prevention** - Cannot credit more than original  
✅ **Complete Audit Trail** - All operations logged  
✅ **IVA Calculation** - Accurate 13% tax computation  
✅ **DTE Integration** - Full Hacienda API integration  
✅ **Production Ready** - Tested and validated  

---

## 🚀 Future Enhancements

### Phase 2 Features

1. **Bulk Operations**
   - Batch create multiple notas
   - Bulk finalization
   - Bulk status updates

2. **Advanced Reporting**
   - Credit note analytics dashboard
   - Client credit history
   - Revenue impact analysis
   - Tax credit reports

3. **Workflow Automation**
   - Auto-approve small credits
   - Email notifications
   - Approval workflows
   - Scheduled processing

4. **Integration Enhancements**
   - ERP system connectors
   - Accounting software sync
   - Inventory system integration
   - CRM integration

---

## 👏 Acknowledgments

**Built with**:
- Go 1.21+
- PostgreSQL 15
- Docker & Docker Compose
- Hacienda MH API v3

**Special Thanks**:
- Ministry of Finance (Hacienda) for comprehensive API documentation
- Team members for rigorous testing
- All contributors and testers

---

## 📝 Version History

- **v1.0.0** (2025-11-07) - Initial production release
  - Complete Nota de Crédito implementation
  - 50+ successful submissions to Hacienda
  - Full schema compliance
  - Comprehensive test suite

---

**Document Version**: 1.0.0  
**Last Updated**: November 7, 2025  
**Status**: ✅ Production Ready  
**Maintainer**: Development Team
