-- Main nota table
CREATE TABLE notas_debito (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL,
    establishment_id UUID NOT NULL,
    point_of_sale_id UUID NOT NULL,
    
    -- Identification
    nota_number VARCHAR NOT NULL,
    nota_type VARCHAR DEFAULT '06', -- Always Nota de Débito
    
    -- Client info (from first CCF)
    client_id UUID NOT NULL,
    client_name VARCHAR NOT NULL,
    -- ... other client snapshot fields
    
    -- Financial totals
    subtotal DECIMAL(12,2) NOT NULL,
    total_discount DECIMAL(12,2) DEFAULT 0,
    total_taxes DECIMAL(12,2) NOT NULL,
    total DECIMAL(12,2) NOT NULL,
    
    currency VARCHAR DEFAULT 'USD',
    
    -- Status
    status VARCHAR NOT NULL, -- 'draft', 'finalized', 'voided'
    
    -- DTE tracking (same as invoices)
    dte_numero_control VARCHAR,
    dte_sello_recibido VARCHAR,
    dte_status VARCHAR,
    dte_codigo_generacion VARCHAR,
    dte_submitted_at TIMESTAMP,
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    finalized_at TIMESTAMP,
    voided_at TIMESTAMP,
    
    -- Audit
    created_by UUID,
    notes TEXT,
    
    FOREIGN KEY (company_id) REFERENCES companies(id),
    FOREIGN KEY (client_id) REFERENCES clients(id)
);

-- Line items for the nota
CREATE TABLE nota_debito_line_items (
    id UUID PRIMARY KEY,
    nota_debito_id UUID NOT NULL,
    line_number INT NOT NULL,
    
    -- Which CCF this adjusts
    related_ccf_id UUID NOT NULL,
    
    -- Which specific line item in the CCF (for adjustments)
    ccf_line_item_id UUID NOT NULL,
    
    -- Original item details (snapshot from CCF)
    original_item_sku VARCHAR NOT NULL,
    original_item_name VARCHAR NOT NULL,
    original_unit_price DECIMAL(12,2) NOT NULL,
    original_quantity DECIMAL(10,2) NOT NULL,
    
    -- Adjustment details
    adjustment_amount DECIMAL(12,2) NOT NULL, -- The additional charge
    adjustment_reason TEXT,
    
    -- Calculated totals for THIS adjustment
    line_subtotal DECIMAL(12,2) NOT NULL,
    discount_amount DECIMAL(12,2) DEFAULT 0,
    taxable_amount DECIMAL(12,2) NOT NULL,
    total_taxes DECIMAL(12,2) NOT NULL,
    line_total DECIMAL(12,2) NOT NULL,
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    FOREIGN KEY (nota_debito_id) REFERENCES notas_debito(id),
    FOREIGN KEY (related_ccf_id) REFERENCES invoices(id),
    FOREIGN KEY (ccf_line_item_id) REFERENCES invoice_line_items(id)
);

-- Link table: Which CCFs does this nota reference?
CREATE TABLE nota_debito_ccf_references (
    id UUID PRIMARY KEY,
    nota_debito_id UUID NOT NULL,
    ccf_id UUID NOT NULL,
    ccf_number VARCHAR NOT NULL,
    ccf_date DATE NOT NULL,
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    FOREIGN KEY (nota_debito_id) REFERENCES notas_debito(id),
    FOREIGN KEY (ccf_id) REFERENCES invoices(id),
    UNIQUE(nota_debito_id, ccf_id)
);
