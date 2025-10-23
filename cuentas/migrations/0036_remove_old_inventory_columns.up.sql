-- =====================================================
-- Migration 36 UP: Remove legacy inventory columns
-- =====================================================
-- NOTE: Only run this AFTER your application code has been updated
--       to use inventory_state instead of current_stock/cost_price
-- =====================================================

-- Drop the constraint that references cost_price
ALTER TABLE inventory_items DROP CONSTRAINT IF EXISTS check_prices_positive;

-- Remove legacy columns
ALTER TABLE inventory_items DROP COLUMN IF EXISTS current_stock;
ALTER TABLE inventory_items DROP COLUMN IF EXISTS cost_price;

-- Re-add constraint without cost_price reference
ALTER TABLE inventory_items ADD CONSTRAINT check_prices_positive CHECK (unit_price >= 0);
