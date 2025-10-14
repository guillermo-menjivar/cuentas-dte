🎯 ACTION ITEMS - DTE Implementation Tasks

📚 Phase 1: Documentation & Setup
✅ Task 1.1: Document Consumidor Final Formula
Priority: HIGH
File: docs/dte-calculations.md
Create documentation explaining the two types of invoices:
A. Consumidor Final (B2C - DUI recipient):
javascript// Customer pays total WITH IVA included
const totalWithIVA = 11.30;

// Item level
precioUni = totalWithIVA;              // 11.30 (includes IVA)
ventaGravada = precioUni × cantidad - montoDescu;  // 11.30
ivaItem = ventaGravada - (ventaGravada / 1.13);    // 1.30

// Resumen level
totalGravada = sum(ventaGravada);      // 11.30 (includes IVA)
totalIva = sum(ivaItem);               // 1.30
subTotal = totalGravada;               // 11.30 (NOT totalGravada + totalIva!)
totalPagar = subTotal;                 // 11.30
B. Crédito Fiscal (B2B - NIT recipient):
javascript// Base price WITHOUT IVA
const basePrice = 10.00;

// Item level
precioUni = basePrice;                 // 10.00 (without IVA)
ventaGravada = precioUni × cantidad - montoDescu;  // 10.00
ivaItem = ventaGravada × 0.13;         // 1.30

// Resumen level
totalGravada = sum(ventaGravada);      // 10.00
totalIva = sum(ivaItem);               // 1.30
subTotal = totalGravada + totalIva;    // 11.30
totalPagar = subTotal;                 // 11.30

✅ Task 1.2: Document Rounding Rules
Priority: HIGH
File: docs/dte-calculations.md
markdown## Rounding Rules

### Item Level (cuerpoDocumento)
- Can use up to 8 decimal places
- Round using banker's rounding (round half to even)
- When 9th decimal ≥ 5, round up the 8th decimal

### Resumen Level
- Must use exactly 2 decimal places
- Round using banker's rounding
- When 3rd decimal ≥ 5, round up the 2nd decimal

### Tolerance
- Maximum difference allowed: ±0.01 cent between calculated and provided values

✅ Task 1.3: Document numeroControl Format
Priority: HIGH
File: docs/dte-format.md
markdown## Numero de Control Format

Pattern: `DTE-{tipoDte}-M{codEstable}P{codPuntoVenta}-{sequence}`

Example: `DTE-01-M0001P0001-000000000000001`

Where:
- `tipoDte`: "01" for Factura
- `M{codEstable}`: "M" + 4-digit establishment code
- `P{codPuntoVenta}`: "P" + 4-digit point of sale code
- `sequence`: 15-digit sequential number (zero-padded)

Total length: 31 characters

🏗️ Phase 2: Go Code Implementation
✅ Task 2.1: Create DTE Calculator Package
Priority: HIGH
File: internal/dte/calculator.go
gopackage dte

import (
    "math"
)

type InvoiceType string

const (
    InvoiceTypeConsumidorFinal InvoiceType = "consumidor_final" // B2C
    InvoiceTypeCreditoFiscal   InvoiceType = "credito_fiscal"   // B2B
)

// ItemCalculation holds calculated values for a single item
type ItemCalculation struct {
    PrecioUni     float64
    VentaGravada  float64
    IvaItem       float64
}

// CalculateItem calculates item-level values based on invoice type
func CalculateItem(totalPrice float64, quantity float64, discount float64, invoiceType InvoiceType) ItemCalculation {
    switch invoiceType {
    case InvoiceTypeConsumidorFinal:
        return calculateConsumidorFinalItem(totalPrice, quantity, discount)
    case InvoiceTypeCreditoFiscal:
        return calculateCreditoFiscalItem(totalPrice, quantity, discount)
    default:
        return ItemCalculation{}
    }
}

