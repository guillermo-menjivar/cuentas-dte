-- migrations/0033_create_notas_credito.up.sql
-- ============================================================================
-- NOTAS DE CRÉDITO (Credit Notes) - Type 05
-- Used to DECREASE invoice amounts (returns, discounts, voids, corrections)
-- ============================================================================

-- Main notas_credito table
CREATE TABLE notas_credito (
    -- Identity
    id VARCHAR(36) PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    establishment_id UUID NOT NULL REFERENCES establishments(id),
    point_of_sale_id UUID NOT NULL REFERENCES point_of_sale(id),

    -- Document identification
    nota_number VARCHAR(50) NOT NULL,
    nota_type VARCHAR(2) NOT NULL DEFAULT '05',

    -- Client information (snapshot from CCFs)
    client_id UUID NOT NULL REFERENCES clients(id),
    client_name VARCHAR(200) NOT NULL,
    client_legal_name VARCHAR(200),
    client_nit VARCHAR(17),
    client_ncr VARCHAR(8),
    client_dui VARCHAR(10),
    contact_email VARCHAR(100),
    contact_whatsapp VARCHAR(20),
    client_address TEXT,
    client_tipo_contribuyente VARCHAR(2),
    client_tipo_persona VARCHAR(1),

    -- Credit reason and details
    credit_reason VARCHAR(20) NOT NULL,
    credit_description TEXT,
    is_full_annulment BOOLEAN DEFAULT FALSE,

    -- Financial totals (POSITIVE numbers - amount being credited)
    subtotal DECIMAL(15,2) NOT NULL DEFAULT 0,
    total_discount DECIMAL(15,2) NOT NULL DEFAULT 0,
    total_taxes DECIMAL(15,2) NOT NULL DEFAULT 0,
    total DECIMAL(15,2) NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',

    -- Payment information
    payment_terms VARCHAR(50) NOT NULL DEFAULT 'net_30',
    payment_method VARCHAR(2) NOT NULL DEFAULT '01',
    due_date DATE,

    -- Status workflow: draft → finalized → voided
    status VARCHAR(20) NOT NULL DEFAULT 'draft',

    -- DTE tracking (Hacienda integration)
    dte_numero_control VARCHAR(50),
    dte_codigo_generacion VARCHAR(36),
    dte_sello_recibido VARCHAR(100),
    dte_status VARCHAR(20),
    dte_hacienda_response JSONB,
    dte_submitted_at TIMESTAMP WITH TIME ZONE,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    finalized_at TIMESTAMP WITH TIME ZONE,
    voided_at TIMESTAMP WITH TIME ZONE,

    -- Audit
    created_by UUID,
    notes TEXT,

    -- Constraints
    CONSTRAINT valid_nota_type CHECK (nota_type = '05'),
    CONSTRAINT valid_status CHECK (status IN ('draft', 'finalized', 'voided')),
    CONSTRAINT valid_credit_reason CHECK (credit_reason IN (
        'void', 'return', 'discount', 'defect', 'overbilling',
        'correction', 'quality', 'cancellation', 'other'
    )),
    CONSTRAINT positive_totals CHECK (
        subtotal >= 0 AND
        total_discount >= 0 AND
        total_taxes >= 0 AND
        total >= 0
    )
);

-- Line items for notas_credito
CREATE TABLE notas_credito_line_items (
    id VARCHAR(36) PRIMARY KEY,
    nota_credito_id VARCHAR(36) NOT NULL REFERENCES notas_credito(id) ON DELETE CASCADE,
    line_number INTEGER NOT NULL,

    -- Which CCF and line item this credits
    related_ccf_id VARCHAR(36) NOT NULL REFERENCES invoices(id),
    related_ccf_number VARCHAR(50) NOT NULL,
    ccf_line_item_id UUID NOT NULL REFERENCES invoice_line_items(id),  -- UUID! Not VARCHAR

    -- Original item details (snapshot from CCF)
    original_item_sku VARCHAR(100) NOT NULL,
    original_item_name VARCHAR(500) NOT NULL,
    original_unit_price DECIMAL(15,2) NOT NULL,
    original_quantity DECIMAL(15,8) NOT NULL,
    original_item_tipo_item VARCHAR(1) NOT NULL,
    original_unit_of_measure VARCHAR(50) NOT NULL,

    -- Credit details
    quantity_credited DECIMAL(15,8) NOT NULL,
    credit_amount DECIMAL(15,2) NOT NULL,
    credit_reason VARCHAR(200),

    -- Calculated totals for THIS credit
    line_subtotal DECIMAL(15,2) NOT NULL,
    discount_amount DECIMAL(15,2) NOT NULL DEFAULT 0,
    taxable_amount DECIMAL(15,2) NOT NULL,
    total_taxes DECIMAL(15,2) NOT NULL,
    line_total DECIMAL(15,2) NOT NULL,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Constraints
    CONSTRAINT positive_quantities CHECK (
        quantity_credited > 0 AND
        quantity_credited <= original_quantity
    ),
    CONSTRAINT positive_amounts CHECK (
        credit_amount >= 0 AND
        line_subtotal >= 0 AND
        total_taxes >= 0 AND
        line_total >= 0
    )
);

