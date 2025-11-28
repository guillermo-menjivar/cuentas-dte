-- Remove contingency tracking columns from purchases table
ALTER TABLE purchases
DROP COLUMN IF EXISTS contingency_period_id,
DROP COLUMN IF EXISTS contingency_event_id,
DROP COLUMN IF EXISTS lote_id,
DROP COLUMN IF EXISTS dte_transmission_status,
DROP COLUMN IF EXISTS dte_unsigned,
DROP COLUMN IF EXISTS dte_signed,
DROP COLUMN IF EXISTS hacienda_observaciones,
DROP COLUMN IF EXISTS signature_retry_count;
