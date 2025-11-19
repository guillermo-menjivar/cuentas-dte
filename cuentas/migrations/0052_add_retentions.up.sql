-- Add retention agent configuration to companies
ALTER TABLE companies 
ADD COLUMN IF NOT EXISTS is_retention_agent BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS retention_rate NUMERIC(5,2) DEFAULT 1.00;

COMMENT ON COLUMN companies.is_retention_agent IS 'Whether company is designated as IVA retention agent (Agente de Retención)';
COMMENT ON COLUMN companies.retention_rate IS 'IVA retention rate: 1.00 (Art 162), 2.00 (special), 13.00 (government)';

-- Create retentions table for DTE 07 (Comprobante de Retención)
CREATE TABLE retentions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- References
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    purchase_id UUID NOT NULL REFERENCES purchases(id) ON DELETE CASCADE,
    establishment_id UUID NOT NULL REFERENCES establishments(id),
    point_of_sale_id UUID NOT NULL REFERENCES point_of_sale(id),
    
    -- Supplier info (copied from purchase at time of retention)
    -- This is a snapshot - if supplier info changes in purchase, retention keeps original
    supplier_id UUID REFERENCES clients(id), -- May be NULL for FSE
    supplier_name TEXT NOT NULL,
    supplier_nit TEXT, -- Required for formal suppliers
    supplier_nrc TEXT, -- Required for formal suppliers
    
    -- DTE identifiers
    codigo_generacion VARCHAR(36) UNIQUE NOT NULL,
    numero_control VARCHAR(50) NOT NULL,
    tipo_dte VARCHAR(2) NOT NULL DEFAULT '07',
    ambiente VARCHAR(2) NOT NULL,
    
    -- Purchase/Invoice reference
    purchase_numero_control VARCHAR(50) NOT NULL,
    purchase_codigo_generacion VARCHAR(36) NOT NULL,
    purchase_tipo_dte VARCHAR(2) NOT NULL,
    purchase_fecha_emision DATE NOT NULL,
    
    -- Retention amounts
    monto_sujeto_grav NUMERIC(15,2) NOT NULL,
    iva_retenido NUMERIC(15,2) NOT NULL,
    retention_rate NUMERIC(5,2) NOT NULL,
    retention_code VARCHAR(2) NOT NULL,
    
    -- Dates
    fecha_emision DATE NOT NULL,
    fecha_procesamiento TIMESTAMPTZ,
    
    -- DTE data
    dte_json JSONB NOT NULL,
    dte_signed TEXT NOT NULL,
    
    -- Hacienda response
    hacienda_estado VARCHAR(20),
    hacienda_sello_recibido VARCHAR(100),
    hacienda_fh_procesamiento TIMESTAMPTZ,
    hacienda_codigo_msg VARCHAR(10),
    hacienda_descripcion_msg TEXT,
    hacienda_observaciones TEXT[],
    hacienda_response JSONB,
    
    -- Audit
    created_by UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    submitted_at TIMESTAMPTZ,
    
    CONSTRAINT retentions_retention_rate_check CHECK (retention_rate IN (1.00, 2.00, 13.00)),
    CONSTRAINT retentions_retention_code_check CHECK (retention_code IN ('22', 'C4', 'C9')),
    -- Formal suppliers must have NIT+NRC for retention
    CONSTRAINT retentions_formal_supplier_check CHECK (
        supplier_nit IS NOT NULL AND supplier_nrc IS NOT NULL
    )
);

-- Indexes for retentions
CREATE INDEX idx_retentions_company ON retentions(company_id);
CREATE INDEX idx_retentions_purchase ON retentions(purchase_id);
CREATE INDEX idx_retentions_supplier ON retentions(supplier_id) WHERE supplier_id IS NOT NULL;
CREATE INDEX idx_retentions_codigo_gen ON retentions(codigo_generacion);
CREATE INDEX idx_retentions_estado ON retentions(hacienda_estado);
CREATE INDEX idx_retentions_fecha_emision ON retentions(fecha_emision);
CREATE INDEX idx_retentions_submitted ON retentions(submitted_at);
CREATE INDEX idx_retentions_supplier_nit ON retentions(supplier_nit) WHERE supplier_nit IS NOT NULL;

-- Add retention reference to purchases
ALTER TABLE purchases 
ADD COLUMN IF NOT EXISTS has_retention BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS retention_id UUID REFERENCES retentions(id);

CREATE INDEX idx_purchases_retention ON purchases(retention_id) WHERE retention_id IS NOT NULL;

-- Comments
COMMENT ON TABLE retentions IS 'DTE 07 - Comprobante de Retención (IVA Retention Certificates)';
COMMENT ON COLUMN retentions.supplier_nit IS 'Supplier NIT - required for retention (must be formal IVA contributor)';
COMMENT ON COLUMN retentions.supplier_nrc IS 'Supplier NRC - required for retention';
COMMENT ON COLUMN retentions.monto_sujeto_grav IS 'Taxable amount subject to retention';
COMMENT ON COLUMN retentions.iva_retenido IS 'Amount of IVA retained from supplier';
COMMENT ON COLUMN retentions.retention_code IS 'MH code: 22=1%, C4=2%, C9=13%';