-- CCF references for notas_credito
CREATE TABLE notas_credito_ccf_references (
    id VARCHAR(36) PRIMARY KEY,
    nota_credito_id VARCHAR(36) NOT NULL REFERENCES notas_credito(id) ON DELETE CASCADE,
    ccf_id VARCHAR(36) NOT NULL REFERENCES invoices(id),
    ccf_number VARCHAR(50) NOT NULL,
    ccf_date DATE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Prevent duplicate references
    UNIQUE(nota_credito_id, ccf_id)
);

-- ============================================================================
-- INDEXES
-- ============================================================================

-- Notas credito indexes
CREATE INDEX idx_notas_credito_company ON notas_credito(company_id);
CREATE INDEX idx_notas_credito_client ON notas_credito(client_id);
CREATE INDEX idx_notas_credito_establishment ON notas_credito(establishment_id);
CREATE INDEX idx_notas_credito_pos ON notas_credito(point_of_sale_id);
CREATE INDEX idx_notas_credito_status ON notas_credito(status);
CREATE INDEX idx_notas_credito_credit_reason ON notas_credito(credit_reason);
CREATE INDEX idx_notas_credito_created_at ON notas_credito(created_at);
CREATE INDEX idx_notas_credito_finalized_at ON notas_credito(finalized_at);
CREATE INDEX idx_notas_credito_numero_control ON notas_credito(dte_numero_control);
CREATE INDEX idx_notas_credito_codigo_gen ON notas_credito(dte_codigo_generacion);
CREATE INDEX idx_notas_credito_full_annulment ON notas_credito(is_full_annulment) WHERE is_full_annulment = TRUE;

-- Line items indexes
CREATE INDEX idx_nc_line_items_nota ON notas_credito_line_items(nota_credito_id);
CREATE INDEX idx_nc_line_items_ccf ON notas_credito_line_items(related_ccf_id);
CREATE INDEX idx_nc_line_items_ccf_line ON notas_credito_line_items(ccf_line_item_id);

-- CCF references indexes
CREATE INDEX idx_nc_ccf_refs_nota ON notas_credito_ccf_references(nota_credito_id);
CREATE INDEX idx_nc_ccf_refs_ccf ON notas_credito_ccf_references(ccf_id);

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE notas_credito IS 'Credit notes (tipo DTE 05) - used to decrease invoice amounts via returns, discounts, voids, or corrections';
COMMENT ON COLUMN notas_credito.credit_reason IS 'Why the credit is being issued: void (full cancellation), return (goods returned), discount (price reduction), defect (compensation), etc';
COMMENT ON COLUMN notas_credito.is_full_annulment IS 'TRUE if this nota de crédito voids 100% of all referenced CCFs';
COMMENT ON COLUMN notas_credito.subtotal IS 'POSITIVE amount being credited before taxes';
COMMENT ON COLUMN notas_credito.total IS 'POSITIVE total amount being credited (reduces customer balance)';

COMMENT ON TABLE notas_credito_line_items IS 'Individual line item credits - can be partial (3 of 10 units) or full';
COMMENT ON COLUMN notas_credito_line_items.quantity_credited IS 'Number of units being credited - can be less than original quantity for partial returns';
COMMENT ON COLUMN notas_credito_line_items.credit_amount IS 'POSITIVE per-unit credit amount';

COMMENT ON TABLE notas_credito_ccf_references IS 'Links nota de crédito to the CCF invoices being credited';
