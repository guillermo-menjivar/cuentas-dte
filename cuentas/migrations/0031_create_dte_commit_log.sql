-- migrations/0031_create_dte_commit_log.sql

CREATE TABLE dte_commit_log (
    -- Primary key is the codigo_generacion (same as invoice.id)
    codigo_generacion VARCHAR(36) PRIMARY KEY,
    
    -- Invoice reference
    invoice_id VARCHAR(36) NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    invoice_number VARCHAR(50) NOT NULL,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    client_id UUID NOT NULL REFERENCES clients(id),
    establishment_id UUID NOT NULL REFERENCES establishments(id),
    point_of_sale_id UUID NOT NULL REFERENCES point_of_sale(id),
    
    -- Financial breakdown (snapshot from invoice)
    subtotal DECIMAL(15,2) NOT NULL,
    total_discount DECIMAL(15,2) NOT NULL,
    total_taxes DECIMAL(15,2) NOT NULL,
    iva_amount DECIMAL(15,2) NOT NULL,
    total_amount DECIMAL(15,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    
    -- Payment information (snapshot from invoice)
    payment_method VARCHAR(2) NOT NULL,
    payment_terms VARCHAR(50) NOT NULL,
    
    -- Document tracking
    references_invoice_id VARCHAR(36),
    
    -- DTE identification
    numero_control VARCHAR(50) NOT NULL,
    tipo_dte VARCHAR(2) NOT NULL,
    ambiente VARCHAR(2) NOT NULL,
    fecha_emision DATE NOT NULL,
    
    -- Fiscal period
    fiscal_year INTEGER NOT NULL,
    fiscal_month INTEGER NOT NULL,
    
    -- Public URL
    dte_url TEXT NOT NULL,
    
    -- DTE content (before signing)
    dte_unsigned JSONB NOT NULL,
    
    -- Signed DTE (JWT from Firmador)
    dte_signed TEXT NOT NULL,
    
    -- Hacienda response
    hacienda_estado VARCHAR(20),
    hacienda_sello_recibido VARCHAR(100),
    hacienda_fh_procesamiento TIMESTAMP WITH TIME ZONE,
    hacienda_codigo_msg VARCHAR(10),
    hacienda_descripcion_msg TEXT,
    hacienda_observaciones TEXT[],
    hacienda_response_full JSONB,
    
    -- User audit trail
    created_by UUID,
    
    -- Timestamps
    submitted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for reporting and queries
CREATE INDEX idx_dte_commit_log_invoice ON dte_commit_log(invoice_id);
CREATE INDEX idx_dte_commit_log_company ON dte_commit_log(company_id);
CREATE INDEX idx_dte_commit_log_client ON dte_commit_log(client_id);
CREATE INDEX idx_dte_commit_log_establishment ON dte_commit_log(establishment_id);
CREATE INDEX idx_dte_commit_log_pos ON dte_commit_log(point_of_sale_id);
CREATE INDEX idx_dte_commit_log_estado ON dte_commit_log(hacienda_estado);
CREATE INDEX idx_dte_commit_log_submitted ON dte_commit_log(submitted_at);
CREATE INDEX idx_dte_commit_log_fecha_emision ON dte_commit_log(fecha_emision);
CREATE INDEX idx_dte_commit_log_fiscal_period ON dte_commit_log(fiscal_year, fiscal_month);
CREATE INDEX idx_dte_commit_log_payment_method ON dte_commit_log(payment_method);
CREATE INDEX idx_dte_commit_log_tipo_dte ON dte_commit_log(tipo_dte);

-- Comments
COMMENT ON TABLE dte_commit_log IS 'Immutable audit trail of all DTE submissions to Hacienda with complete financial snapshot';
COMMENT ON COLUMN dte_commit_log.codigo_generacion IS 'Código de Generación (uppercase UUID) - primary key matching invoice.id';
COMMENT ON COLUMN dte_commit_log.invoice_number IS 'Human-readable invoice number (e.g., INV-2025-00001)';
COMMENT ON COLUMN dte_commit_log.subtotal IS 'Subtotal before discounts and taxes';
COMMENT ON COLUMN dte_commit_log.total_discount IS 'Total discount amount applied';
COMMENT ON COLUMN dte_commit_log.total_taxes IS 'Total taxes (primarily IVA)';
COMMENT ON COLUMN dte_commit_log.iva_amount IS 'IVA amount for tax reports';
COMMENT ON COLUMN dte_commit_log.total_amount IS 'Final total amount';
COMMENT ON COLUMN dte_commit_log.payment_method IS 'Payment method code from Hacienda catalog';
COMMENT ON COLUMN dte_commit_log.payment_terms IS 'Payment terms (cash, net_30, net_60, cuenta)';
COMMENT ON COLUMN dte_commit_log.references_invoice_id IS 'Referenced invoice for voids/corrections';
COMMENT ON COLUMN dte_commit_log.fiscal_year IS 'Fiscal year for period reporting';
COMMENT ON COLUMN dte_commit_log.fiscal_month IS 'Fiscal month (1-12) for period reporting';
COMMENT ON COLUMN dte_commit_log.dte_url IS 'Public URL to view the DTE on Hacienda portal';
COMMENT ON COLUMN dte_commit_log.dte_unsigned IS 'Original unsigned DTE JSON structure before signing';
COMMENT ON COLUMN dte_commit_log.dte_signed IS 'Signed JWT from Firmador service';
COMMENT ON COLUMN dte_commit_log.hacienda_response_full IS 'Complete response JSON from Hacienda API';
COMMENT ON COLUMN dte_commit_log.created_by IS 'User who submitted the DTE';