// calculateConsumidorFinalItem - price includes IVA
func calculateConsumidorFinalItem(totalPrice, quantity, discount float64) ItemCalculation {
    precioUni := totalPrice
    ventaGravada := precioUni*quantity - discount
    ivaItem := ventaGravada - (ventaGravada / 1.13)
    
    return ItemCalculation{
        PrecioUni:    roundTo8Decimals(precioUni),
        VentaGravada: roundTo8Decimals(ventaGravada),
        IvaItem:      roundTo8Decimals(ivaItem),
    }
}

// calculateCreditoFiscalItem - price excludes IVA
func calculateCreditoFiscalItem(basePrice, quantity, discount float64) ItemCalculation {
    precioUni := basePrice
    ventaGravada := precioUni*quantity - discount
    ivaItem := ventaGravada * 0.13
    
    return ItemCalculation{
        PrecioUni:    roundTo8Decimals(precioUni),
        VentaGravada: roundTo8Decimals(ventaGravada),
        IvaItem:      roundTo8Decimals(ivaItem),
    }
}

// ResumenCalculation holds calculated values for resumen
type ResumenCalculation struct {
    TotalGravada        float64
    SubTotalVentas      float64
    TotalIva            float64
    SubTotal            float64
    MontoTotalOperacion float64
    TotalPagar          float64
}

// CalculateResumen calculates resumen-level totals
func CalculateResumen(items []ItemCalculation, invoiceType InvoiceType) ResumenCalculation {
    var totalGravada, totalIva float64
    
    for _, item := range items {
        totalGravada += item.VentaGravada
        totalIva += item.IvaItem
    }
    
    // Round to 2 decimals for resumen
    totalGravada = roundTo2Decimals(totalGravada)
    totalIva = roundTo2Decimals(totalIva)
    
    var subTotal float64
    if invoiceType == InvoiceTypeConsumidorFinal {
        // For consumidor final, subTotal = totalGravada (already includes IVA)
        subTotal = totalGravada
    } else {
        // For credito fiscal, subTotal = totalGravada + totalIva
        subTotal = roundTo2Decimals(totalGravada + totalIva)
    }
    
    return ResumenCalculation{
        TotalGravada:        totalGravada,
        SubTotalVentas:      totalGravada,
        TotalIva:            totalIva,
        SubTotal:            subTotal,
        MontoTotalOperacion: subTotal,
        TotalPagar:          subTotal,
    }
}

// roundTo8Decimals rounds to 8 decimal places using banker's rounding
func roundTo8Decimals(value float64) float64 {
    return math.Round(value*100000000) / 100000000
}

// roundTo2Decimals rounds to 2 decimal places using banker's rounding
func roundTo2Decimals(value float64) float64 {
    return math.Round(value*100) / 100
}
Test file: internal/dte/calculator_test.go

✅ Task 2.2: Create Numero Control Generator
Priority: HIGH
File: internal/dte/numero_control.go
gopackage dte

import "fmt"

// GenerateNumeroControl creates a properly formatted numero de control
func GenerateNumeroControl(tipoDte, codEstable, codPuntoVenta string, sequence int64) string {
    return fmt.Sprintf(
        "DTE-%s-M%sP%s-%015d",
        tipoDte,
        codEstable,      // Already 4 digits: "0001"
        codPuntoVenta,   // Already 4 digits: "0001"
        sequence,        // Zero-padded to 15 digits
    )
}

// Example:
// GenerateNumeroControl("01", "0001", "0001", 1)
// Returns: "DTE-01-M0001P0001-000000000000001"
Test file: internal/dte/numero_control_test.go

✅ Task 2.3: Create DTE Builder
Priority: HIGH
File: internal/dte/builder.go
gopackage dte

import (
    "time"
)

type DTEBuilder struct {
    emisor          Emisor
    receptor        Receptor
    items           []Item
    invoiceType     InvoiceType
    codEstable      string
    codPuntoVenta   string
    nextSequence    int64
}

func NewDTEBuilder() *DTEBuilder {
    return &DTEBuilder{
        items: make([]Item, 0),
    }
}

func (b *DTEBuilder) SetEmisor(emisor Emisor) *DTEBuilder {
    b.emisor = emisor
    return b
}

