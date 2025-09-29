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
