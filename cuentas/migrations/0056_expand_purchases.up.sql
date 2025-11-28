-- Add contingency tracking columns to purchases table
ALTER TABLE purchases
ADD COLUMN IF NOT EXISTS contingency_period_id UUID REFERENCES contingency_periods(id),
ADD COLUMN IF NOT EXISTS contingency_event_id UUID REFERENCES contingency_events(id),
ADD COLUMN IF NOT EXISTS lote_id UUID REFERENCES lotes(id),
ADD COLUMN IF NOT EXISTS dte_transmission_status VARCHAR(20) DEFAULT 'pending',
ADD COLUMN IF NOT EXISTS dte_unsigned JSONB,
ADD COLUMN IF NOT EXISTS dte_signed TEXT,
ADD COLUMN IF NOT EXISTS hacienda_observaciones TEXT[],
ADD COLUMN IF NOT EXISTS signature_retry_count INT DEFAULT 0;

-- Add indexes
CREATE INDEX IF NOT EXISTS idx_purchases_contingency_period ON purchases(contingency_period_id) WHERE contingency_period_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_purchases_transmission_status ON purchases(dte_transmission_status) WHERE dte_transmission_status != 'pending';
