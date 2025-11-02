-- ============================================================================
-- Migration: Create Nota de Débito Tables
-- Description: Tables for managing Notas de Débito (price adjustment documents)
-- ============================================================================

-- Main notas_debito table
CREATE TABLE IF NOT EXISTS notas_debito (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL,
    establishment_id UUID NOT NULL,
    point_of_sale_id UUID NOT NULL,
    
    -- Identification
    nota_number VARCHAR(50) NOT NULL,
    nota_type VARCHAR(10) DEFAULT '06', -- Always '06' for Nota de Débito
    
    -- Client info (snapshot from CCF)
    client_id UUID NOT NULL,
    client_name VARCHAR(255) NOT NULL,
    client_legal_name VARCHAR(255) NOT NULL,
    client_nit VARCHAR(20),
    client_ncr VARCHAR(20),
    client_dui VARCHAR(20),
    contact_email VARCHAR(255),
    contact_whatsapp VARCHAR(20),
    client_address TEXT NOT NULL,
    client_tipo_contribuyente VARCHAR(10),
    client_tipo_persona VARCHAR(10),
    
    -- Financial totals
    subtotal DECIMAL(12,2) NOT NULL,
    total_discount DECIMAL(12,2) DEFAULT 0,
    total_taxes DECIMAL(12,2) NOT NULL,
    total DECIMAL(12,2) NOT NULL,
    
    currency VARCHAR(3) DEFAULT 'USD',
    
    -- Payment
    payment_terms VARCHAR(50),
    payment_method VARCHAR(10) NOT NULL,
    due_date TIMESTAMP,
    
    -- Status
    status VARCHAR(20) NOT NULL, -- 'draft', 'finalized', 'voided'
    
    -- DTE tracking
    dte_numero_control VARCHAR(50),
    dte_codigo_generacion VARCHAR(50),
    dte_sello_recibido VARCHAR(255),
    dte_status VARCHAR(20),
    dte_hacienda_response TEXT,
    dte_submitted_at TIMESTAMP,
    dte_fecha_procesamiento TIMESTAMP,
    dte_observaciones TEXT[],
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    finalized_at TIMESTAMP,
    voided_at TIMESTAMP,
    
    -- Audit
    created_by UUID,
    voided_by UUID,
    notes TEXT,
    
    FOREIGN KEY (company_id) REFERENCES companies(id),
    FOREIGN KEY (establishment_id) REFERENCES establishments(id),
    FOREIGN KEY (point_of_sale_id) REFERENCES point_of_sale(id),
    FOREIGN KEY (client_id) REFERENCES clients(id),
    
    CONSTRAINT unique_nota_number UNIQUE (company_id, nota_number)
);

-- Line items for nota de débito (adjustments to CCF line items)
CREATE TABLE IF NOT EXISTS nota_debito_line_items (
    id UUID PRIMARY KEY,
    nota_debito_id UUID NOT NULL,
    line_number INT NOT NULL,
    
    -- Which CCF and line item this adjusts
    related_ccf_id UUID NOT NULL,
    related_ccf_number VARCHAR(50) NOT NULL,
    ccf_line_item_id UUID NOT NULL,
    
    -- Original item details (snapshot from CCF line item)
    original_item_sku VARCHAR(100) NOT NULL,
    original_item_name VARCHAR(255) NOT NULL,
    original_unit_price DECIMAL(12,2) NOT NULL,
    original_quantity DECIMAL(10,2) NOT NULL,
    original_item_tipo_item VARCHAR(10) NOT NULL,
    original_unit_of_measure VARCHAR(10) NOT NULL,
    
    -- Adjustment details
    adjustment_amount DECIMAL(12,2) NOT NULL, -- Per-unit price increase
    adjustment_reason TEXT,
    
    -- Calculated totals for this adjustment line
    line_subtotal DECIMAL(12,2) NOT NULL,
    discount_amount DECIMAL(12,2) DEFAULT 0,
    taxable_amount DECIMAL(12,2) NOT NULL,
    total_taxes DECIMAL(12,2) NOT NULL,
    line_total DECIMAL(12,2) NOT NULL,
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    FOREIGN KEY (nota_debito_id) REFERENCES notas_debito(id) ON DELETE CASCADE,
    FOREIGN KEY (related_ccf_id) REFERENCES invoices(id),
    FOREIGN KEY (ccf_line_item_id) REFERENCES invoice_line_items(id),
    
    CONSTRAINT unique_line_number UNIQUE (nota_debito_id, line_number),
    -- Prevent duplicate adjustments to same CCF line item
    CONSTRAINT unique_ccf_line_adjustment UNIQUE (nota_debito_id, ccf_line_item_id)
);

-- Link table: Which CCFs does this nota reference?
CREATE TABLE IF NOT EXISTS nota_debito_ccf_references (
    id UUID PRIMARY KEY,
    nota_debito_id UUID NOT NULL,
    ccf_id UUID NOT NULL,
    ccf_number VARCHAR(50) NOT NULL,
    ccf_date DATE NOT NULL,
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    FOREIGN KEY (nota_debito_id) REFERENCES notas_debito(id) ON DELETE CASCADE,
    FOREIGN KEY (ccf_id) REFERENCES invoices(id),
    
    CONSTRAINT unique_nota_ccf UNIQUE (nota_debito_id, ccf_id)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_notas_debito_company ON notas_debito(company_id);
CREATE INDEX IF NOT EXISTS idx_notas_debito_client ON notas_debito(client_id);
CREATE INDEX IF NOT EXISTS idx_notas_debito_status ON notas_debito(status);
CREATE INDEX IF NOT EXISTS idx_notas_debito_created_at ON notas_debito(created_at);
CREATE INDEX IF NOT EXISTS idx_notas_debito_numero_control ON notas_debito(dte_numero_control);

CREATE INDEX IF NOT EXISTS idx_nota_line_items_nota ON nota_debito_line_items(nota_debito_id);
CREATE INDEX IF NOT EXISTS idx_nota_line_items_ccf ON nota_debito_line_items(related_ccf_id);
CREATE INDEX IF NOT EXISTS idx_nota_line_items_ccf_line ON nota_debito_line_items(ccf_line_item_id);

CREATE INDEX IF NOT EXISTS idx_nota_ccf_refs_nota ON nota_debito_ccf_references(nota_debito_id);
CREATE INDEX IF NOT EXISTS idx_nota_ccf_refs_ccf ON nota_debito_ccf_references(ccf_id);

-- Comments for documentation
COMMENT ON TABLE notas_debito IS 'Notas de Débito - Documents for increasing prices on existing CCF invoices';
COMMENT ON TABLE nota_debito_line_items IS 'Line items representing price adjustments to CCF line items';
COMMENT ON TABLE nota_debito_ccf_references IS 'Links notas de débito to the CCFs they reference';

COMMENT ON COLUMN nota_debito_line_items.adjustment_amount IS 'Per-unit price increase (e.g., if original was $50 and adjustment is $10, new effective price is $60)';
COMMENT ON COLUMN nota_debito_line_items.line_subtotal IS 'Total adjustment amount: adjustment_amount × original_quantity';
