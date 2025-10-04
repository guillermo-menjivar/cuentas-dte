Phase 2 Transaction/Invoice Layer - Implementation Plan
Overview
Build an immutable transaction system with complete snapshots at point of sale. Every invoice is self-contained and auditable indefinitely. DTE (government compliance) is tracked separately but guaranteed for all finalized invoices.
Core Principles

Immutability - No updates/deletes on finalized invoices
Complete Snapshots - Capture all data that could change
Audit Trail - Every transaction fully traceable
DTE Guarantee - Every finalized invoice MUST have DTE record
Business First - Invoice represents business transaction; DTE represents compliance

Database Schema
1. Update clients table (Credit Tracking)
sqlALTER TABLE clients ADD COLUMN credit_limit DECIMAL(15,2) DEFAULT 0;
ALTER TABLE clients ADD COLUMN current_balance DECIMAL(15,2) DEFAULT 0;
ALTER TABLE clients ADD COLUMN credit_status VARCHAR(20) DEFAULT 'good_standing';
2. invoices table
sqlCREATE TABLE invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id),
    client_id UUID NOT NULL REFERENCES clients(id),
    
    -- Invoice identification
    invoice_number VARCHAR(50) NOT NULL,
    invoice_type VARCHAR(20) NOT NULL DEFAULT 'sale',
    
    -- Reference (for voids/corrections)
    references_invoice_id UUID REFERENCES invoices(id),
    void_reason TEXT,
    
    -- Client snapshot (at transaction time)
    client_name VARCHAR(255) NOT NULL,
    client_legal_name VARCHAR(255) NOT NULL,
    client_nit VARCHAR(20),
    client_ncr VARCHAR(10),
    client_dui VARCHAR(15),
    client_address TEXT NOT NULL,
    client_tipo_contribuyente VARCHAR(50),
    client_tipo_persona VARCHAR(1),
    
    -- Financial totals
    subtotal DECIMAL(15,2) NOT NULL,
    total_discount DECIMAL(15,2) DEFAULT 0,
    total_taxes DECIMAL(15,2) NOT NULL,
    total DECIMAL(15,2) NOT NULL,
    
    currency VARCHAR(3) DEFAULT 'USD',
    
    -- Payment tracking
    payment_terms VARCHAR(50) DEFAULT 'cash',
    payment_status VARCHAR(20) DEFAULT 'unpaid',
    amount_paid DECIMAL(15,2) DEFAULT 0,
    balance_due DECIMAL(15,2) NOT NULL,
    due_date DATE,
    
    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    
    -- DTE tracking (populated when finalized)
    dte_codigo_generacion UUID,
    dte_numero_control VARCHAR(50),
    dte_status VARCHAR(20),
    dte_hacienda_response JSONB,
    dte_submitted_at TIMESTAMP,
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    finalized_at TIMESTAMP,
    voided_at TIMESTAMP,
    
    -- Audit
    created_by UUID,
    voided_by UUID,
    notes TEXT,
    
    CONSTRAINT unique_invoice_number UNIQUE (company_id, invoice_number),
    CONSTRAINT unique_dte_codigo UNIQUE (dte_codigo_generacion) WHERE dte_codigo_generacion IS NOT NULL,
    CONSTRAINT check_invoice_type CHECK (invoice_type IN ('sale', 'credit_note', 'debit_note')),
    CONSTRAINT check_status CHECK (status IN ('draft', 'finalized', 'void')),
    CONSTRAINT check_payment_status CHECK (payment_status IN ('unpaid', 'partial', 'paid', 'overdue')),
    CONSTRAINT check_finalized_has_dte CHECK (
        (status = 'draft') OR 
        (status IN ('finalized', 'void') AND dte_codigo_generacion IS NOT NULL)
    )
);

