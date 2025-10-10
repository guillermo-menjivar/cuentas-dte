DTE Service - Architecture & Implementation Plan
Document Version: 1.0
Date: 2025-01-XX
Status: Design Phase
Based on: Manual Funcional del Sistema de Transmisión V1.2 (Ministerio de Hacienda)

Table of Contents

Overview
System Context
Core Components
Data Model
State Machine
Integration Points
Business Rules
Implementation Phases
Error Handling & Contingency
Security Considerations
Open Questions


1. Overview
Purpose
The DTE (Documento Tributario Electrónico) service is responsible for transforming finalized invoices into legally compliant electronic tax documents for El Salvador's tax authority (Ministerio de Hacienda).
Scope
This service handles:

Generation of DTE JSON documents from finalized invoices
Digital signing through external firmador service
Submission to Ministerio de Hacienda (MH)
State tracking and audit logging
Contingency handling when MH is unavailable
Invalidation workflows

Legal Compliance
All implementations must comply with:

Normativa de Cumplimiento de los Documentos Tributarios Electrónicos
Manual Funcional del Sistema de Transmisión V1.2 (October 9, 2025)


2. System Context
┌─────────────┐
│   Invoice   │
│   Service   │
└──────┬──────┘
       │ 1. Finalize Invoice
       ▼
┌─────────────────────────────────────┐
│         DTE Service                 │
│  ┌───────────────────────────────┐ │
│  │  DTE Builder/Generator        │ │
│  └───────────┬───────────────────┘ │
│              │                       │
│  ┌───────────▼───────────────────┐ │
│  │  State Machine & Storage      │ │
│  └───────────┬───────────────────┘ │
└──────────────┼─────────────────────┘
               │
       ┌───────┴────────┐
       │                │
       ▼                ▼
┌─────────────┐  ┌─────────────┐
│  Firmador   │  │  Hacienda   │
│   Client    │  │   Client    │
└─────────────┘  └─────────────┘
       │                │
       ▼                ▼
┌─────────────┐  ┌─────────────┐
│  Firmador   │  │ Ministerio  │
│  Service    │  │    de       │
│  (External) │  │  Hacienda   │
└─────────────┘  └─────────────┘

3. Core Components
3.1 DTE Builder/Generator
Responsibility: Transform invoice data into compliant DTE JSON structure
Key Functions:

BuildFactura(invoice) - Generate Factura Electrónica (tipo 01)
BuildCCF(invoice) - Generate Comprobante de Crédito Fiscal (tipo 03)
GenerateCodigoGeneracion() - Create UUID v4 (uppercase, no dashes)
GenerateNumeroControl(establishment, pos, docType, year) - Create 31-char control number
CalculateTaxes(lineItems) - Apply IVA and specific taxes (tributos)
ApplyRounding(value, decimals) - Follow official rounding rules
ValidateHolgura(calculated, provided) - Verify tolerance rules

Input:

Invoice entity (with line items, taxes, client, payment info)
Configuration (establishment codes, ambiente, etc.)

Output:

Complete DTE JSON structure (unsigned)
Generated codigoGeneracion and numeroControl

Business Rules Applied:

8 decimal places for line item calculations
2 decimal places for totals (resumen section)
Holgura (tolerance) of ±$0.01 on summary fields
Numero control format: DTE-{tipo}-{estab}-{pos}-{sequential15}

Example: DTE-01-0001-0002-000000000000001


Sequential numbers reset annually
Tax codes from CAT-015 (Tributos catalog)


3.2 Firmador Client
Responsibility: Interface with external document signing service
Key Functions:

SignDocument(dteJSON) - Send unsigned DTE for signing
HandleSigningResponse(response) - Process signed document
RetryOnFailure(request) - Implement retry logic

Request Format:
json{
  "nit": "06140506661018",
  "activo": true,
  "passwordPri": "encrypted_password",
  "dteJson": { ... } // The complete DTE structure
}
Response Format:
json{
  "body": {
    "firmaElectronica": "eyJhbGciOiJSUzUxMiJ9...", // JWT signature
    ... // complete DTE with signature
  },
  "status": "OK"
}
Error Handling:

