-- Drop trigger
DROP TRIGGER IF EXISTS update_companies_updated_at ON companies;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_companies_last_activity;
DROP INDEX IF EXISTS idx_companies_active;
DROP INDEX IF EXISTS idx_companies_email;
DROP INDEX IF EXISTS idx_companies_ncr;
DROP INDEX IF EXISTS idx_companies_nit;

-- Drop table
DROP TABLE IF EXISTS companies;
CREATE TABLE IF NOT EXISTS companies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    nit BIGINT NOT NULL UNIQUE,
    ncr BIGINT NOT NULL UNIQUE,
    hc_username TEXT NOT NULL,
    hc_password_ref TEXT NOT NULL,
    last_activity_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    email TEXT NOT NULL UNIQUE,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_companies_nit ON companies(nit);
CREATE INDEX IF NOT EXISTS idx_companies_ncr ON companies(ncr);
CREATE INDEX IF NOT EXISTS idx_companies_email ON companies(email);
CREATE INDEX IF NOT EXISTS idx_companies_active ON companies(active);
CREATE INDEX IF NOT EXISTS idx_companies_last_activity ON companies(last_activity_at);

-- Create function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger to automatically update updated_at
CREATE TRIGGER update_companies_updated_at 
    BEFORE UPDATE ON companies 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();
DROP TRIGGER IF EXISTS trigger_update_clients_updated_at ON clients;
DROP FUNCTION IF EXISTS update_clients_updated_at();
DROP TABLE IF EXISTS clients;
CREATE TABLE IF NOT EXISTS clients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    
    -- Tax identification numbers (stored as BIGINT, displayed with formatting)
    ncr BIGINT NOT NULL,
    nit BIGINT NOT NULL,
    dui BIGINT NOT NULL,
    
    -- Business information
    business_name VARCHAR(255) NOT NULL,
    legal_business_name VARCHAR(255) NOT NULL,
    giro VARCHAR(255) NOT NULL,
    tipo_contribuyente VARCHAR(100) NOT NULL,
    
    -- Address information
    full_address TEXT NOT NULL,
    country_code VARCHAR(2) NOT NULL DEFAULT 'SV',
    department_code VARCHAR(2) NOT NULL,
    municipality_code VARCHAR(4) NOT NULL,
    
    -- Metadata
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Indexes for faster lookups
    CONSTRAINT unique_client_nit_per_company UNIQUE (company_id, nit),
    CONSTRAINT unique_client_dui_per_company UNIQUE (company_id, dui)
);

-- Indexes for common queries
CREATE INDEX idx_clients_company_id ON clients(company_id);
CREATE INDEX idx_clients_nit ON clients(nit);
CREATE INDEX idx_clients_dui ON clients(dui);
CREATE INDEX idx_clients_ncr ON clients(ncr);
CREATE INDEX idx_clients_active ON clients(active);

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_clients_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_clients_updated_at
    BEFORE UPDATE ON clients
    FOR EACH ROW
    EXECUTE FUNCTION update_clients_updated_at();
-- Drop partial unique indexes
DROP INDEX IF EXISTS unique_client_nit_per_company;
DROP INDEX IF EXISTS unique_client_dui_per_company;

-- Remove check constraints
ALTER TABLE clients DROP CONSTRAINT IF EXISTS check_at_least_one_id;
ALTER TABLE clients DROP CONSTRAINT IF EXISTS check_nit_requires_ncr;

-- Recreate original unique constraints
ALTER TABLE clients ADD CONSTRAINT unique_client_nit_per_company UNIQUE (company_id, nit);
ALTER TABLE clients ADD CONSTRAINT unique_client_dui_per_company UNIQUE (company_id, dui);

-- Revert to NOT NULL (this will fail if there are NULL values in the database)
ALTER TABLE clients ALTER COLUMN ncr SET NOT NULL;
ALTER TABLE clients ALTER COLUMN nit SET NOT NULL;
ALTER TABLE clients ALTER COLUMN dui SET NOT NULL;
-- Make ncr, nit, and dui nullable since they're optional
ALTER TABLE clients ALTER COLUMN ncr DROP NOT NULL;
ALTER TABLE clients ALTER COLUMN nit DROP NOT NULL;
ALTER TABLE clients ALTER COLUMN dui DROP NOT NULL;

