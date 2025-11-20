-- Migration: Create DTE Storage Index Table
-- Description: Tracks DTE storage locations in S3 for fast retrieval and bulk downloads
-- Date: 2025-01-15

-- ============================================================================
-- Table: dte_storage_index
-- Purpose: Index of all DTEs stored in S3 with paths and metadata for fast lookups
-- ============================================================================

CREATE TABLE IF NOT EXISTS dte_storage_index (
    -- Primary identification
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    
    -- DTE identification (from Hacienda)
    generation_code VARCHAR(36) NOT NULL UNIQUE,
    control_number VARCHAR(50) NOT NULL,
    document_type VARCHAR(2) NOT NULL,
    
    -- Timestamps
    issue_date TIMESTAMP NOT NULL,
    processing_date TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Sender info for quick filtering
    sender_nit VARCHAR(17),
    sender_nrc VARCHAR(10),
    sender_name VARCHAR(250),
    
    -- Receiver info for quick filtering
    receiver_nit VARCHAR(17),
    receiver_nrc VARCHAR(10),
    receiver_name VARCHAR(250),
    receiver_document_type VARCHAR(20),  -- NIT, DUI, etc.
    receiver_document_number VARCHAR(25),
    
    -- Financial summary
    total_amount DECIMAL(18,2),
    taxable_amount DECIMAL(18,2),
    exempt_amount DECIMAL(18,2),
    tax_amount DECIMAL(18,2),
    currency VARCHAR(3) DEFAULT 'USD',
    
    -- DTE status
    status VARCHAR(20) NOT NULL,  -- GENERATED, PROCESSED, REJECTED, etc.
    received_stamp VARCHAR(500),
    observations TEXT,
    
    -- S3 storage paths
    s3_download_path VARCHAR(500) NOT NULL,
    s3_analytics_path VARCHAR(500) NOT NULL,
    s3_bucket VARCHAR(100) NOT NULL,
    s3_region VARCHAR(20) NOT NULL DEFAULT 'us-east-1',
    
    -- Partition information (for analytics path construction)
    partition_year INT NOT NULL,
    partition_month INT NOT NULL,
    partition_day INT NOT NULL,
    
    -- File metadata
    file_size_bytes BIGINT,
    file_checksum VARCHAR(64),  -- SHA256 hash
    
    -- Soft delete
    deleted_at TIMESTAMP,
    
    -- Constraints
    CONSTRAINT ck_document_type CHECK (document_type IN ('01', '03', '04', '05', '06', '07', '08', '09', '11', '14', '15')),
    CONSTRAINT ck_status CHECK (status IN ('GENERATED', 'PROCESSED', 'REJECTED', 'CONTINGENCY', 'INVALIDATED')),
    CONSTRAINT ck_partition_month CHECK (partition_month BETWEEN 1 AND 12),
    CONSTRAINT ck_partition_day CHECK (partition_day BETWEEN 1 AND 31),
    CONSTRAINT ck_partition_year CHECK (partition_year >= 2024),
    CONSTRAINT ck_currency CHECK (currency IN ('USD', 'SVC'))
);

-- ============================================================================
-- Indexes for performance
-- ============================================================================

-- Primary query patterns
CREATE INDEX idx_dte_storage_company_date ON dte_storage_index(company_id, issue_date DESC);
CREATE INDEX idx_dte_storage_company_type_date ON dte_storage_index(company_id, document_type, issue_date DESC);
CREATE INDEX idx_dte_storage_generation_code ON dte_storage_index(generation_code) WHERE deleted_at IS NULL;
CREATE INDEX idx_dte_storage_control_number ON dte_storage_index(control_number) WHERE deleted_at IS NULL;

-- Filtering indexes
CREATE INDEX idx_dte_storage_sender_nit ON dte_storage_index(sender_nit) WHERE deleted_at IS NULL;
CREATE INDEX idx_dte_storage_receiver_nit ON dte_storage_index(receiver_nit) WHERE deleted_at IS NULL;
CREATE INDEX idx_dte_storage_status ON dte_storage_index(company_id, status) WHERE deleted_at IS NULL;

-- Analytics partition lookup
CREATE INDEX idx_dte_storage_partitions ON dte_storage_index(company_id, document_type, partition_year, partition_month, partition_day);