func (b *DTEBuilder) SetReceptor(receptor Receptor) *DTEBuilder {
    b.receptor = receptor
    
    // Determine invoice type based on receptor's document type
    if receptor.TipoDocumento == "36" { // NIT
        b.invoiceType = InvoiceTypeCreditoFiscal
    } else { // DUI or others
        b.invoiceType = InvoiceTypeConsumidorFinal
    }
    
    return b
}

func (b *DTEBuilder) AddItem(item Item) *DTEBuilder {
    b.items = append(b.items, item)
    return b
}

func (b *DTEBuilder) Build() (*DTE, error) {
    // Generate codigo de generacion (UUID v4)
    codigoGeneracion := generateUUID()
    
    // Generate numero de control
    numeroControl := GenerateNumeroControl(
        "01", // tipoDte (Factura)
        b.codEstable,
        b.codPuntoVenta,
        b.nextSequence,
    )
    
    // Calculate items
    var itemCalculations []ItemCalculation
    var cuerpoDocumento []CuerpoDocumentoItem
    
    for i, item := range b.items {
        calc := CalculateItem(
            item.TotalPrice,
            item.Quantity,
            item.Discount,
            b.invoiceType,
        )
        itemCalculations = append(itemCalculations, calc)
        
        cuerpoDocumento = append(cuerpoDocumento, CuerpoDocumentoItem{
            NumItem:       i + 1,
            TipoItem:      item.TipoItem,
            Cantidad:      calc.PrecioUni, // TODO: Fix this
            Codigo:        item.Codigo,
            Descripcion:   item.Descripcion,
            PrecioUni:     calc.PrecioUni,
            MontoDescu:    item.Discount,
            VentaGravada:  calc.VentaGravada,
            IvaItem:       calc.IvaItem,
            // ... other fields
        })
    }
    
    // Calculate resumen
    resumen := CalculateResumen(itemCalculations, b.invoiceType)
    
    // Build DTE
    now := time.Now()
    dte := &DTE{
        Identificacion: Identificacion{
            Version:          1,
            Ambiente:         "00", // Test
            TipoDte:          "01",
            NumeroControl:    numeroControl,
            CodigoGeneracion: codigoGeneracion,
            TipoModelo:       1,
            TipoOperacion:    1,
            FecEmi:           now.Format("2006-01-02"),
            HorEmi:           now.Format("15:04:05"),
            TipoMoneda:       "USD",
        },
        Emisor:          b.emisor,
        Receptor:        &b.receptor,
        CuerpoDocumento: cuerpoDocumento,
        Resumen: Resumen{
            TotalGravada:        resumen.TotalGravada,
            SubTotalVentas:      resumen.SubTotalVentas,
            TotalIva:            resumen.TotalIva,
            SubTotal:            resumen.SubTotal,
            MontoTotalOperacion: resumen.MontoTotalOperacion,
            TotalPagar:          resumen.TotalPagar,
            TotalLetras:         numberToWords(resumen.TotalPagar),
            CondicionOperacion:  1,
            // ... other fields
        },
    }
    
    return dte, nil
}

✅ Task 2.4: Implement UUID Generator
Priority: MEDIUM
File: internal/dte/uuid.go
gopackage dte

import (
    "crypto/rand"
    "fmt"
    "strings"
)

