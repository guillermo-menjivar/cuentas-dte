-- ============================================================================
-- Migration 003: Create Indexes for Contingency
-- Description: Performance indexes for contingency queries
-- ============================================================================

-- Invoice indexes
CREATE UNIQUE INDEX idx_invoices_codigo_generacion ON invoices(codigo_generacion);
CREATE INDEX idx_invoices_contingency_period ON invoices(contingency_period_id) WHERE contingency_period_id IS NOT NULL;
CREATE INDEX idx_invoices_contingency_event ON invoices(contingency_event_id) WHERE contingency_event_id IS NOT NULL;
CREATE INDEX idx_invoices_lote ON invoices(lote_id) WHERE lote_id IS NOT NULL;
CREATE INDEX idx_invoices_status ON invoices(dte_transmission_status);
CREATE INDEX idx_invoices_pending_sig ON invoices(contingency_period_id, signature_retry_count) 
    WHERE dte_transmission_status = 'pending_signature';

-- Period indexes
CREATE INDEX idx_periods_active ON contingency_periods(company_id, status) 
    WHERE status IN ('active', 'reporting');
CREATE INDEX idx_periods_claimable ON contingency_periods(status, processing, created_at) 
    WHERE status IN ('active', 'reporting') AND processing = false;
CREATE INDEX idx_periods_by_pos ON contingency_periods(company_id, establishment_id, point_of_sale_id);

-- Enforce one active period per POS
CREATE UNIQUE INDEX idx_one_active_period_per_pos 
ON contingency_periods(company_id, establishment_id, point_of_sale_id, ambiente)
WHERE status = 'active';

-- Event indexes
CREATE INDEX idx_events_by_period ON contingency_events(contingency_period_id);
CREATE INDEX idx_events_by_status ON contingency_events(estado);
CREATE UNIQUE INDEX idx_events_codigo ON contingency_events(codigo_generacion);

-- Lote indexes
CREATE INDEX idx_lotes_by_event ON lotes(contingency_event_id);
CREATE INDEX idx_lotes_pending ON lotes(status, created_at) 
    WHERE status IN ('pending', 'submitted');
CREATE INDEX idx_lotes_by_codigo ON lotes(codigo_lote) WHERE codigo_lote IS NOT NULL;
CREATE INDEX idx_lotes_claimable ON lotes(status, processing, created_at) 
    WHERE status IN ('pending', 'submitted') AND processing = false;

-- Performance: updated_at for stale lock cleanup
CREATE INDEX idx_periods_stale_locks ON contingency_periods(processing, updated_at) 
    WHERE processing = true;
CREATE INDEX idx_lotes_stale_locks ON lotes(processing, updated_at) 
    WHERE processing = true;