-- Drop existing unique constraints
ALTER TABLE clients DROP CONSTRAINT IF EXISTS unique_client_nit_per_company;
ALTER TABLE clients DROP CONSTRAINT IF EXISTS unique_client_dui_per_company;

-- Add check constraint to ensure at least one identification is provided
ALTER TABLE clients ADD CONSTRAINT check_at_least_one_id 
    CHECK (ncr IS NOT NULL OR nit IS NOT NULL OR dui IS NOT NULL);

-- Add check constraint to ensure if NIT is provided, NCR must also be provided
ALTER TABLE clients ADD CONSTRAINT check_nit_requires_ncr 
    CHECK ((nit IS NULL AND ncr IS NULL) OR (nit IS NOT NULL AND ncr IS NOT NULL));

-- Recreate unique constraints that work with NULL values
CREATE UNIQUE INDEX unique_client_nit_per_company ON clients(company_id, nit) WHERE nit IS NOT NULL;
CREATE UNIQUE INDEX unique_client_dui_per_company ON clients(company_id, dui) WHERE dui IS NOT NULL;
ALTER TABLE clients ALTER COLUMN municipality_code TYPE VARCHAR(4);

-- Remove comment
COMMENT ON COLUMN clients.municipality_code IS NULL;
-- Update municipality_code column to support dot notation format (DD.MM)
ALTER TABLE clients ALTER COLUMN municipality_code TYPE VARCHAR(5);

-- Add comment explaining the format
COMMENT ON COLUMN clients.municipality_code IS 'Municipality code in format DD.MM where DD is department code and MM is municipality code (e.g., 06.23 for San Salvador Centro)';
-- Remove tipo_persona column
ALTER TABLE clients DROP CONSTRAINT IF EXISTS check_valid_tipo_persona;
ALTER TABLE clients DROP COLUMN IF EXISTS tipo_persona;
-- Add tipo_persona column to clients table
ALTER TABLE clients ADD COLUMN tipo_persona VARCHAR(1) NOT NULL DEFAULT '1';

-- Add check constraint to ensure valid values
ALTER TABLE clients ADD CONSTRAINT check_valid_tipo_persona 
    CHECK (tipo_persona IN ('1', '2'));

-- Add comment
COMMENT ON COLUMN clients.tipo_persona IS 'Type of person: 1 = Natural Person (Individual), 2 = Juridical Person (Company/Legal Entity)';
DROP TRIGGER IF EXISTS trigger_update_inventory_items_updated_at ON inventory_items;
DROP FUNCTION IF EXISTS update_inventory_items_updated_at();
DROP TABLE IF EXISTS inventory_items;
CREATE TABLE IF NOT EXISTS inventory_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    
    tipo_item VARCHAR(1) NOT NULL,
    
    sku VARCHAR(100) NOT NULL,
    codigo_barras VARCHAR(100),
    
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    manufacturer VARCHAR(255),
    image_url TEXT,
    
    cost_price DECIMAL(15,2),
    unit_price DECIMAL(15,2) NOT NULL,
    unit_of_measure VARCHAR(50) NOT NULL,
    
    color VARCHAR(50),
    
    track_inventory BOOLEAN DEFAULT true,
    current_stock DECIMAL(15,3) DEFAULT 0,
    minimum_stock DECIMAL(15,3) DEFAULT 0,
    
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT unique_company_sku UNIQUE (company_id, sku),
    CONSTRAINT unique_company_barcode UNIQUE (company_id, codigo_barras),
    CONSTRAINT check_tipo_item_valid CHECK (tipo_item IN ('1', '2')),
    CONSTRAINT check_prices_positive CHECK (unit_price >= 0 AND (cost_price IS NULL OR cost_price >= 0))
);