// generateUUID creates a UUID v4 with uppercase hex digits (A-F, 0-9)
func generateUUID() string {
    b := make([]byte, 16)
    rand.Read(b)
    
    // Set version (4) and variant bits
    b[6] = (b[6] & 0x0f) | 0x40
    b[8] = (b[8] & 0x3f) | 0x80
    
    uuid := fmt.Sprintf("%X-%X-%X-%X-%X",
        b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
    
    return strings.ToUpper(uuid)
}

✅ Task 2.5: Implement Number to Words
Priority: LOW
File: internal/dte/number_to_words.go
gopackage dte

import "fmt"

// numberToWords converts a float to Spanish words
// Example: 11.30 -> "ONCE DOLARES CON TREINTA CENTAVOS"
func numberToWords(amount float64) string {
    // TODO: Implement proper Spanish number-to-words conversion
    // For now, return a placeholder
    dollars := int(amount)
    cents := int((amount - float64(dollars)) * 100)
    
    return fmt.Sprintf("%d DOLARES CON %d CENTAVOS", dollars, cents)
}

🔌 Phase 3: API Integration
✅ Task 3.1: Create Firmador Client
Priority: HIGH
File: internal/firmador/client.go
gopackage firmador

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

const firmadorURL = "http://167.172.230.154:8113/firmardocumento/"

type Client struct {
    httpClient *http.Client
}

func NewClient() *Client {
    return &Client{
        httpClient: &http.Client{},
    }
}

type SignRequest struct {
    NIT         string      `json:"nit"`
    Activo      bool        `json:"activo"`
    PasswordPri string      `json:"passwordPri"`
    DTEJson     interface{} `json:"dteJson"`
}

type SignResponse struct {
    Status string `json:"status"`
    Body   string `json:"body"` // JWT token
}

func (c *Client) SignDTE(nit, password string, dte interface{}) (string, error) {
    req := SignRequest{
        NIT:         nit,
        Activo:      true,
        PasswordPri: password,
        DTEJson:     dte,
    }
    
    jsonData, err := json.Marshal(req)
    if err != nil {
        return "", fmt.Errorf("marshal request: %w", err)
    }
    
    resp, err := c.httpClient.Post(firmadorURL, "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        return "", fmt.Errorf("post request: %w", err)
    }
    defer resp.Body.Close()
    
    var signResp SignResponse
    if err := json.NewDecoder(resp.Body).Decode(&signResp); err != nil {
        return "", fmt.Errorf("decode response: %w", err)
    }
    
    if signResp.Body == "" {
        return "", fmt.Errorf("empty JWT in response")
    }
    
    return signResp.Body, nil
}

✅ Task 3.2: Create Hacienda Client
Priority: HIGH
File: internal/hacienda/client.go
gopackage hacienda

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

const (
    testURL = "https://apitest.dtes.mh.gob.sv/fesv/recepciondte"
    prodURL = "https://api.dtes.mh.gob.sv/fesv/recepciondte"
)

type Client struct {
    httpClient *http.Client
    baseURL    string
    token      string
}

func NewClient(isTest bool, token string) *Client {
    url := prodURL
    if isTest {
        url = testURL
    }
    
    return &Client{
        httpClient: &http.Client{},
        baseURL:    url,
        token:      token,
    }
}

type SubmitRequest struct {
    Ambiente        string `json:"ambiente"`
    IDEnvio         int    `json:"idEnvio"`
    Version         int    `json:"version"`
    TipoDTE         string `json:"tipoDte"`
    Documento       string `json:"documento"` // Signed JWT
    CodigoGeneracion string `json:"codigoGeneracion,omitempty"`
}

type SubmitResponse struct {
    Version         int      `json:"version"`
    Ambiente        string   `json:"ambiente"`
    VersionApp      int      `json:"versionApp"`
    Estado          string   `json:"estado"` // "PROCESADO" or "RECHAZADO"
    CodigoGeneracion string  `json:"codigoGeneracion"`
    SelloRecibido   *string  `json:"selloRecibido"`
    FhProcesamiento string   `json:"fhProcesamiento"`
    ClasificaMsg    string   `json:"clasificaMsg"`
    CodigoMsg       string   `json:"codigoMsg"`
    DescripcionMsg  string   `json:"descripcionMsg"`
    Observaciones   []string `json:"observaciones"`
}

func (c *Client) SubmitDTE(signedJWT, codigoGeneracion string) (*SubmitResponse, error) {
    req := SubmitRequest{
        Ambiente:         "00", // Test
        IDEnvio:          1,
        Version:          2,
        TipoDTE:          "01",
        Documento:        signedJWT,
        CodigoGeneracion: codigoGeneracion,
    }
    
    jsonData, err := json.Marshal(req)
    if err != nil {
        return nil, fmt.Errorf("marshal request: %w", err)
    }
    
    httpReq, err := http.NewRequest("POST", c.baseURL, bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }
    
    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("Authorization", c.token)
    
    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("post request: %w", err)
    }
    defer resp.Body.Close()
    
    var submitResp SubmitResponse
    if err := json.NewDecoder(resp.Body).Decode(&submitResp); err != nil {
        return nil, fmt.Errorf("decode response: %w", err)
    }
    
    return &submitResp, nil
}