CREATE INDEX idx_invoices_company ON invoices(company_id);
CREATE INDEX idx_invoices_client ON invoices(client_id);
CREATE INDEX idx_invoices_date ON invoices(created_at);
CREATE INDEX idx_invoices_finalized ON invoices(finalized_at);
CREATE INDEX idx_invoices_number ON invoices(company_id, invoice_number);
CREATE INDEX idx_invoices_status ON invoices(status);
CREATE INDEX idx_invoices_payment_status ON invoices(payment_status);
CREATE INDEX idx_invoices_dte_codigo ON invoices(dte_codigo_generacion);
3. invoice_line_items table
sqlCREATE TABLE invoice_line_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    line_number INT NOT NULL,
    
    -- Item reference (for tracking only, not pricing)
    item_id UUID REFERENCES inventory_items(id),
    
    -- Item snapshot (complete state at transaction time)
    item_sku VARCHAR(100) NOT NULL,
    item_name VARCHAR(255) NOT NULL,
    item_description TEXT,
    item_tipo_item VARCHAR(1) NOT NULL,
    unit_of_measure VARCHAR(50) NOT NULL,
    
    -- Pricing snapshot
    unit_price DECIMAL(15,2) NOT NULL,
    quantity DECIMAL(15,3) NOT NULL,
    line_subtotal DECIMAL(15,2) NOT NULL,
    
    -- Discount (line-level)
    discount_percentage DECIMAL(5,2) DEFAULT 0,
    discount_amount DECIMAL(15,2) DEFAULT 0,
    
    -- Tax calculations
    taxable_amount DECIMAL(15,2) NOT NULL,
    total_taxes DECIMAL(15,2) NOT NULL,
    line_total DECIMAL(15,2) NOT NULL,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT check_quantity_positive CHECK (quantity > 0),
    CONSTRAINT check_line_number_positive CHECK (line_number > 0),
    CONSTRAINT unique_invoice_line UNIQUE (invoice_id, line_number)
);

CREATE INDEX idx_line_items_invoice ON invoice_line_items(invoice_id);
CREATE INDEX idx_line_items_item ON invoice_line_items(item_id);
4. invoice_line_item_taxes table
sqlCREATE TABLE invoice_line_item_taxes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    line_item_id UUID NOT NULL REFERENCES invoice_line_items(id) ON DELETE CASCADE,
    
    -- Tax snapshot (complete state at transaction time)
    tributo_code VARCHAR(10) NOT NULL,
    tributo_name VARCHAR(255) NOT NULL,
    
    -- Rate and calculation
    tax_rate DECIMAL(8,6) NOT NULL,
    taxable_base DECIMAL(15,2) NOT NULL,
    tax_amount DECIMAL(15,2) NOT NULL,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT check_tax_rate_valid CHECK (tax_rate >= 0 AND tax_rate <= 1)
);

CREATE INDEX idx_line_item_taxes_line ON invoice_line_item_taxes(line_item_id);
CREATE INDEX idx_line_item_taxes_code ON invoice_line_item_taxes(tributo_code);
5. invoice_payments table
sqlCREATE TABLE invoice_payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES invoices(id),
    
    payment_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    payment_method VARCHAR(50) NOT NULL,
    amount DECIMAL(15,2) NOT NULL,
    
    reference_number VARCHAR(100),
    notes TEXT,
    
    created_by UUID,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT check_payment_amount_positive CHECK (amount > 0)
);

CREATE INDEX idx_payments_invoice ON invoice_payments(invoice_id);
CREATE INDEX idx_payments_date ON invoice_payments(payment_date);
6. dte_submissions table (Future Phase)
sql-- Created later when building DTE integration
CREATE TABLE dte_submissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES invoices(id),
    
    request_payload JSONB NOT NULL,
    response_payload JSONB,
    
    status VARCHAR(20) NOT NULL,
    submitted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Contingency tracking
    contingency_type INT,
    contingency_reason TEXT,
    
    attempt_number INT DEFAULT 1,
    error_message TEXT
);
Invoice Creation Flow
Step 1: Create Draft
POST /v1/invoices
{
  "client_id": "uuid",
  "payment_terms": "cash|net_30|cuenta",
  "due_date": "2025-10-15" (optional),
  "line_items": [
    {
      "item_id": "uuid",
      "quantity": 2.5,
      "discount_percentage": 5.0
    }
  ]
}
Step 2: Service Layer Snapshots Data
For invoice header:

Look up client → snapshot all fields
Generate invoice_number (sequential)
Calculate totals from line items
INSERT into invoices with status='draft'

For each line item:

Look up inventory_item by item_id
Snapshot: SKU, name, description, tipo_item, unit_of_measure, unit_price
Calculate line_subtotal = unit_price × quantity
Apply discount if provided
Calculate taxable_amount

For taxes per line item:

Look up inventory_item_taxes
For each tax, look up in codigos.Tributos → get name and current rate
Calculate tax_amount = taxable_amount × tax_rate
Store: tributo_code, tributo_name, tax_rate, taxable_base, tax_amount

Step 3: Calculate Totals

Sum all line totals → invoice total
Sum all discounts → total_discount
Sum all taxes → total_taxes
balance_due = total (initially unpaid)

Step 4: Atomic Transaction
BEGIN TRANSACTION
  INSERT invoices
  FOR EACH line_item:
    INSERT invoice_line_items
    FOR EACH tax:
      INSERT invoice_line_item_taxes
COMMIT
Step 5: Return Complete Invoice
Full invoice with all line items and tax details.
Finalize Invoice Flow
Step 1: Validate
POST /v1/invoices/:id/finalize

Verify invoice is draft
Verify all required fields present