-- Soft delete support
CREATE INDEX idx_dte_storage_deleted ON dte_storage_index(deleted_at) WHERE deleted_at IS NOT NULL;

-- Full-text search on receiver name (optional, requires pg_trgm extension)
-- CREATE EXTENSION IF NOT EXISTS pg_trgm;
-- CREATE INDEX idx_dte_storage_receiver_name_trgm ON dte_storage_index USING gin(receiver_name gin_trgm_ops);

-- ============================================================================
-- Trigger: Update updated_at timestamp
-- ============================================================================

CREATE OR REPLACE FUNCTION update_dte_storage_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_dte_storage_updated_at
    BEFORE UPDATE ON dte_storage_index
    FOR EACH ROW
    EXECUTE FUNCTION update_dte_storage_updated_at();

-- ============================================================================
-- Function: Get S3 download path
-- ============================================================================

CREATE OR REPLACE FUNCTION get_dte_download_path(
    p_company_id UUID,
    p_generation_code VARCHAR
)
RETURNS VARCHAR AS $$
DECLARE
    v_path VARCHAR;
BEGIN
    SELECT s3_download_path INTO v_path
    FROM dte_storage_index
    WHERE company_id = p_company_id
      AND generation_code = p_generation_code
      AND deleted_at IS NULL;
    
    RETURN v_path;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- Function: Get DTEs by date range
-- ============================================================================

