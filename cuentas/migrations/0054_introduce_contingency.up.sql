-- ============================================================================
-- CORRECTED SCHEMA - Matches your existing Go models exactly
-- ============================================================================

-- Main contingency queue
CREATE TABLE dte_contingency_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- What failed
    invoice_id VARCHAR(36) REFERENCES invoices(id),
    purchase_id UUID REFERENCES purchases(id),
    tipo_dte VARCHAR(2) NOT NULL,
    codigo_generacion VARCHAR(36) NOT NULL UNIQUE,
    ambiente VARCHAR(2) NOT NULL,
    
    -- Failure tracking
    failure_stage VARCHAR(20) NOT NULL,
    failure_reason TEXT NOT NULL,
    failure_timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- DTE data
    dte_unsigned JSONB NOT NULL,
    dte_signed TEXT,
    
    -- Contingency flow tracking
    contingency_event_id UUID REFERENCES dte_contingency_events(id),
    batch_id UUID REFERENCES dte_contingency_batches(id),
    
    -- Status tracking
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 3,
    
    -- Results
    sello_recibido TEXT,
    hacienda_response JSONB,
    hacienda_observations TEXT[],
    
    -- Timing
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    
    -- Metadata
    company_id UUID NOT NULL,
    created_by UUID,
    
    CONSTRAINT one_document_type CHECK (
        (invoice_id IS NOT NULL AND purchase_id IS NULL) OR
        (invoice_id IS NULL AND purchase_id IS NOT NULL)
    )
);

-- Contingency events (Reporte de Contingencia)
CREATE TABLE dte_contingency_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Event identification
    codigo_generacion VARCHAR(36) NOT NULL UNIQUE,
    company_id UUID NOT NULL,
    ambiente VARCHAR(2) NOT NULL,
    
    -- Contingency period
    fecha_inicio TIMESTAMPTZ NOT NULL,
    fecha_fin TIMESTAMPTZ NOT NULL,
    -- CHANGE: TIME -> TIMESTAMPTZ to match Go time.Time
    hora_inicio TIMESTAMPTZ NOT NULL,  -- NEW: was TIME, now TIMESTAMPTZ
    hora_fin TIMESTAMPTZ NOT NULL,     -- NEW: was TIME, now TIMESTAMPTZ
    
    -- Contingency reason
    tipo_contingencia INT NOT NULL,
    motivo_contingencia TEXT,
    
    -- Event data
    event_unsigned JSONB NOT NULL,
    event_signed TEXT,
    
    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    
    -- Hacienda response
    sello_recibido TEXT,
    hacienda_response JSONB,
    
    -- Timing
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    submitted_at TIMESTAMPTZ,
    accepted_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),  -- NEW: added updated_at
    
    -- Metadata
    created_by UUID,
    dte_count INT DEFAULT 0
);

-- Contingency batches (for batch submission)
CREATE TABLE dte_contingency_batches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Links to event
    contingency_event_id UUID NOT NULL REFERENCES dte_contingency_events(id),
    
    -- Batch info
    codigo_lote VARCHAR(36),
    company_id UUID NOT NULL,
    ambiente VARCHAR(2) NOT NULL,
    
    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    
    -- Batch stats
    total_dtes INT NOT NULL DEFAULT 0,
    processed_count INT DEFAULT 0,
    rejected_count INT DEFAULT 0,
    
    -- Hacienda response
    hacienda_response JSONB,
    
    -- Timing
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    submitted_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),  -- NEW: added updated_at
    
    -- Retry tracking
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 3,
    next_retry_at TIMESTAMPTZ
);

-- Indexes
CREATE INDEX idx_contingency_queue_status ON dte_contingency_queue(status);
CREATE INDEX idx_contingency_queue_company ON dte_contingency_queue(company_id, status);
CREATE INDEX idx_contingency_queue_event ON dte_contingency_queue(contingency_event_id);
CREATE INDEX idx_contingency_queue_batch ON dte_contingency_queue(batch_id);
CREATE INDEX idx_contingency_queue_created ON dte_contingency_queue(created_at);
CREATE INDEX idx_contingency_queue_pending ON dte_contingency_queue(company_id, status) WHERE status = 'pending';  -- NEW: added index

CREATE INDEX idx_contingency_events_company ON dte_contingency_events(company_id);
CREATE INDEX idx_contingency_events_status ON dte_contingency_events(status);
CREATE INDEX idx_contingency_events_period ON dte_contingency_events(fecha_inicio, fecha_fin);
CREATE INDEX idx_contingency_events_accepted ON dte_contingency_events(status, accepted_at) WHERE status = 'accepted';  -- NEW: added index

CREATE INDEX idx_contingency_batches_event ON dte_contingency_batches(contingency_event_id);
CREATE INDEX idx_contingency_batches_status ON dte_contingency_batches(status);
CREATE INDEX idx_contingency_batches_retry ON dte_contingency_batches(next_retry_at) WHERE status IN ('pending', 'failed');
CREATE INDEX idx_contingency_batches_processing ON dte_contingency_batches(status, submitted_at) WHERE status IN ('submitted', 'processing');  -- NEW: added index

-- Update trigger
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_contingency_queue_updated_at BEFORE UPDATE ON dte_contingency_queue
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- NEW: Added triggers for other tables
CREATE TRIGGER update_contingency_events_updated_at BEFORE UPDATE ON dte_contingency_events
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_contingency_batches_updated_at BEFORE UPDATE ON dte_contingency_batches
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
