-- =====================================================
-- Migration 35 DOWN: Remove inventory cost tracking tables
-- =====================================================

-- Drop tables (cascades will handle indexes and constraints)
DROP TABLE IF EXISTS inventory_state CASCADE;
DROP TABLE IF EXISTS inventory_events CASCADE;
