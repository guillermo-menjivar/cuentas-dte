-- Create inventory_transactions table
CREATE TABLE IF NOT EXISTS inventory_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    item_id UUID NOT NULL REFERENCES inventory_items(id) ON DELETE CASCADE,
    
    transaction_type VARCHAR(20) NOT NULL,
    quantity DECIMAL(15,3) NOT NULL,
    
    -- Reference to source document
    reference_type VARCHAR(50),
    reference_id UUID,
    
    notes TEXT,
    transaction_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by UUID,
    
    CONSTRAINT check_transaction_type CHECK (
        transaction_type IN ('purchase', 'sale', 'adjustment', 'return', 'transfer')
    ),
    CONSTRAINT check_purchase_positive CHECK (
        transaction_type != 'purchase' OR quantity > 0
    ),
    CONSTRAINT check_sale_negative CHECK (
        transaction_type != 'sale' OR quantity < 0
    ),
    CONSTRAINT check_return_logic CHECK (
        transaction_type != 'return' OR quantity != 0
    )
);

-- Indexes
CREATE INDEX idx_inventory_txn_company ON inventory_transactions(company_id);
CREATE INDEX idx_inventory_txn_item ON inventory_transactions(item_id);
CREATE INDEX idx_inventory_txn_date ON inventory_transactions(transaction_date);
CREATE INDEX idx_inventory_txn_type ON inventory_transactions(transaction_type);

-- Function to update current_stock when inventory_transactions are inserted
CREATE OR REPLACE FUNCTION update_inventory_stock()
RETURNS TRIGGER AS $$
BEGIN
    -- Only update stock if the item tracks inventory
    UPDATE inventory_items 
    SET current_stock = current_stock + NEW.quantity,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.item_id 
      AND track_inventory = true;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger on INSERT
CREATE TRIGGER trigger_update_stock_on_insert
    AFTER INSERT ON inventory_transactions
    FOR EACH ROW
    EXECUTE FUNCTION update_inventory_stock();

-- Function to handle UPDATE on inventory_transactions
CREATE OR REPLACE FUNCTION update_inventory_stock_on_change()
RETURNS TRIGGER AS $$
BEGIN
    -- Only update stock if the item tracks inventory
    UPDATE inventory_items 
    SET current_stock = current_stock - OLD.quantity + NEW.quantity,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.item_id 
      AND track_inventory = true;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger on UPDATE
CREATE TRIGGER trigger_update_stock_on_update
    AFTER UPDATE ON inventory_transactions
    FOR EACH ROW
    WHEN (OLD.quantity IS DISTINCT FROM NEW.quantity)
    EXECUTE FUNCTION update_inventory_stock_on_change();

-- Function to handle DELETE on inventory_transactions
CREATE OR REPLACE FUNCTION update_inventory_stock_on_delete()
RETURNS TRIGGER AS $$
BEGIN
    -- Only update stock if the item tracks inventory
    UPDATE inventory_items 
    SET current_stock = current_stock - OLD.quantity,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = OLD.item_id 
      AND track_inventory = true;
    
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

-- Trigger on DELETE
CREATE TRIGGER trigger_update_stock_on_delete
    AFTER DELETE ON inventory_transactions
    FOR EACH ROW
    EXECUTE FUNCTION update_inventory_stock_on_delete();

-- Comments
COMMENT ON TABLE inventory_transactions IS 'Audit log of all inventory stock movements';
COMMENT ON COLUMN inventory_transactions.transaction_type IS 'Type of transaction: purchase, sale, adjustment, return, transfer';
COMMENT ON COLUMN inventory_transactions.quantity IS 'Quantity changed (positive for additions, negative for reductions)';