Step 2: Check Credit Limit (if payment_terms = 'cuenta')

Get client.current_balance + invoice.total
If > client.credit_limit → reject or warn

Step 3: Generate DTE Data
1. Generate dte_codigo_generacion (UUID)
2. Generate dte_numero_control (format: DTE-01-XXXXXXXX-XXXXXXXXXXXXXXX)
3. Update invoice:
   - status = 'finalized'
   - finalized_at = NOW()
   - dte_codigo_generacion = UUID
   - dte_numero_control = control_number
   - dte_status = 'pending'
Step 4: Create DTE Submission Record
INSERT INTO dte_submissions (
  invoice_id,
  status = 'pending',
  request_payload = NULL (built later)
)
Step 5: Update Client Balance (if credit)
UPDATE clients 
SET current_balance = current_balance + invoice.total
WHERE id = invoice.client_id
Step 6: Deduct Stock (if tracking inventory)
FOR EACH line_item:
  CALL inventory_service.RecordTransaction(
    item_id,
    'sale',
    -quantity  (negative for deduction)
  )
Void Invoice Flow
POST /v1/invoices/:id/void
{
  "reason": "Customer cancelled order"
}
Steps:

Verify invoice is finalized
Create credit note (new invoice with invoice_type='credit_note')
Update original: status='void', voided_at=NOW(), void_reason=reason
Reverse client balance if credit
Reverse stock transaction

Payment Recording Flow
POST /v1/invoices/:id/payments
{
  "amount": 100.00,
  "payment_method": "cash",
  "reference_number": "CHK-12345"
}
Steps:

INSERT into invoice_payments
UPDATE invoice:

amount_paid += payment.amount
balance_due -= payment.amount
payment_status = calculate_status()


UPDATE client:

current_balance -= payment.amount



Invoice Number Generation
Format: INV-{YEAR}-{SEQUENCE}

Example: INV-2025-00001
Continuous sequence (never resets)
Unique per company

Service:
1. Get MAX(invoice_number) for company
2. Parse sequence number
3. Increment
4. Format new number
API Endpoints
POST   /v1/invoices                    Create draft invoice
GET    /v1/invoices/:id                Get invoice with details
GET    /v1/invoices                    List invoices (filters)
POST   /v1/invoices/:id/finalize       Finalize invoice
POST   /v1/invoices/:id/void           Void invoice
DELETE /v1/invoices/:id                Delete draft only
POST   /v1/invoices/:id/payments       Record payment
GET    /v1/invoices/:id/payments       List payments
Reporting Queries (Phase 3)
Sales by period:
sqlSELECT * FROM invoices 
WHERE company_id = ? 
AND finalized_at BETWEEN ? AND ?
AND status = 'finalized'
Tax collected:
sqlSELECT t.tributo_code, t.tributo_name, SUM(t.tax_amount)
FROM invoice_line_item_taxes t
JOIN invoice_line_items li ON t.line_item_id = li.id
JOIN invoices i ON li.invoice_id = i.id
WHERE i.company_id = ? AND i.finalized_at BETWEEN ? AND ?
GROUP BY t.tributo_code, t.tributo_name
Client account statement:
sqlSELECT * FROM invoices
WHERE client_id = ?
AND status = 'finalized'
ORDER BY finalized_at DESC
Aging report:
sqlSELECT 
  CASE 
    WHEN due_date >= CURRENT_DATE THEN 'current'
    WHEN due_date >= CURRENT_DATE - 30 THEN '1-30 days'
    WHEN due_date >= CURRENT_DATE - 60 THEN '31-60 days'
    ELSE '60+ days'
  END as aging_bucket,
  SUM(balance_due) as total
FROM invoices
WHERE payment_status != 'paid'
GROUP BY aging_bucket
Key Features

Complete Snapshots - Every field that could change is captured
Immutable Finalized Records - No updates/deletes after finalization
Credit Management - Track client balances and limits
Payment Tracking - Multiple payments per invoice
Void with Audit Trail - Original invoice preserved
DTE Guaranteed - Every finalized invoice has DTE record
Stock Integration - Automatic inventory deduction
Self-Contained - Invoice understandable without external lookups

Implementation Order

Migrations (invoices, line_items, taxes, payments tables)
Models with validation
Invoice service (snapshot logic)
Invoice handlers
Payment service
Reporting queries
DTE integration (separate phase)

Success Criteria

Can create draft invoices
Can finalize with complete snapshots
Historical invoices remain accurate despite price/tax changes
Full audit trail of all transactions
Credit limits enforced
Payments tracked correctly
Can generate tax reports
Can export data for accountants


Status: Ready to implement
Dependencies: Phase 1 (Inventory) ✅, Clients ✅, Codigos ✅
Next Step: Create migrations for invoice tables
