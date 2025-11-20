CREATE TABLE IF NOT EXISTS dte_storage_index (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- DTE identification
    company_id UUID NOT NULL,
    generation_code VARCHAR(36) NOT NULL UNIQUE,
    document_type VARCHAR(2) NOT NULL,
    
    -- S3 paths
    s3_unsigned_path VARCHAR(500) NOT NULL,
    s3_signed_path VARCHAR(500) NOT NULL,
    s3_analytics_path VARCHAR(500) NOT NULL,
    
    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Indexes
    CONSTRAINT idx_generation_code UNIQUE (generation_code)
);

CREATE INDEX idx_dte_storage_company ON dte_storage_index(company_id);
CREATE INDEX idx_dte_storage_type ON dte_storage_index(document_type);