Network timeouts
Invalid JSON structure
Signing failures
Authentication errors


3.3 Hacienda Client
Responsibility: Submit signed DTEs to Ministerio de Hacienda
Key Functions:

SubmitDTE(signedDTE) - Send to MH reception endpoint
CheckStatus(codigoGeneracion) - Query DTE status
HandleMHResponse(response) - Process acceptance/rejection
DetectContingency() - Identify when MH is unavailable

Request Format:
json{
  "ambiente": "00", // 00=production, 01=test
  "idEnvio": 1,
  "version": 3,
  "tipoDte": "01",
  "documento": { ... }, // Signed DTE with firmaElectronica
  "codigoGeneracion": "341CA743-70F1-4CFE-88BC-7E4AE72E60CB"
}
Success Response:
json{
  "selloRecibido": "20227A1DE27CA974440985C1791AE37F7821LT8M",
  "estado": "PROCESADO",
  "codigoGeneracion": "341CA743-70F1-4CFE-88BC-7E4AE72E60CB"
}
Rejection Response:
json{
  "estado": "RECHAZADO",
  "descripcionMsg": "Error en campo X: descripción del error",
  "observaciones": ["campo Y no cumple validación"]
}
Timeout Policy:

Initial request: 5 second timeout
Retry policy: Apply exponential backoff (see manual section on "Política de reintentos")
After exhausting retries: Enter contingency mode


3.4 DTE State Machine
Responsibility: Track DTE lifecycle and orchestrate state transitions
States:
draft → generated → signed → submitted → accepted
                                  ↓
                              rejected
                                  ↓
                            (retry or fix)
State Definitions:
StateDescriptionCan Transition ToStored FieldsdraftInvoice finalized, DTE not yet generatedgeneratedinvoice_idgeneratedDTE JSON created with codigo generacionsigneddte_json, codigo_generacion, numero_controlsignedReturned from firmador with firma electronicasubmittedfirma_electronicasubmittedSent to Hacienda, awaiting responseaccepted, rejectedsubmitted_at, submission_idacceptedReceived sello recibido - officially a DTEinvalidatedsello_recibido, accepted_atrejectedMH rejected with errorsgenerated (after fixes)mh_errors, rejected_atinvalidatedInvalidation event processed(terminal)invalidation_event_idcontingencyGenerated offline, pending submissionsubmittedcontingency_event_id
State Transition Rules:

Only finalized invoices can generate DTEs
Cannot modify invoice once DTE is in submitted or later states
accepted DTEs can only be invalidated within legal timeframes:

Factura: 3 months from acceptance
CCF/NRE/NCE/NDE/CRE/CLE/DCLE/CDE: 1 day from acceptance


Invalidation requires generating replacement DTE first (except for "rescindir" type or NCE/CLE)


3.5 Sequence Manager
Responsibility: Generate sequential numero de control
Key Functions:

GetNextSequence(companyId, establishmentCode, posCode, dteType, year) - Thread-safe increment
ResetYearlySequences() - Reset all sequences on January 1st

Storage:
sqlCREATE TABLE dte_sequences (
  id UUID PRIMARY KEY,
  company_id UUID NOT NULL,
  establishment_code VARCHAR(4) NOT NULL, -- MH code
  pos_code VARCHAR(4) NOT NULL,           -- MH code
  dte_type VARCHAR(2) NOT NULL,           -- 01, 03, etc.
  year INTEGER NOT NULL,
  current_sequence BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMP,
  updated_at TIMESTAMP,
  UNIQUE(company_id, establishment_code, pos_code, dte_type, year)
);
Numero Control Format:
DTE-{tipo}-{establecimiento}-{puntoVenta}-{secuencial}
├─┬─┘ ├┬┘  └──────┬─────────┘ └─────┬────┘ └────┬────┘
│ │   ││          │                  │            │
│ │   ││          │                  │            └─ 15 digits (zero-padded)
│ │   ││          │                  └─ 4 chars (MH point of sale code)
│ │   ││          └─ 4 chars (MH establishment code)
│ │   │└─ 2 digits (document type: 01=Factura, 03=CCF, etc.)
│ │   └─ Separator (-)
│ └─ Separator (-)
└─ Fixed prefix "DTE"

