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
