DROP INDEX IF EXISTS idx_inventory_items_tax_exempt;
ALTER TABLE inventory_items DROP COLUMN IF EXISTS is_tax_exempt;
