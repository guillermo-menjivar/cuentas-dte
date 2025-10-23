-- =====================================================
-- Migration 36 DOWN: Restore legacy inventory columns
-- =====================================================
-- WARNING: Data will be lost on rollback
-- =====================================================

-- Drop the modified constraint
ALTER TABLE inventory_items DROP CONSTRAINT IF EXISTS check_prices_positive;

-- Add columns back
ALTER TABLE inventory_items ADD COLUMN IF NOT EXISTS cost_price DECIMAL(15,2);
ALTER TABLE inventory_items ADD COLUMN IF NOT EXISTS current_stock DECIMAL(15,3) DEFAULT 0;

-- Re-add original constraint
ALTER TABLE inventory_items ADD CONSTRAINT check_prices_positive 
    CHECK (unit_price >= 0 AND (cost_price IS NULL OR cost_price >= 0));