Total: 31 characters

Example: DTE-01-0001-0002-000000000000001
Concurrency Handling:

Use database-level locking (SELECT ... FOR UPDATE)
Or use optimistic locking with version field
Ensure no gaps in sequences (important for audit)


3.6 Contingency Handler
Responsibility: Manage offline DTE generation when MH is unavailable
Contingency Detection:

MH endpoint returns 5xx errors
Network timeouts after retry exhaustion
Explicit MH maintenance windows (if announced)

Contingency Process:

During Contingency:

Generate DTE with tipoTransmision: 2 (contingency)
Generate DTE with tipoModelo: 2 (modelo diferido)
Store with status contingency
Can deliver to customer without selloRecibido


Contingency Event Creation:

Within 24 hours of contingency ending
Create EventoContingencia structure
List all DTE codigoGeneracion values (max 5000 per event)
Submit event to MH


DTE Submission:

Within 72 hours of event acceptance
Submit each contingency DTE individually
Update status to accepted when selloRecibido received



Contingency Event Structure:
json{
  "identificacion": {
    "version": 1,
    "ambiente": "00",
    "codigoGeneracion": "UUID-for-event",
    "fTransmision": "2025-01-15T10:30:00"
  },
  "emisor": { ... },
  "detalleDTE": [
    {
      "codigoGeneracion": "DTE-UUID-1",
      "tipoDte": "01"
    },
    // ... up to 5000 DTEs
  ],
  "motivo": {
    "fInicio": "2025-01-14T15:00:00",
    "fFin": "2025-01-15T10:00:00",
    "tipoContingencia": "1", // CAT-005
    "motivoContingencia": "Falla de internet del proveedor"
  }
}
Contingency Types (CAT-005):

1: Falla en conexiones del sistema del emisor
2: Falla suministro de Internet
3: Falla suministro energía eléctrica
4: No disponibilidad sistema del MH
5: Otro motivo


4. Data Model
4.1 Main DTE Table
sqlCREATE TABLE dtes (
  -- Identity
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID NOT NULL REFERENCES companies(id),
  invoice_id UUID NOT NULL REFERENCES invoices(id) UNIQUE,
  
  -- DTE Identification (from JSON estructura)
  codigo_generacion UUID NOT NULL UNIQUE,
  numero_control VARCHAR(31) NOT NULL UNIQUE,
  tipo_dte VARCHAR(2) NOT NULL, -- 01=Factura, 03=CCF, etc.
  version INTEGER NOT NULL DEFAULT 3,
  
  -- Configuration
  ambiente VARCHAR(2) NOT NULL, -- 00=production, 01=test
  tipo_modelo INTEGER NOT NULL DEFAULT 1, -- 1=previo, 2=diferido
  tipo_transmision INTEGER NOT NULL DEFAULT 1, -- 1=normal, 2=contingencia
  
  -- State Machine
  status VARCHAR(20) NOT NULL, -- draft, generated, signed, submitted, accepted, rejected, invalidated
  
  -- Document Storage
  dte_json JSONB NOT NULL, -- Unsigned DTE
  firma_electronica TEXT, -- JWT signature from firmador
  sello_recibido VARCHAR(255), -- From MH
  
  -- MH Communication
  mh_response JSONB, -- Full response from MH
  mh_errors TEXT[], -- Array of error messages if rejected
  submission_count INTEGER NOT NULL DEFAULT 0,
  
  -- Timestamps
  generated_at TIMESTAMP NOT NULL DEFAULT NOW(),
  signed_at TIMESTAMP,
  submitted_at TIMESTAMP,
  accepted_at TIMESTAMP,
  rejected_at TIMESTAMP,
  invalidated_at TIMESTAMP,
  
  -- Relationships
  contingency_event_id UUID REFERENCES dte_contingency_events(id),
  invalidation_event_id UUID REFERENCES dte_invalidation_events(id),
  replaces_dte_id UUID REFERENCES dtes(id), -- For invalidations with replacement
  
  -- Audit
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
  created_by UUID REFERENCES users(id),
  
  -- Indexes
  CONSTRAINT chk_status CHECK (status IN ('draft', 'generated', 'signed', 'submitted', 'accepted', 'rejected', 'invalidated', 'contingency')),
  CONSTRAINT chk_ambiente CHECK (ambiente IN ('00', '01'))
);

