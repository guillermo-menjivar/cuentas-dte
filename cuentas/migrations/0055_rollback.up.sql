-- ============================================================================
-- Rollback old contingency schema
-- ============================================================================

-- Drop old triggers
DROP TRIGGER IF EXISTS update_contingency_batches_updated_at ON dte_contingency_batches;
DROP TRIGGER IF EXISTS update_contingency_events_updated_at ON dte_contingency_events;
DROP TRIGGER IF EXISTS update_contingency_queue_updated_at ON dte_contingency_queue;

-- Drop old tables (in reverse dependency order)
DROP TABLE IF EXISTS dte_contingency_queue CASCADE;
DROP TABLE IF EXISTS dte_contingency_batches CASCADE;
DROP TABLE IF EXISTS dte_contingency_events CASCADE;

-- Function can stay (we'll reuse it)
-- DROP FUNCTION IF EXISTS update_updated_at_column();
