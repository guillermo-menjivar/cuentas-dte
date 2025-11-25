-- ============================================================================
-- Migration 005: Rollback Script (if needed)
-- Description: Removes all contingency changes
-- ============================================================================

-- Drop triggers
DROP TRIGGER IF EXISTS check_invoice_ambiente ON invoices;
DROP TRIGGER IF EXISTS update_lotes_updated_at ON lotes;
DROP TRIGGER IF EXISTS update_contingency_periods_updated_at ON contingency_periods;

-- Drop functions
DROP FUNCTION IF EXISTS check_ambiente_consistency();
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop invoice columns
ALTER TABLE invoices DROP COLUMN IF EXISTS contingency_period_id;
ALTER TABLE invoices DROP COLUMN IF EXISTS contingency_event_id;
ALTER TABLE invoices DROP COLUMN IF EXISTS lote_id;
ALTER TABLE invoices DROP COLUMN IF EXISTS dte_transmission_status;
ALTER TABLE invoices DROP COLUMN IF EXISTS dte_unsigned;
ALTER TABLE invoices DROP COLUMN IF EXISTS dte_signed;
ALTER TABLE invoices DROP COLUMN IF EXISTS dte_sello_recibido;
ALTER TABLE invoices DROP COLUMN IF EXISTS hacienda_observaciones;
ALTER TABLE invoices DROP COLUMN IF EXISTS signature_retry_count;

-- Drop tables (cascade removes foreign keys)
DROP TABLE IF EXISTS lotes CASCADE;
DROP TABLE IF EXISTS contingency_events CASCADE;
DROP TABLE IF EXISTS contingency_periods CASCADE;
