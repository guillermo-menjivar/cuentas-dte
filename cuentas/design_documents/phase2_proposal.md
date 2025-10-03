Phase 2 Transaction/Invoice Layer - Design Proposal
Overview
Build an immutable transaction system that captures complete snapshots of all data at the moment of sale. Every invoice must be self-contained and auditable indefinitely, regardless of future changes to inventory, prices, or tax rates.
Core Design Principle: Immutability
Nothing is ever updated or deleted.

Corrections are handled by void/credit notes + new invoices
All historical transactions remain exactly as they occurred
Full audit trail of every change

Database Schema
1. invoices (Header Level)
sqlCREATE TABLE invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id),
    client_id UUID NOT NULL REFERENCES clients(id),
    
    -- Invoice identification
    invoice_number VARCHAR(50) NOT NULL,
    invoice_type VARCHAR(20) NOT NULL, -- 'sale', 'credit_note', 'debit_note'
    
    -- References (for corrections/voids)
    references_invoice_id UUID REFERENCES invoices(id),
    void_reason TEXT,
    
    -- Client snapshot (at transaction time)
    client_name VARCHAR(255) NOT NULL,
    client_nit VARCHAR(20),
    client_ncr VARCHAR(10),
    client_dui VARCHAR(15),
    client_address TEXT NOT NULL,
    client_tipo_contribuyente VARCHAR(50),
    client_tipo_persona VARCHAR(1),
    
    -- Financial totals
    subtotal DECIMAL(15,2) NOT NULL,
    total_taxes DECIMAL(15,2) NOT NULL,
    total DECIMAL(15,2) NOT NULL,
    
    -- Currency
    currency VARCHAR(3) DEFAULT 'USD',
    exchange_rate DECIMAL(15,6) DEFAULT 1.0,
    
    -- Timestamps (immutable)
    transaction_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    due_date DATE,
    
    -- Audit fields
    created_by UUID, -- user who created
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- DTE integration (future)
    dte_uuid UUID,
    dte_sent_at TIMESTAMP,
    dte_status VARCHAR(20),
    
    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'draft', -- draft, finalized, void
    
    CONSTRAINT unique_invoice_number UNIQUE (company_id, invoice_number),
    CONSTRAINT check_invoice_type CHECK (invoice_type IN ('sale', 'credit_note', 'debit_note')),
    CONSTRAINT check_status CHECK (status IN ('draft', 'finalized', 'void'))
);

CREATE INDEX idx_invoices_company ON invoices(company_id);
CREATE INDEX idx_invoices_client ON invoices(client_id);
CREATE INDEX idx_invoices_date ON invoices(transaction_date);
CREATE INDEX idx_invoices_number ON invoices(company_id, invoice_number);
CREATE INDEX idx_invoices_status ON invoices(status);
2. invoice_line_items (Line Item Level)
sqlCREATE TABLE invoice_line_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    line_number INT NOT NULL,
    
    -- Item reference (for tracking only, not for pricing)
    item_id UUID REFERENCES inventory_items(id),
    
    -- Item snapshot (complete state at transaction time)
    item_sku VARCHAR(100) NOT NULL,
    item_name VARCHAR(255) NOT NULL,
    item_description TEXT,
    item_tipo_item VARCHAR(1) NOT NULL, -- 1=Bienes, 2=Servicios
    unit_of_measure VARCHAR(50) NOT NULL,
    
    -- Pricing snapshot
    unit_price DECIMAL(15,2) NOT NULL, -- Price at transaction time
    quantity DECIMAL(15,3) NOT NULL,
    line_subtotal DECIMAL(15,2) NOT NULL, -- unit_price * quantity
    
    -- Discount (if applicable)
    discount_percentage DECIMAL(5,2),
    discount_amount DECIMAL(15,2),
    
    -- Tax calculations
    taxable_amount DECIMAL(15,2) NOT NULL, -- After discount
    total_taxes DECIMAL(15,2) NOT NULL,
    line_total DECIMAL(15,2) NOT NULL,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT check_quantity_positive CHECK (quantity > 0),
    CONSTRAINT check_line_number_positive CHECK (line_number > 0)
);

CREATE INDEX idx_line_items_invoice ON invoice_line_items(invoice_id);
CREATE INDEX idx_line_items_item ON invoice_line_items(item_id);
3. invoice_line_item_taxes (Tax Detail Level)
sqlCREATE TABLE invoice_line_item_taxes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    line_item_id UUID NOT NULL REFERENCES invoice_line_items(id) ON DELETE CASCADE,
    
    -- Tax snapshot (complete state at transaction time)
    tributo_code VARCHAR(10) NOT NULL,
    tributo_name VARCHAR(255) NOT NULL, -- "Impuesto al Valor Agregado 13%"
    
    -- Rate and calculation
    tax_rate DECIMAL(8,6) NOT NULL, -- 0.130000 (stored as decimal for precision)
    taxable_base DECIMAL(15,2) NOT NULL, -- Amount this tax was calculated on
    tax_amount DECIMAL(15,2) NOT NULL, -- Calculated tax
    
    -- Metadata
    applies_to VARCHAR(20), -- 'goods', 'services', 'both' (for future tipo_item=3)
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT check_tax_rate_valid CHECK (tax_rate >= 0 AND tax_rate <= 1)
);

