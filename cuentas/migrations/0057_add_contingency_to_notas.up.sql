-- Add contingency tracking columns to notas_debito table
ALTER TABLE notas_debito
ADD COLUMN IF NOT EXISTS contingency_period_id UUID REFERENCES contingency_periods(id),
ADD COLUMN IF NOT EXISTS contingency_event_id UUID REFERENCES contingency_events(id),
ADD COLUMN IF NOT EXISTS lote_id UUID REFERENCES lotes(id),
ADD COLUMN IF NOT EXISTS dte_transmission_status VARCHAR(20) DEFAULT 'pending',
ADD COLUMN IF NOT EXISTS dte_unsigned JSONB,
ADD COLUMN IF NOT EXISTS dte_signed TEXT,
ADD COLUMN IF NOT EXISTS signature_retry_count INT DEFAULT 0;

-- Add contingency tracking columns to notas_credito table
ALTER TABLE notas_credito
ADD COLUMN IF NOT EXISTS contingency_period_id UUID REFERENCES contingency_periods(id),
ADD COLUMN IF NOT EXISTS contingency_event_id UUID REFERENCES contingency_events(id),
ADD COLUMN IF NOT EXISTS lote_id UUID REFERENCES lotes(id),
ADD COLUMN IF NOT EXISTS dte_transmission_status VARCHAR(20) DEFAULT 'pending',
ADD COLUMN IF NOT EXISTS dte_unsigned JSONB,
ADD COLUMN IF NOT EXISTS dte_signed TEXT,
ADD COLUMN IF NOT EXISTS signature_retry_count INT DEFAULT 0;

-- Add indexes for notas_debito
CREATE INDEX IF NOT EXISTS idx_notas_debito_contingency_period ON notas_debito(contingency_period_id) WHERE contingency_period_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_notas_debito_transmission_status ON notas_debito(dte_transmission_status) WHERE dte_transmission_status != 'pending';

-- Add indexes for notas_credito
CREATE INDEX IF NOT EXISTS idx_notas_credito_contingency_period ON notas_credito(contingency_period_id) WHERE contingency_period_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_notas_credito_transmission_status ON notas_credito(dte_transmission_status) WHERE dte_transmission_status != 'pending';
