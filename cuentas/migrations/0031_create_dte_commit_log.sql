-- migrations/0031_create_dte_commit_log.sql

CREATE TABLE dte_commit_log (
    -- Primary key is the codigo_generacion (same as invoice.id)
    codigo_generacion VARCHAR(36) PRIMARY KEY,
    
    -- Invoice reference
    invoice_id VARCHAR(36) NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    
    -- DTE identification
    numero_control VARCHAR(50) NOT NULL,
    tipo_dte VARCHAR(2) NOT NULL,
    ambiente VARCHAR(2) NOT NULL,
    fecha_emision DATE NOT NULL,
    
    -- Public receipt URL
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
    
    -- Timestamps
    submitted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_dte_commit_log_invoice ON dte_commit_log(invoice_id);
CREATE INDEX idx_dte_commit_log_company ON dte_commit_log(company_id);
CREATE INDEX idx_dte_commit_log_estado ON dte_commit_log(hacienda_estado);
CREATE INDEX idx_dte_commit_log_submitted ON dte_commit_log(submitted_at);

-- Comments
COMMENT ON TABLE dte_commit_log IS 'Immutable audit trail of all DTE submissions to Hacienda';
COMMENT ON COLUMN dte_commit_log.codigo_generacion IS 'Código de Generación (uppercase UUID) - primary key matching invoice.id';
COMMENT ON COLUMN dte_commit_log.receipt_url IS 'Public URL to view the DTE receipt on Hacienda portal';
COMMENT ON COLUMN dte_commit_log.dte_unsigned IS 'Original unsigned DTE JSON structure before signing';
COMMENT ON COLUMN dte_commit_log.dte_signed IS 'Signed JWT from Firmador service';
COMMENT ON COLUMN dte_commit_log.hacienda_response_full IS 'Complete response JSON from Hacienda API';