CREATE INDEX idx_inventory_items_company_id ON inventory_items(company_id);
CREATE INDEX idx_inventory_items_sku ON inventory_items(company_id, sku);
CREATE INDEX idx_inventory_items_barcode ON inventory_items(company_id, codigo_barras) WHERE codigo_barras IS NOT NULL;
CREATE INDEX idx_inventory_items_active ON inventory_items(active);
CREATE INDEX idx_inventory_items_tipo ON inventory_items(tipo_item);

CREATE OR REPLACE FUNCTION update_inventory_items_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_inventory_items_updated_at
    BEFORE UPDATE ON inventory_items
    FOR EACH ROW
    EXECUTE FUNCTION update_inventory_items_updated_at();
CREATE TABLE IF NOT EXISTS inventory_item_taxes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    item_id UUID NOT NULL REFERENCES inventory_items(id) ON DELETE CASCADE,
    tributo_code VARCHAR(10) NOT NULL,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT unique_item_tributo UNIQUE (item_id, tributo_code)
);

CREATE INDEX idx_item_taxes_item_id ON inventory_item_taxes(item_id);
ALTER TABLE clients DROP CONSTRAINT IF EXISTS check_credit_status;
ALTER TABLE clients DROP COLUMN IF EXISTS credit_limit;
ALTER TABLE clients DROP COLUMN IF EXISTS current_balance;
ALTER TABLE clients DROP COLUMN IF EXISTS credit_status;
-- Add credit tracking fields to clients table
ALTER TABLE clients ADD COLUMN credit_limit DECIMAL(15,2) DEFAULT 0;
ALTER TABLE clients ADD COLUMN current_balance DECIMAL(15,2) DEFAULT 0;
ALTER TABLE clients ADD COLUMN credit_status VARCHAR(20) DEFAULT 'good_standing';

-- Add check constraint for credit status
ALTER TABLE clients ADD CONSTRAINT check_credit_status 
    CHECK (credit_status IN ('good_standing', 'over_limit', 'suspended'));

-- Add comments
COMMENT ON COLUMN clients.credit_limit IS 'Maximum credit amount allowed for this client';
COMMENT ON COLUMN clients.current_balance IS 'Current outstanding balance (total owed across all invoices)';
COMMENT ON COLUMN clients.credit_status IS 'Credit standing: good_standing, over_limit, suspended';
DROP TABLE IF EXISTS invoices;
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
    contact_email VARCHAR(255),
    contact_whatsapp VARCHAR(20),
    
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

DROP TABLE IF EXISTS invoice_line_items;
-- Create invoice_line_items table
CREATE TABLE IF NOT EXISTS invoice_line_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    line_number INT NOT NULL,
    
    -- Item reference (for tracking only, not for pricing)
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
    
    -- Constraints
    CONSTRAINT check_quantity_positive CHECK (quantity > 0),
    CONSTRAINT check_line_number_positive CHECK (line_number > 0),
    CONSTRAINT unique_invoice_line UNIQUE (invoice_id, line_number)
);

-- Indexes
CREATE INDEX idx_line_items_invoice ON invoice_line_items(invoice_id);
CREATE INDEX idx_line_items_item ON invoice_line_items(item_id);

-- Comments
COMMENT ON TABLE invoice_line_items IS 'Invoice line items with complete item snapshots';
COMMENT ON COLUMN invoice_line_items.item_id IS 'Reference to inventory item (for tracking, not pricing)';
COMMENT ON COLUMN invoice_line_items.unit_price IS 'Price at transaction time (snapshot)';
COMMENT ON COLUMN invoice_line_items.taxable_amount IS 'Amount after discount, used for tax calculation';
DROP TABLE IF EXISTS invoice_line_item_taxes;

CREATE TABLE IF NOT EXISTS invoice_line_item_taxes (
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
    
    -- Constraints
    CONSTRAINT check_tax_rate_valid CHECK (tax_rate >= 0 AND tax_rate <= 1)
);

-- Indexes
CREATE INDEX idx_line_item_taxes_line ON invoice_line_item_taxes(line_item_id);
CREATE INDEX idx_line_item_taxes_code ON invoice_line_item_taxes(tributo_code);

