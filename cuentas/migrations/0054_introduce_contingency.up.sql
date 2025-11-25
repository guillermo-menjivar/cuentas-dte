-- ============================================================================
-- Migration 0054: Contingency System - New Design
-- Description: Creates contingency tracking for POS offline and service failures
-- ============================================================================

-- 1. Create contingency_periods (no dependencies)
CREATE TABLE contingency_periods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    company_id UUID NOT NULL,
    establishment_id UUID NOT NULL,
    point_of_sale_id UUID NOT NULL,
    ambiente VARCHAR(2) NOT NULL CHECK (ambiente IN ('00', '01')),
    
    f_inicio DATE NOT NULL,
    h_inicio TIME NOT NULL,
    f_fin DATE,
    h_fin TIME,
    
    tipo_contingencia INT NOT NULL CHECK (tipo_contingencia BETWEEN 1 AND 5),
    motivo_contingencia TEXT,
    
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'reporting', 'completed')),
    processing BOOLEAN DEFAULT false,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Create contingency_events (depends on periods)
CREATE TABLE contingency_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contingency_period_id UUID NOT NULL REFERENCES contingency_periods(id) ON DELETE CASCADE,
    
    codigo_generacion VARCHAR(36) NOT NULL,
    
    company_id UUID NOT NULL,
    establishment_id UUID NOT NULL,
    point_of_sale_id UUID NOT NULL,
    ambiente VARCHAR(2) NOT NULL,
    
    event_json JSONB NOT NULL,
    event_signed TEXT NOT NULL,
    
    estado VARCHAR(20),
    sello_recibido TEXT,
    hacienda_response JSONB,
    
    submitted_at TIMESTAMPTZ,
    accepted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. Create lotes (depends on events)
CREATE TABLE lotes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contingency_event_id UUID NOT NULL REFERENCES contingency_events(id) ON DELETE CASCADE,
    
    codigo_lote VARCHAR(100),
    
    company_id UUID NOT NULL,
    establishment_id UUID NOT NULL,
    point_of_sale_id UUID NOT NULL,
    
    dte_count INT NOT NULL,
    
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'submitted', 'completed')),
    processing BOOLEAN DEFAULT false,
    
    hacienda_response JSONB,
    
    submitted_at TIMESTAMPTZ,
    last_polled_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 4. Modify invoices table
ALTER TABLE invoices ADD COLUMN contingency_period_id UUID REFERENCES contingency_periods(id) ON DELETE SET NULL;
ALTER TABLE invoices ADD COLUMN contingency_event_id UUID REFERENCES contingency_events(id) ON DELETE SET NULL;
ALTER TABLE invoices ADD COLUMN lote_id UUID REFERENCES lotes(id) ON DELETE SET NULL;

ALTER TABLE invoices ADD COLUMN dte_transmission_status VARCHAR(20) DEFAULT 'pending' 
    CHECK (dte_transmission_status IN (
        'pending', 'pending_signature', 'contingency_queued', 
        'failed_retry', 'procesado', 'rechazado'
    ));

ALTER TABLE invoices ADD COLUMN dte_unsigned JSONB;
ALTER TABLE invoices ADD COLUMN dte_signed TEXT;
ALTER TABLE invoices ADD COLUMN dte_sello_recibido TEXT;
ALTER TABLE invoices ADD COLUMN hacienda_observaciones TEXT[];
ALTER TABLE invoices ADD COLUMN signature_retry_count INT DEFAULT 0;

-- 5. Create indexes
CREATE UNIQUE INDEX idx_invoices_codigo_generacion ON invoices(codigo_generacion);
CREATE INDEX idx_invoices_contingency_period ON invoices(contingency_period_id) WHERE contingency_period_id IS NOT NULL;
CREATE INDEX idx_invoices_contingency_event ON invoices(contingency_event_id) WHERE contingency_event_id IS NOT NULL;
CREATE INDEX idx_invoices_lote ON invoices(lote_id) WHERE lote_id IS NOT NULL;
CREATE INDEX idx_invoices_status ON invoices(dte_transmission_status);
CREATE INDEX idx_invoices_pending_sig ON invoices(contingency_period_id, signature_retry_count) 
    WHERE dte_transmission_status = 'pending_signature';

CREATE INDEX idx_periods_active ON contingency_periods(company_id, status) 
    WHERE status IN ('active', 'reporting');
CREATE INDEX idx_periods_claimable ON contingency_periods(status, processing, created_at) 
    WHERE status IN ('active', 'reporting') AND processing = false;
CREATE INDEX idx_periods_by_pos ON contingency_periods(company_id, establishment_id, point_of_sale_id);

CREATE UNIQUE INDEX idx_one_active_period_per_pos 
ON contingency_periods(company_id, establishment_id, point_of_sale_id, ambiente)
WHERE status = 'active';

CREATE INDEX idx_events_by_period ON contingency_events(contingency_period_id);
CREATE INDEX idx_events_by_status ON contingency_events(estado);
CREATE UNIQUE INDEX idx_events_codigo ON contingency_events(codigo_generacion);

CREATE INDEX idx_lotes_by_event ON lotes(contingency_event_id);
CREATE INDEX idx_lotes_pending ON lotes(status, created_at) WHERE status IN ('pending', 'submitted');
CREATE INDEX idx_lotes_by_codigo ON lotes(codigo_lote) WHERE codigo_lote IS NOT NULL;
CREATE INDEX idx_lotes_claimable ON lotes(status, processing, created_at) 
    WHERE status IN ('pending', 'submitted') AND processing = false;

CREATE INDEX idx_periods_stale_locks ON contingency_periods(processing, updated_at) WHERE processing = true;
CREATE INDEX idx_lotes_stale_locks ON lotes(processing, updated_at) WHERE processing = true;

-- 6. Create triggers
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_contingency_periods_updated_at
    BEFORE UPDATE ON contingency_periods
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_lotes_updated_at
    BEFORE UPDATE ON lotes
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE OR REPLACE FUNCTION check_ambiente_consistency()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.contingency_period_id IS NOT NULL THEN
        IF NOT EXISTS (
            SELECT 1 FROM contingency_periods 
            WHERE id = NEW.contingency_period_id 
            AND ambiente = NEW.ambiente
        ) THEN
            RAISE EXCEPTION 'Invoice ambiente (%) does not match period ambiente', NEW.ambiente;
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER check_invoice_ambiente
    BEFORE INSERT OR UPDATE ON invoices
    FOR EACH ROW
    WHEN (NEW.contingency_period_id IS NOT NULL)
    EXECUTE FUNCTION check_ambiente_consistency();

-- Comments
COMMENT ON TABLE contingency_periods IS 'Tracks contingency periods (outage episodes) per POS';
COMMENT ON TABLE contingency_events IS 'Eventos de Contingencia submitted to Hacienda';
COMMENT ON TABLE lotes IS 'Batches of DTEs submitted during contingency recovery';

COMMENT ON COLUMN contingency_periods.f_fin IS 'End date - represents when period was closed (first event created), may be slightly after actual outage end';
COMMENT ON COLUMN contingency_periods.h_fin IS 'End time - represents when period was closed (first event created), may be slightly after actual outage end';
COMMENT ON COLUMN contingency_periods.processing IS 'Concurrency control flag - prevents duplicate processing by workers';
COMMENT ON COLUMN lotes.processing IS 'Concurrency control flag - prevents duplicate processing by workers';