CREATE INDEX idx_dtes_company_status ON dtes(company_id, status);
CREATE INDEX idx_dtes_codigo_generacion ON dtes(codigo_generacion);
CREATE INDEX idx_dtes_numero_control ON dtes(numero_control);
CREATE INDEX idx_dtes_invoice ON dtes(invoice_id);
CREATE INDEX idx_dtes_submitted_at ON dtes(submitted_at) WHERE status = 'submitted';
4.2 Submission Audit Log
sqlCREATE TABLE dte_submissions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  dte_id UUID NOT NULL REFERENCES dtes(id),
  
  -- Submission Details
  attempt_number INTEGER NOT NULL,
  endpoint VARCHAR(255) NOT NULL, -- firmador or hacienda
  request_payload JSONB NOT NULL,
  response_payload JSONB,
  
  -- Result
  success BOOLEAN NOT NULL DEFAULT FALSE,
  error_message TEXT,
  http_status_code INTEGER,
  
  -- Timing
  started_at TIMESTAMP NOT NULL DEFAULT NOW(),
  completed_at TIMESTAMP,
  duration_ms INTEGER,
  
  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_dte_submissions_dte ON dte_submissions(dte_id);
CREATE INDEX idx_dte_submissions_created ON dte_submissions(created_at DESC);
4.3 Contingency Events
sqlCREATE TABLE dte_contingency_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID NOT NULL REFERENCES companies(id),
  
  -- Event Identification
  codigo_generacion UUID NOT NULL UNIQUE,
  
  -- Contingency Details
  tipo_contingencia VARCHAR(1) NOT NULL, -- CAT-005
  motivo_contingencia TEXT NOT NULL,
  fecha_inicio TIMESTAMP NOT NULL,
  fecha_fin TIMESTAMP NOT NULL,
  
  -- Event Status
  status VARCHAR(20) NOT NULL, -- draft, submitted, accepted, rejected
  
  -- Storage
  event_json JSONB NOT NULL,
  sello_recibido VARCHAR(255),
  
  -- MH Response
  mh_response JSONB,
  
  -- Timestamps
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  submitted_at TIMESTAMP,
  accepted_at TIMESTAMP,
  
  -- Stats
  total_dtes INTEGER NOT NULL DEFAULT 0,
  dtes_submitted INTEGER NOT NULL DEFAULT 0,
  dtes_accepted INTEGER NOT NULL DEFAULT 0,
  
  CONSTRAINT chk_contingency_status CHECK (status IN ('draft', 'submitted', 'accepted', 'rejected')),
  CONSTRAINT chk_max_dtes CHECK (total_dtes <= 5000)
);

CREATE INDEX idx_contingency_events_company ON dte_contingency_events(company_id);
CREATE INDEX idx_contingency_events_status ON dte_contingency_events(status);
4.4 Invalidation Events
sqlCREATE TABLE dte_invalidation_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID NOT NULL REFERENCES companies(id),
  dte_id UUID NOT NULL REFERENCES dtes(id),
  
  -- Event Identification
  codigo_generacion UUID NOT NULL UNIQUE,
  
  -- Invalidation Details
  tipo_invalidacion VARCHAR(1) NOT NULL, -- CAT-024: 1=Error, 2=Rescindir, 3=Otro
  motivo TEXT NOT NULL,
  fecha_evento DATE NOT NULL,
  
  -- Replacement DTE (if applicable)
  replacement_dte_id UUID REFERENCES dtes(id),
  
  -- Event Status
  status VARCHAR(20) NOT NULL, -- draft, submitted, accepted, rejected
  
  -- Storage
  event_json JSONB NOT NULL,
  sello_recibido VARCHAR(255),
  
  -- MH Response
  mh_response JSONB,
  
  -- Responsible Parties
  responsable_emisor_nombre VARCHAR(255) NOT NULL,
  responsable_emisor_doc VARCHAR(50) NOT NULL,
  solicitante_nombre VARCHAR(255) NOT NULL,
  solicitante_doc VARCHAR(50) NOT NULL,
  
  -- Timestamps
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  submitted_at TIMESTAMP,
  accepted_at TIMESTAMP,
  
  CONSTRAINT chk_invalidation_status CHECK (status IN ('draft', 'submitted', 'accepted', 'rejected')),
  CONSTRAINT chk_tipo_invalidacion CHECK (tipo_invalidacion IN ('1', '2', '3'))
);

