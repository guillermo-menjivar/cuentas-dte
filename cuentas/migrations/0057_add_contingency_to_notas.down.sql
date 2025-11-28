-- Remove contingency tracking columns from notas_debito
ALTER TABLE notas_debito
DROP COLUMN IF EXISTS contingency_period_id,
DROP COLUMN IF EXISTS contingency_event_id,
DROP COLUMN IF EXISTS lote_id,
DROP COLUMN IF EXISTS dte_transmission_status,
DROP COLUMN IF EXISTS dte_unsigned,
DROP COLUMN IF EXISTS dte_signed,
DROP COLUMN IF EXISTS signature_retry_count;

-- Remove contingency tracking columns from notas_credito
ALTER TABLE notas_credito
DROP COLUMN IF EXISTS contingency_period_id,
DROP COLUMN IF EXISTS contingency_event_id,
DROP COLUMN IF EXISTS lote_id,
DROP COLUMN IF EXISTS dte_transmission_status,
DROP COLUMN IF EXISTS dte_unsigned,
DROP COLUMN IF EXISTS dte_signed,
DROP COLUMN IF EXISTS signature_retry_count;