✅ Task 3.3: Create DTE Service
Priority: HIGH
File: internal/service/dte_service.go
gopackage service

import (
    "fmt"
    "your-project/internal/dte"
    "your-project/internal/firmador"
    "your-project/internal/hacienda"
)

type DTEService struct {
    firmadorClient *firmador.Client
    haciendaClient *hacienda.Client
    dteBuilder     *dte.DTEBuilder
}

func NewDTEService(haciendaToken string, isTest bool) *DTEService {
    return &DTEService{
        firmadorClient: firmador.NewClient(),
        haciendaClient: hacienda.NewClient(isTest, haciendaToken),
        dteBuilder:     dte.NewDTEBuilder(),
    }
}

func (s *DTEService) CreateAndSubmitInvoice(req CreateInvoiceRequest) (*hacienda.SubmitResponse, error) {
    // 1. Build DTE
    dteDoc, err := s.dteBuilder.
        SetEmisor(req.Emisor).
        SetReceptor(req.Receptor).
        // Add items...
        Build()
    if err != nil {
        return nil, fmt.Errorf("build DTE: %w", err)
    }
    
    // 2. Sign DTE
    signedJWT, err := s.firmadorClient.SignDTE(
        req.Emisor.NIT,
        req.FirmadorPassword,
        dteDoc,
    )
    if err != nil {
        return nil, fmt.Errorf("sign DTE: %w", err)
    }
    
    // 3. Submit to Hacienda
    response, err := s.haciendaClient.SubmitDTE(
        signedJWT,
        dteDoc.Identificacion.CodigoGeneracion,
    )
    if err != nil {
        return nil, fmt.Errorf("submit DTE: %w", err)
    }
    
    return response, nil
}

🧪 Phase 4: Testing
✅ Task 4.1: Unit Tests for Calculator
Priority: HIGH
File: internal/dte/calculator_test.go
Test cases:

✅ Consumidor Final with $11.30 total
✅ Consumidor Final with $10.00 total (rounding)
✅ Consumidor Final with $79.90 total (real example)
✅ Crédito Fiscal with $10.00 base
✅ Multiple items
✅ Items with discounts


✅ Task 4.2: Integration Tests
Priority: MEDIUM
File: test/integration_test.go
Test full flow:

Build DTE
Sign with Firmador
Submit to Hacienda (test environment)
Verify response is "PROCESADO"


✅ Task 4.3: Create Test Helper Scripts
Priority: MEDIUM
Folder: test/
Keep existing scripts:

✅ validate_dte_schema.sh - Validate DTE against JSON schema
✅ test_hacienda_manually.sh - Manual end-to-end test
✅ decode_jwt.sh - Decode signed JWT for debugging


🗄️ Phase 5: Database & Sequence Management
✅ Task 5.1: Create DTE Sequences Table
Priority: HIGH
File: migrations/001_create_dte_sequences.sql
sqlCREATE TABLE dte_sequences (
    id SERIAL PRIMARY KEY,
    emisor_nit VARCHAR(14) NOT NULL,
    tipo_dte VARCHAR(2) NOT NULL,
    cod_estable VARCHAR(4) NOT NULL,
    cod_punto_venta VARCHAR(4) NOT NULL,
    next_sequence BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(emisor_nit, tipo_dte, cod_estable, cod_punto_venta)
);

CREATE INDEX idx_dte_sequences_lookup ON dte_sequences(
    emisor_nit, tipo_dte, cod_estable, cod_punto_venta
);

