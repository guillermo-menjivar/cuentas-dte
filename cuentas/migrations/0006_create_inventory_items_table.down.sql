-- Drop trigger and function
DROP TRIGGER IF EXISTS trigger_update_inventory_items_updated_at ON inventory_items;
DROP FUNCTION IF EXISTS update_inventory_items_updated_at();

-- Drop table
DROP TABLE IF EXISTS inventory_items;