CREATE INDEX idx_invalidation_events_dte ON dte_invalidation_events(dte_id);
CREATE INDEX idx_invalidation_events_status ON dte_invalidation_events(status);

5. State Machine
5.1 State Diagram
┌─────────┐
│  draft  │ (Invoice finalized, DTE not created)
└────┬────┘
     │ GenerateDTE()
     ▼
┌───────────┐
│ generated │ (DTE JSON created with codigo_generacion)
└─────┬─────┘
      │ SignDTE()
      ▼
┌─────────┐
│ signed  │ (Firma electronica received from firmador)
└────┬────┘
     │ SubmitToHacienda()
     ▼
┌───────────┐
│ submitted │ (Sent to MH, awaiting response)
└─────┬─────┘
      │
      ├─ Success: ReceiveSello()
      │     ▼
      │  ┌──────────┐
      │  │ accepted │ (Sello recibido - official DTE)
      │  └────┬─────┘
      │       │ Invalidate() [within legal timeframe]
      │       ▼
      │  ┌─────────────┐
      │  │ invalidated │ (Terminal state)
      │  └─────────────┘
      │
      └─ Failure: ReceiveRejection()
            ▼
         ┌──────────┐
         │ rejected │ (MH rejected with errors)
         └────┬─────┘
              │ FixErrors()
              └──> back to generated

┌─────────────┐
│ contingency │ (Generated offline, pending submission)
└──────┬──────┘
       │ SubmitContingencyEvent() + SubmitDTE()
       └──> submitted
5.2 State Transitions
From StateEventTo StateConditionsSide EffectsdraftGenerateDTEgeneratedInvoice must be finalizedCreate DTE JSON, assign codigo_generacion & numero_controlgeneratedSignDTEsignedDTE JSON validStore firma_electronicasignedSubmitToHaciendasubmittedFirmador succeededRecord submission timestamp, increment attempt countersubmittedReceiveSelloacceptedMH returns selloRecibidoStore sello, mark invoice as dte_compliantsubmittedReceiveRejectionrejectedMH returns errorsStore error details, allow retryrejectedFixAndRegenerategeneratedErrors correctedRegenerate DTE JSON with fixesacceptedInvalidateinvalidatedWithin legal timeframe, replacement DTE exists (if required)Create invalidation event, mark original DTEsignedEnterContingencycontingencyMH unavailable after retriesCreate contingency event recordcontingencySubmitAfterRecoverysubmittedContingency event accepted, within 72hr windowNormal submission flow resumes
5.3 Business Rules by State
draft:

Can modify invoice
Can delete invoice
No DTE artifacts exist yet

generated:

Invoice locked (cannot modify)
Can retry generation if needed
DTE JSON stored but not signed

signed:

Ready for submission
Can retry signing if corrupted
Firma electronica stored

submitted:

Waiting for MH response
Can check status via MH API
May timeout and retry

accepted:

Immutable (except via invalidation)
Can print version legible
Must be archived per tax law
Can be invalidated within legal timeframes

rejected:

Need to analyze errors
May require data fixes
Can regenerate after corrections

invalidated:

Terminal state
Original DTE still in system but marked invalid
Must have corresponding invalidation event