CREATE OR REPLACE FUNCTION get_dtes_by_date_range(
    p_company_id UUID,
    p_start_date DATE,
    p_end_date DATE,
    p_document_type VARCHAR DEFAULT NULL
)
RETURNS TABLE (
    generation_code VARCHAR,
    control_number VARCHAR,
    document_type VARCHAR,
    issue_date TIMESTAMP,
    receiver_name VARCHAR,
    total_amount DECIMAL,
    status VARCHAR,
    s3_download_path VARCHAR
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        dsi.generation_code,
        dsi.control_number,
        dsi.document_type,
        dsi.issue_date,
        dsi.receiver_name,
        dsi.total_amount,
        dsi.status,
        dsi.s3_download_path
    FROM dte_storage_index dsi
    WHERE dsi.company_id = p_company_id
      AND dsi.issue_date >= p_start_date
      AND dsi.issue_date < p_end_date + INTERVAL '1 day'
      AND dsi.deleted_at IS NULL
      AND (p_document_type IS NULL OR dsi.document_type = p_document_type)
    ORDER BY dsi.issue_date DESC;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- Function: Get analytics path components
-- ============================================================================

CREATE OR REPLACE FUNCTION get_analytics_path_components(
    p_company_id UUID,
    p_document_type VARCHAR,
    p_year INT,
    p_month INT,
    p_day INT DEFAULT NULL
)
RETURNS TABLE (
    generation_code VARCHAR,
    s3_analytics_path VARCHAR
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        dsi.generation_code,
        dsi.s3_analytics_path
    FROM dte_storage_index dsi
    WHERE dsi.company_id = p_company_id
      AND dsi.document_type = p_document_type
      AND dsi.partition_year = p_year
      AND dsi.partition_month = p_month
      AND (p_day IS NULL OR dsi.partition_day = p_day)
      AND dsi.deleted_at IS NULL
    ORDER BY dsi.issue_date DESC;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- View: DTE Summary Statistics
-- ============================================================================

CREATE OR REPLACE VIEW v_dte_storage_stats AS
SELECT 
    company_id,
    document_type,
    COUNT(*) as total_dtes,
    SUM(total_amount) as total_amount,
    MIN(issue_date) as first_date,
    MAX(issue_date) as last_date,
    COUNT(DISTINCT DATE(issue_date)) as days_with_activity,
    SUM(file_size_bytes) as total_storage_bytes,
    ROUND(AVG(file_size_bytes), 0) as avg_file_size_bytes
FROM dte_storage_index
WHERE deleted_at IS NULL
GROUP BY company_id, document_type;

-- ============================================================================
-- View: Recent DTEs
-- ============================================================================

CREATE OR REPLACE VIEW v_dte_storage_recent AS
SELECT 
    dsi.id,
    dsi.company_id,
    c.business_name as company_name,
    dsi.generation_code,
    dsi.control_number,
    dsi.document_type,
    CASE dsi.document_type
        WHEN '01' THEN 'Invoice'
        WHEN '03' THEN 'Tax Credit'
        WHEN '04' THEN 'Delivery Note'
        WHEN '05' THEN 'Credit Note'
        WHEN '06' THEN 'Debit Note'
        WHEN '07' THEN 'Withholding'
        WHEN '11' THEN 'Export Invoice'
        WHEN '14' THEN 'Excluded Subject Invoice'
        WHEN '15' THEN 'Donation Receipt'
        ELSE dsi.document_type
    END as document_type_name,
    dsi.issue_date,
    dsi.receiver_name,
    dsi.total_amount,
    dsi.status,
    dsi.s3_download_path,
    dsi.created_at
FROM dte_storage_index dsi
LEFT JOIN companies c ON dsi.company_id = c.id
WHERE dsi.deleted_at IS NULL
  AND dsi.issue_date >= NOW() - INTERVAL '30 days'
ORDER BY dsi.issue_date DESC;

-- ============================================================================
-- Grant permissions (adjust as needed for your roles)
-- ============================================================================

-- GRANT SELECT, INSERT, UPDATE ON dte_storage_index TO app_user;
-- GRANT SELECT ON v_dte_storage_stats TO app_user;
-- GRANT SELECT ON v_dte_storage_recent TO app_user;
-- GRANT EXECUTE ON FUNCTION get_dte_download_path TO app_user;
-- GRANT EXECUTE ON FUNCTION get_dtes_by_date_range TO app_user;
-- GRANT EXECUTE ON FUNCTION get_analytics_path_components TO app_user;

-- ============================================================================
-- Comments for documentation
-- ============================================================================

COMMENT ON TABLE dte_storage_index IS 'Index of all DTEs stored in S3 with metadata for fast retrieval';
COMMENT ON COLUMN dte_storage_index.generation_code IS 'Unique DTE identifier from Hacienda';
COMMENT ON COLUMN dte_storage_index.control_number IS 'Control number (DTE-XX-XXXXXXXX-XXXXXXXXXXXXXXX)';
COMMENT ON COLUMN dte_storage_index.s3_download_path IS 'S3 path for direct download: dtes/{company_id}/{generation_code}/document.json';
COMMENT ON COLUMN dte_storage_index.s3_analytics_path IS 'S3 path for analytics: analytics/company_id={}/document_type={}/year={}/month={}/day={}/*.json';
COMMENT ON COLUMN dte_storage_index.partition_year IS 'Year partition for analytics path';
COMMENT ON COLUMN dte_storage_index.partition_month IS 'Month partition for analytics path (1-12)';
COMMENT ON COLUMN dte_storage_index.partition_day IS 'Day partition for analytics path (1-31)';

-- ============================================================================
-- Sample queries for testing
-- ============================================================================

-- Find DTE by generation_code
-- SELECT * FROM dte_storage_index WHERE generation_code = 'A1B2C3D4-E5F6-7890-ABCD-EF1234567890';

-- Get all DTEs for a company in January 2025
-- SELECT * FROM get_dtes_by_date_range('123e4567-e89b-12d3-a456-426614174000', '2025-01-01', '2025-01-31');

-- Get invoices only for a date range
-- SELECT * FROM get_dtes_by_date_range('123e4567-e89b-12d3-a456-426614174000', '2025-01-01', '2025-01-31', '01');

-- Get statistics per company
-- SELECT * FROM v_dte_storage_stats WHERE company_id = '123e4567-e89b-12d3-a456-426614174000';

-- Get recent DTEs
-- SELECT * FROM v_dte_storage_recent LIMIT 100;

-- Find DTEs by client
-- SELECT * FROM dte_storage_index WHERE receiver_nit = '06140000000001' AND deleted_at IS NULL;

-- Storage usage by company
-- SELECT 
--     company_id,
--     COUNT(*) as total_dtes,
--     SUM(file_size_bytes) / 1024.0 / 1024.0 as total_mb,
--     AVG(file_size_bytes) / 1024.0 as avg_kb
-- FROM dte_storage_index
-- WHERE deleted_at IS NULL
-- GROUP BY company_id;