✅ Task 5.2: Create DTEs Table
Priority: HIGH
File: migrations/002_create_dtes.sql
sqlCREATE TABLE dtes (
    id SERIAL PRIMARY KEY,
    emisor_nit VARCHAR(14) NOT NULL,
    receptor_documento VARCHAR(20),
    tipo_dte VARCHAR(2) NOT NULL,
    numero_control VARCHAR(31) NOT NULL UNIQUE,
    codigo_generacion UUID NOT NULL UNIQUE,
    estado VARCHAR(20) NOT NULL, -- PROCESADO, RECHAZADO, PENDIENTE
    sello_recibido VARCHAR(100),
    fecha_emision DATE NOT NULL,
    hora_emision TIME NOT NULL,
    fecha_procesamiento TIMESTAMP,
    monto_total DECIMAL(12,2) NOT NULL,
    dte_json JSONB NOT NULL,
    signed_jwt TEXT,
    hacienda_response JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_dtes_emisor ON dtes(emisor_nit);
CREATE INDEX idx_dtes_receptor ON dtes(receptor_documento);
CREATE INDEX idx_dtes_estado ON dtes(estado);
CREATE INDEX idx_dtes_fecha ON dtes(fecha_emision);
CREATE INDEX idx_dtes_codigo_gen ON dtes(codigo_generacion);

✅ Task 5.3: Implement Sequence Service
Priority: HIGH
File: internal/service/sequence_service.go
gopackage service

import (
    "database/sql"
    "fmt"
)

type SequenceService struct {
    db *sql.DB
}

func NewSequenceService(db *sql.DB) *SequenceService {
    return &SequenceService{db: db}
}

func (s *SequenceService) GetNextSequence(
    emisorNIT, tipoDTE, codEstable, codPuntoVenta string,
) (int64, error) {
    tx, err := s.db.Begin()
    if err != nil {
        return 0, err
    }
    defer tx.Rollback()
    
    var nextSeq int64
    err = tx.QueryRow(`
        INSERT INTO dte_sequences (
            emisor_nit, tipo_dte, cod_estable, cod_punto_venta, next_sequence
        ) VALUES ($1, $2, $3, $4, 1)
        ON CONFLICT (emisor_nit, tipo_dte, cod_estable, cod_punto_venta)
        DO UPDATE SET 
            next_sequence = dte_sequences.next_sequence + 1,
            updated_at = NOW()
        RETURNING next_sequence
    `, emisorNIT, tipoDTE, codEstable, codPuntoVenta).Scan(&nextSeq)
    
    if err != nil {
        return 0, err
    }
    
    if err := tx.Commit(); err != nil {
        return 0, err
    }
    
    return nextSeq, nil
}

📡 Phase 6: API Endpoints
✅ Task 6.1: Create Invoice Endpoint
Priority: HIGH
File: internal/handler/invoice_handler.go
gopackage handler

import (
    "encoding/json"
    "net/http"
)

type InvoiceHandler struct {
    dteService *service.DTEService
}

// POST /api/v1/invoices
func (h *InvoiceHandler) CreateInvoice(w http.ResponseWriter, r *http.Request) {
    var req CreateInvoiceRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Validate request
    if err := req.Validate(); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Create and submit invoice
    response, err := h.dteService.CreateAndSubmitInvoice(req)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // Return response
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

✅ Task 6.2: Create Get Invoice Endpoint
Priority: MEDIUM
File: internal/handler/invoice_handler.go
go// GET /api/v1/invoices/:codigo_generacion
func (h *InvoiceHandler) GetInvoice(w http.ResponseWriter, r *http.Request) {
    // TODO: Implement
}

// GET /api/v1/invoices
func (h *InvoiceHandler) ListInvoices(w http.ResponseWriter, r *http.Request) {
    // TODO: Implement with pagination
}

🔒 Phase 7: Security & Configuration
✅ Task 7.1: Environment Configuration
Priority: HIGH
File: .env.example
bash# Hacienda API
HACIENDA_ENVIRONMENT=test  # test or production
HACIENDA_TOKEN=your_bearer_token_here

# Firmador
FIRMADOR_URL=http://167.172.230.154:8113/firmardocumento/

# Database
DATABASE_URL=postgresql://user:pass@localhost:5432/dbname

# Emisor Defaults
EMISOR_NIT=06143005061013
EMISOR_NRC=1726313
EMISOR_PASSWORD_PRI=your_firmador_password

# Establishment
COD_ESTABLE=M001
COD_PUNTO_VENTA=P001

✅ Task 7.2: Secrets Management
Priority: HIGH
File: internal/config/config.go
gopackage config

import (
    "os"
)

type Config struct {
    HaciendaToken    string
    HaciendaIsTest   bool
    FirmadorURL      string
    DatabaseURL      string
    EmisorNIT        string
    EmisorNRC        string
    EmisorPassword   string
    CodEstable       string
    CodPuntoVenta    string
}

func Load() (*Config, error) {
    return &Config{
        HaciendaToken:  os.Getenv("HACIENDA_TOKEN"),
        HaciendaIsTest: os.Getenv("HACIENDA_ENVIRONMENT") == "test",
        FirmadorURL:    os.Getenv("FIRMADOR_URL"),
        DatabaseURL:    os.Getenv("DATABASE_URL"),
        EmisorNIT:      os.Getenv("EMISOR_NIT"),
        EmisorNRC:      os.Getenv("EMISOR_NRC"),
        EmisorPassword: os.Getenv("EMISOR_PASSWORD_PRI"),
        CodEstable:     os.Getenv("COD_ESTABLE"),
        CodPuntoVenta:  os.Getenv("COD_PUNTO_VENTA"),
    }, nil
}

📊 Phase 8: Monitoring & Logging
✅ Task 8.1: Add Structured Logging
Priority: MEDIUM
File: Throughout codebase
Use structured logging for:

DTE creation attempts
Firmador signing requests/responses
Hacienda submission requests/responses
Errors and rejections
Success with sello recibido


✅ Task 8.2: Add Metrics
Priority: LOW
File: internal/metrics/metrics.go
Track:

DTEs created per minute
DTEs accepted vs rejected
Average response time from Hacienda
Errors by type


🚀 Phase 9: Deployment
✅ Task 9.1: Docker Configuration
Priority: MEDIUM
File: Dockerfile
dockerfileFROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]

✅ Task 9.2: AWS Infrastructure
Priority: MEDIUM
Using your existing MCP AWS integration:

Create DynamoDB table for DTEs (optional alternative to PostgreSQL)
Create S3 bucket for DTE backups
Set up proper IAM roles


📝 Phase 10: Future Enhancements
✅ Task 10.1: Support Other Document Types
Priority: LOW
Implement:

Crédito Fiscal (already designed)
Nota de Crédito (credit notes)
Nota de Débito (debit notes)
Factura de Exportación (export invoices)


✅ Task 10.2: PDF Generation
Priority: LOW
File: internal/pdf/generator.go
Generate printable PDF invoices from DTE JSON

✅ Task 10.3: Email Notifications
Priority: LOW
Send invoices to customers via email

✅ Task 10.4: Retry Logic
Priority: MEDIUM
Implement retry mechanism for:

Firmador connection failures
Hacienda connection failures
Network timeouts


✅ Task 10.5: Webhook Support
Priority: LOW
Allow external systems to receive notifications when DTEs are processed

🎯 Summary - Recommended Order

START HERE ⭐

Task 1.1, 1.2, 1.3 (Documentation)
Task 2.1 (Calculator)
Task 2.2 (Numero Control)


Core Implementation 🏗️

Task 2.3, 2.4 (DTE Builder)
Task 3.1, 3.2 (Clients)
Task 4.1 (Tests)


Database & API 🗄️

Task 5.1, 5.2, 5.3 (Database)
Task 6.1 (API Endpoints)
Task 7.1, 7.2 (Config)


Polish & Deploy 🚀

Task 4.2, 4.3 (Integration tests)
Task 8.1 (Logging)
Task 9.1 (Docker)



YOU'VE GOT THIS! 💪🎉
