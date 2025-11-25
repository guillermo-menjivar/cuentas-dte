-- ============================================================================
-- Migration 002: Modify Invoices Table for Contingency
-- Description: Add contingency tracking columns to invoices
-- ============================================================================

-- Add contingency tracking columns
ALTER TABLE invoices ADD COLUMN contingency_period_id UUID REFERENCES contingency_periods(id) ON DELETE SET NULL;
ALTER TABLE invoices ADD COLUMN contingency_event_id UUID REFERENCES contingency_events(id) ON DELETE SET NULL;
ALTER TABLE invoices ADD COLUMN lote_id UUID REFERENCES lotes(id) ON DELETE SET NULL;

-- Add DTE transmission tracking
ALTER TABLE invoices ADD COLUMN dte_transmission_status VARCHAR(20) DEFAULT 'pending' 
    CHECK (dte_transmission_status IN (
        'pending', 
        'pending_signature', 
        'contingency_queued', 
        'failed_retry', 
        'procesado', 
        'rechazado'
    ));

-- Add DTE storage
ALTER TABLE invoices ADD COLUMN dte_unsigned JSONB;
ALTER TABLE invoices ADD COLUMN dte_signed TEXT;
ALTER TABLE invoices ADD COLUMN dte_sello_recibido TEXT;
ALTER TABLE invoices ADD COLUMN hacienda_observaciones TEXT[];

-- Add signature retry tracking
ALTER TABLE invoices ADD COLUMN signature_retry_count INT DEFAULT 0;

-- Add comments
COMMENT ON COLUMN invoices.contingency_period_id IS 'Links to contingency period if in contingency';
COMMENT ON COLUMN invoices.contingency_event_id IS 'Links to event that reported this DTE';
COMMENT ON COLUMN invoices.lote_id IS 'Links to lote that submitted this DTE';
COMMENT ON COLUMN invoices.dte_transmission_status IS 'Current transmission status: pending -> pending_signature/contingency_queued/failed_retry -> procesado/rechazado';
COMMENT ON COLUMN invoices.dte_unsigned IS 'Unsigned DTE JSON for retry signing if firmador fails';
COMMENT ON COLUMN invoices.dte_signed IS 'Signed DTE (JWS string) ready for submission';
COMMENT ON COLUMN invoices.signature_retry_count IS 'Number of signature retry attempts (alert after 10)';
