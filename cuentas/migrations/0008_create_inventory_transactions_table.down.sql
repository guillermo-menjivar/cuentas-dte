-- Drop triggers and functions
DROP TRIGGER IF EXISTS trigger_update_stock_on_delete ON inventory_transactions;
DROP TRIGGER IF EXISTS trigger_update_stock_on_update ON inventory_transactions;
DROP TRIGGER IF EXISTS trigger_update_stock_on_insert ON inventory_transactions;
DROP FUNCTION IF EXISTS update_inventory_stock_on_delete();
DROP FUNCTION IF EXISTS update_inventory_stock_on_change();
DROP FUNCTION IF EXISTS update_inventory_stock();

-- Drop table
DROP TABLE IF EXISTS inventory_transactions;
