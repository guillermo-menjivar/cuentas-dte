-- ============================================================================
-- Modify Invoices Table for Contingency
-- ============================================================================

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
