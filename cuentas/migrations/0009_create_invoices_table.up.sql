-- Create invoices table
CREATE TABLE IF NOT EXISTS invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
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
    
    -- Constraints
    CONSTRAINT unique_invoice_number UNIQUE (company_id, invoice_number),
    CONSTRAINT check_invoice_type CHECK (invoice_type IN ('sale', 'credit_note', 'debit_note')),
    CONSTRAINT check_status CHECK (status IN ('draft', 'finalized', 'void')),
    CONSTRAINT check_payment_status CHECK (payment_status IN ('unpaid', 'partial', 'paid', 'overdue')),
    CONSTRAINT check_finalized_has_dte CHECK (
        (status = 'draft') OR
        (status IN ('finalized', 'void') AND dte_codigo_generacion IS NOT NULL)
    )
);

-- Indexes
CREATE INDEX idx_invoices_company ON invoices(company_id);
CREATE INDEX idx_invoices_client ON invoices(client_id);
CREATE INDEX idx_invoices_date ON invoices(created_at);
CREATE INDEX idx_invoices_finalized ON invoices(finalized_at);
CREATE INDEX idx_invoices_number ON invoices(company_id, invoice_number);
CREATE INDEX idx_invoices_status ON invoices(status);
CREATE INDEX idx_invoices_payment_status ON invoices(payment_status);

-- Partial unique index for DTE codigo (only when NOT NULL)
CREATE UNIQUE INDEX idx_invoices_dte_codigo_unique ON invoices(dte_codigo_generacion) WHERE dte_codigo_generacion IS NOT NULL;

-- Comments
COMMENT ON TABLE invoices IS 'Invoice transactions with complete snapshots';
COMMENT ON COLUMN invoices.invoice_number IS 'Sequential invoice number per company (e.g., INV-2025-00001)';
COMMENT ON COLUMN invoices.status IS 'Invoice status: draft (editable), finalized (immutable), void (cancelled)';
COMMENT ON COLUMN invoices.payment_terms IS 'Payment terms: cash, net_30, net_60, cuenta (credit account)';
COMMENT ON COLUMN invoices.dte_codigo_generacion IS 'UUID generated for DTE submission to Hacienda';
COMMENT ON COLUMN invoices.dte_numero_control IS 'DTE control number format: DTE-01-XXXXXXXX-XXXXXXXXXXXXXXX';
COMMENT ON COLUMN invoices.balance_due IS 'Remaining amount owed (total - amount_paid)';