CREATE INDEX idx_line_item_taxes_line ON invoice_line_item_taxes(line_item_id);
CREATE INDEX idx_line_item_taxes_code ON invoice_line_item_taxes(tributo_code);
4. invoice_payments (Payment Tracking)
sqlCREATE TABLE invoice_payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES invoices(id),
    
    payment_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    payment_method VARCHAR(50) NOT NULL, -- 'cash', 'card', 'transfer', 'check'
    amount DECIMAL(15,2) NOT NULL,
    
    reference_number VARCHAR(100), -- Check number, transfer ID, etc.
    notes TEXT,
    
    created_by UUID,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT check_payment_amount_positive CHECK (amount > 0)
);

CREATE INDEX idx_payments_invoice ON invoice_payments(invoice_id);
CREATE INDEX idx_payments_date ON invoice_payments(payment_date);
Invoice Creation Flow
Step 1: Validate Input
POST /v1/invoices
{
  "client_id": "uuid",
  "line_items": [
    {
      "item_id": "uuid",
      "quantity": 2,
      "discount_percentage": 5.0  // optional
    }
  ]
}
Step 2: Snapshot Data (Service Layer)
For each line item:

Look up inventory_item by item_id
Capture snapshot:

SKU, name, description, tipo_item
unit_price at this moment
unit_of_measure


Look up inventory_item_taxes for this item
For each tax, look up in codigos.Tributos:

Get tributo name
Get current tax rate


Calculate:

Line subtotal = unit_price × quantity
Apply discount if provided
Taxable amount = subtotal - discount
For each tax:

tax_amount = taxable_amount × tax_rate


Line total = taxable_amount + sum(all tax_amounts)



Step 3: Snapshot Client Data
Look up clients table and capture:

Name, NIT, NCR, DUI
Address
Tipo contribuyente
Tipo persona

Step 4: Create Transaction (Atomic)
Within a database transaction:

INSERT into invoices (header with totals)
For each line item:

INSERT into invoice_line_items (with snapshots)
For each tax:

INSERT into invoice_line_item_taxes (with rate snapshot)




Call inventory_service.RecordTransaction() to deduct stock (Phase 4)
Commit transaction

Step 5: Return Complete Invoice
Return fully populated invoice with all line items and tax details.
Key Design Features
1. Complete Snapshots
Every field that could change is captured:

Client details (name, address can change)
Item details (name, description can change)
Prices (can change)
Tax rates (can change)
Tax names (can be renamed)

2. Immutability Enforcement
sql-- No UPDATE or DELETE allowed on these tables
-- Enforced at application level
-- Only INSERT operations permitted
3. Corrections via New Records
Original invoice: $100
Mistake found: should be $95

Solution:
1. Create credit note: -$100 (references original)
2. Create new invoice: $95
3. Original invoice remains in database
4. Audit trail shows: original → void → correction
4. Self-Contained Records
You can query any invoice and understand:

What was sold
At what price
With what taxes and rates
To which client
On what date

No dependencies on current inventory or tax data.
Invoice Number Generation
Format: INV-{YEAR}{MONTH}-{SEQUENCE}

Example: INV-202510-00001
Reset sequence monthly or yearly (business decision)
Unique per company

API Endpoints (Phase 2)
POST   /v1/invoices              Create invoice (draft)
POST   /v1/invoices/:id/finalize Finalize (locks, generates DTE)
GET    /v1/invoices/:id          Get invoice with full details
GET    /v1/invoices              List invoices (filters: date, client, status)
POST   /v1/invoices/:id/void     Void invoice (with reason)
POST   /v1/invoices/:id/payments Record payment
GET    /v1/invoices/:id/pdf      Generate PDF
Reporting Capabilities (Phase 3)
With this structure, you can generate:
Sales Reports:

Total sales by date range
Sales by client
Sales by product/service
Sales by user

Tax Reports:

IVA collected by period
Tax breakdown by tributo code
Taxable vs non-taxable sales

Audit Reports:

All transactions for a period
Voided invoices with reasons
Price history for an item
Tax rate changes over time

Export Formats:

CSV (all fields)
Excel (formatted)
PDF (invoice format)
JSON (API consumption)

Audit Trail Features
Every query can be answered:

"Show me all invoices from Q3 2024"
"What tax rate was applied to this sale?"
"What was the price of this item on March 15?"
"Which invoices were voided and why?"
"How much IVA did we collect in January?"

Integration Points
With Phase 1 (Inventory):

Read item data for snapshot
Read tax configuration
Record stock transaction on finalize

With Future DTE System:

Generate XML from snapshot data
Submit to Ministerio de Hacienda
Store DTE response (UUID, status)

With Payment System:

Track payments against invoices
Calculate outstanding balance
Payment reconciliation

Data Retention
Since everything is immutable:

Keep all data indefinitely
No cleanup needed
Compliance with 10-year retention requirements
Can always regenerate exact invoice as sent

Status Flow
draft → finalized → [paid]
           ↓
         void (with reason)

draft: Can still be edited (delete + recreate)
finalized: Immutable, DTE sent
void: Cancelled, credit note created
paid: Payment recorded (status tracking only)


Dependencies:

Phase 1 (Inventory) - Complete ✅
codigos.Tributos - Available ✅
Client management - Complete ✅

Next Steps:

Create migrations for invoice tables
Build invoice models with validation
Implement invoice service (snapshot logic)
Create invoice handlers
Build PDF generation
Add reporting endpoints

Estimated Complexity: High
Estimated Time: 2-3 weeks
Critical for: DTE compliance, audit requirements
