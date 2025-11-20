-- Migration: Rollback DTE Storage Index Table
-- Description: Removes the DTE storage index and related objects
-- Date: 2025-01-15

-- ============================================================================
-- Drop views
-- ============================================================================

DROP VIEW IF EXISTS v_dte_storage_recent;
DROP VIEW IF EXISTS v_dte_storage_stats;

-- ============================================================================
-- Drop functions
-- ============================================================================

DROP FUNCTION IF EXISTS get_analytics_path_components(UUID, VARCHAR, INT, INT, INT);
DROP FUNCTION IF EXISTS get_dtes_by_date_range(UUID, DATE, DATE, VARCHAR);
DROP FUNCTION IF EXISTS get_dte_download_path(UUID, VARCHAR);

-- ============================================================================
-- Drop trigger
-- ============================================================================

DROP TRIGGER IF EXISTS trigger_dte_storage_updated_at ON dte_storage_index;
DROP FUNCTION IF EXISTS update_dte_storage_updated_at();

-- ============================================================================
-- Drop table
-- ============================================================================

DROP TABLE IF EXISTS dte_storage_index;