-- Comments
COMMENT ON TABLE invoice_line_item_taxes IS 'Tax details per line item with rate snapshots';
COMMENT ON COLUMN invoice_line_item_taxes.tax_rate IS 'Tax rate at transaction time as decimal (e.g., 0.13 for 13%)';
COMMENT ON COLUMN invoice_line_item_taxes.taxable_base IS 'Amount this tax was calculated on';
COMMENT ON COLUMN invoice_line_item_taxes.tax_amount IS 'Calculated tax amount';
DROP TABLE IF EXISTS invoice_payments;
-- Create invoice_payments table
CREATE TABLE IF NOT EXISTS invoice_payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES invoices(id),
    
    payment_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    payment_method VARCHAR(50) NOT NULL,
    amount DECIMAL(15,2) NOT NULL,
    
    reference_number VARCHAR(100),
    notes TEXT,
    
    created_by UUID,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT check_payment_amount_positive CHECK (amount > 0),
    CONSTRAINT check_payment_method CHECK (payment_method IN ('cash', 'card', 'transfer', 'check', 'other'))
);

-- Indexes
CREATE INDEX idx_payments_invoice ON invoice_payments(invoice_id);
CREATE INDEX idx_payments_date ON invoice_payments(payment_date);

-- Comments
COMMENT ON TABLE invoice_payments IS 'Payment records for invoices';
COMMENT ON COLUMN invoice_payments.payment_method IS 'Payment method: cash, card, transfer, check, other';
COMMENT ON COLUMN invoice_payments.reference_number IS 'Check number, transfer ID, transaction reference, etc.';
DROP TRIGGER IF EXISTS trigger_create_consumidor_final ON companies;
DROP FUNCTION IF EXISTS create_consumidor_final_for_company();
DELETE FROM clients WHERE nit = 9999999999999;
-- Create consumidor final client for each existing company
INSERT INTO clients (
    company_id,
    nit,
    ncr,
    business_name,
    legal_business_name,
    giro,
    tipo_contribuyente,
    tipo_persona,
    full_address,
    country_code,
    department_code,
    municipality_code,
    active
)
SELECT 
    id as company_id,
    '9999999999999' as nit,
    '999999' as ncr,  -- Added NCR
    'Consumidor Final' as business_name,
    'Consumidor Final' as legal_business_name,
    'Consumidor Final' as giro,
    'Consumidor Final' as tipo_contribuyente,
    '1' as tipo_persona,
    'N/A' as full_address,
    'SV' as country_code,
    '06' as department_code,
    '01' as municipality_code,
    true as active
FROM companies
ON CONFLICT DO NOTHING;

-- Create trigger to automatically create consumidor final for new companies
CREATE OR REPLACE FUNCTION create_consumidor_final_for_company()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO clients (
        company_id,
        nit,
        ncr,
        business_name,
        legal_business_name,
        giro,
        tipo_contribuyente,
        tipo_persona,
        full_address,
        country_code,
        department_code,
        municipality_code,
        active
    ) VALUES (
        NEW.id,
        '9999999999999',
        '999999',  -- Added NCR
        'Consumidor Final',
        'Consumidor Final',
        'Consumidor Final',
        'Consumidor Final',
        '1',
        'N/A',
        'SV',
        '06',
        '01',
        true
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_create_consumidor_final
    AFTER INSERT ON companies
    FOR EACH ROW
    EXECUTE FUNCTION create_consumidor_final_for_company();

COMMENT ON FUNCTION create_consumidor_final_for_company IS 'Automatically creates a Consumidor Final client for each new company';
DROP TABLE IF EXISTS codigos.tributos;
DROP SCHEMA IF EXISTS codigos CASCADE;
-- Create codigos schema
CREATE SCHEMA IF NOT EXISTS codigos;

-- Create tributos table
CREATE TABLE IF NOT EXISTS codigos.tributos (
    codigo VARCHAR(10) PRIMARY KEY,
    descripcion VARCHAR(255) NOT NULL,
    porcentaje DECIMAL(8,6) NOT NULL,
    tipo VARCHAR(50)
);

CREATE INDEX idx_tributos_codigo ON codigos.tributos(codigo);

COMMENT ON TABLE codigos.tributos IS 'Catalog of tax codes (tributos) from Ministerio de Hacienda';
COMMENT ON COLUMN codigos.tributos.porcentaje IS 'Tax percentage (e.g., 13.00 for 13%)';
CREATE TABLE establishments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    tipo_establecimiento VARCHAR(2) NOT NULL,
    nombre VARCHAR(100) NOT NULL,
    cod_establecimiento_mh VARCHAR(4), -- From MH, nullable until assigned
    cod_establecimiento VARCHAR(10),   -- Company's internal code
    -- Address for DTE
    departamento VARCHAR(2) NOT NULL,
    municipio VARCHAR(2) NOT NULL,
    complemento_direccion TEXT NOT NULL,
    telefono VARCHAR(30) NOT NULL,
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_establishments_company ON establishments(company_id);
CREATE INDEX idx_establishments_active ON establishments(company_id, active);
CREATE TABLE point_of_sale (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    establishment_id UUID NOT NULL REFERENCES establishments(id) ON DELETE CASCADE,
    nombre VARCHAR(100) NOT NULL,
    cod_punto_venta_mh VARCHAR(4),  -- From MH, nullable until assigned
    cod_punto_venta VARCHAR(15),    -- Company's internal code
    latitude DECIMAL(10, 8),        -- GPS latitude for portable POS
    longitude DECIMAL(11, 8),       -- GPS longitude for portable POS
    is_portable BOOLEAN DEFAULT false,
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_pos_establishment ON point_of_sale(establishment_id);
CREATE INDEX idx_pos_active ON point_of_sale(establishment_id, active);
CREATE TABLE dte_sequences (
    point_of_sale_id UUID NOT NULL REFERENCES point_of_sale(id) ON DELETE CASCADE,
    tipo_dte VARCHAR(2) NOT NULL,
    last_sequence BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (point_of_sale_id, tipo_dte)
);
ALTER TABLE invoices 
ADD COLUMN point_of_sale_id UUID REFERENCES point_of_sale(id);

-- For existing invoices, we'll need to handle this separately
-- For now, allow NULL, then later make NOT NULL after migration
DROP INDEX IF EXISTS idx_invoices_pos;
DROP INDEX IF EXISTS idx_invoices_establishment;

ALTER TABLE invoices 
DROP COLUMN IF EXISTS point_of_sale_id,
DROP COLUMN IF EXISTS establishment_id;
ALTER TABLE invoices 
ADD COLUMN establishment_id UUID REFERENCES establishments(id),
ADD COLUMN point_of_sale_id UUID REFERENCES point_of_sale(id);

-- Index for queries by establishment
CREATE INDEX idx_invoices_establishment ON invoices(establishment_id);

-- Index for queries by POS
CREATE INDEX idx_invoices_pos ON invoices(point_of_sale_id);
ALTER TABLE invoices DROP CONSTRAINT IF EXISTS check_payment_method;
ALTER TABLE invoices DROP COLUMN IF EXISTS payment_method;
ALTER TABLE invoices ADD COLUMN payment_method VARCHAR(2);

-- Add check constraint
ALTER TABLE invoices ADD CONSTRAINT check_payment_method 
CHECK (payment_method IS NULL OR payment_method IN (
    '01', '02', '03', '04', '05', '08', '09', '11', '12', '13', '14', '99'
));
-- Remove DTE-related fields from companies table
ALTER TABLE companies 
DROP COLUMN IF EXISTS cod_actividad,
DROP COLUMN IF EXISTS desc_actividad,
DROP COLUMN IF EXISTS nombre_comercial,
DROP COLUMN IF EXISTS dte_ambiente,
DROP COLUMN IF EXISTS firmador_username,
DROP COLUMN IF EXISTS firmador_password_ref;
-- Add DTE-related fields to companies table
ALTER TABLE companies 
ADD COLUMN cod_actividad VARCHAR(6),
ADD COLUMN desc_actividad VARCHAR(150),
ADD COLUMN nombre_comercial VARCHAR(150),
ADD COLUMN dte_ambiente VARCHAR(2) DEFAULT '00',
ADD COLUMN firmador_username VARCHAR(100),
ADD COLUMN firmador_password_ref VARCHAR(255);

-- Add comment explaining dte_ambiente values
COMMENT ON COLUMN companies.dte_ambiente IS '00 = test/sandbox, 01 = production';
DROP TABLE IF EXISTS dte_submission_attempts;
-- Create DTE submission attempts table for audit trail
CREATE TABLE dte_submission_attempts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE RESTRICT,
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE RESTRICT,
    
    -- Attempt tracking
    attempt_number INT NOT NULL,
    attempt_source VARCHAR(20) NOT NULL, -- 'finalization' or 'background_retry'
    
    -- Precise timestamps for audit compliance
    attempted_at_epoch BIGINT NOT NULL,  -- Unix timestamp in seconds
    attempted_at TIMESTAMP NOT NULL,      -- Human-readable timestamp
    duration_ms INT,                      -- Request duration in milliseconds
    
    -- Outcome tracking
    outcome VARCHAR(30) NOT NULL,         -- 'success', 'network_error', 'validation_error', 'hacienda_rejected', 'timeout', 'signing_error'
    http_status INT,                      -- HTTP status code from response
    
    -- Complete audit trail
    request_body TEXT NOT NULL,           -- Full DTE JSON sent
    response_body TEXT,                   -- Full response received
    error_message TEXT,                   -- Human-readable error description
    
    -- Context
    hacienda_endpoint VARCHAR(255) NOT NULL,  -- URL used (test vs production)
    
    created_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for fast queries and reporting
CREATE INDEX idx_dte_attempts_company ON dte_submission_attempts(company_id);
CREATE INDEX idx_dte_attempts_invoice ON dte_submission_attempts(invoice_id);
CREATE INDEX idx_dte_attempts_outcome ON dte_submission_attempts(outcome);
CREATE INDEX idx_dte_attempts_epoch ON dte_submission_attempts(attempted_at_epoch);
CREATE INDEX idx_dte_attempts_source ON dte_submission_attempts(attempt_source);
CREATE INDEX idx_dte_attempts_created ON dte_submission_attempts(created_at);

-- Add comments for documentation
COMMENT ON TABLE dte_submission_attempts IS 'Audit log of all DTE submission attempts to Hacienda';
COMMENT ON COLUMN dte_submission_attempts.attempted_at_epoch IS 'Unix timestamp in seconds for precise audit trail';
COMMENT ON COLUMN dte_submission_attempts.outcome IS 'Result: success, network_error, validation_error, hacienda_rejected, timeout, signing_error';
COMMENT ON COLUMN dte_submission_attempts.attempt_source IS 'Source: finalization (during invoice finalization) or background_retry (scheduled retry)';
-- Drop payments table
DROP TABLE IF EXISTS payments;
-- Create payments table to track all payments against invoices
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE RESTRICT,
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE RESTRICT,
    
    -- Payment details
    amount DECIMAL(15, 2) NOT NULL CHECK (amount > 0),
    payment_method VARCHAR(2) NOT NULL,   -- cat_012 formas de pago code
    payment_reference VARCHAR(100),        -- Optional reference (check number, transaction ID, etc.)
    
    -- Timestamps
    payment_date TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW(),
    created_by UUID,  -- Future: user who recorded payment
    
    -- Audit
    notes TEXT
);

-- Indexes for fast queries
CREATE INDEX idx_payments_company ON payments(company_id);
CREATE INDEX idx_payments_method ON payments(payment_method);

-- Add comments for documentation
COMMENT ON TABLE payments IS 'Record of all payments received against invoices';
COMMENT ON COLUMN payments.payment_method IS 'Payment method code from cat_012 (formas de pago)';
COMMENT ON COLUMN payments.amount IS 'Payment amount - must be positive';
COMMENT ON COLUMN payments.payment_reference IS 'Optional reference like check number, transaction ID, or receipt number';
