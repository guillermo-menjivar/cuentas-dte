-- Create invoice_line_items table
CREATE TABLE IF NOT EXISTS invoice_line_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    line_number INT NOT NULL,
    
    -- Item reference (for tracking only, not for pricing)
    item_id UUID REFERENCES inventory_items(id),
    
    -- Item snapshot (complete state at transaction time)
    item_sku VARCHAR(100) NOT NULL,
    item_name VARCHAR(255) NOT NULL,
    item_description TEXT,
    item_tipo_item VARCHAR(1) NOT NULL,
    unit_of_measure VARCHAR(50) NOT NULL,
    
    -- Pricing snapshot
    unit_price DECIMAL(15,2) NOT NULL,
    quantity DECIMAL(15,3) NOT NULL,
    line_subtotal DECIMAL(15,2) NOT NULL,
    
    -- Discount (line-level)
    discount_percentage DECIMAL(5,2) DEFAULT 0,
    discount_amount DECIMAL(15,2) DEFAULT 0,
    
    -- Tax calculations
    taxable_amount DECIMAL(15,2) NOT NULL,
    total_taxes DECIMAL(15,2) NOT NULL,
    line_total DECIMAL(15,2) NOT NULL,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT check_quantity_positive CHECK (quantity > 0),
    CONSTRAINT check_line_number_positive CHECK (line_number > 0),
    CONSTRAINT unique_invoice_line UNIQUE (invoice_id, line_number)
);

-- Indexes
CREATE INDEX idx_line_items_invoice ON invoice_line_items(invoice_id);
CREATE INDEX idx_line_items_item ON invoice_line_items(item_id);

-- Comments
COMMENT ON TABLE invoice_line_items IS 'Invoice line items with complete item snapshots';
COMMENT ON COLUMN invoice_line_items.item_id IS 'Reference to inventory item (for tracking, not pricing)';
COMMENT ON COLUMN invoice_line_items.unit_price IS 'Price at transaction time (snapshot)';
COMMENT ON COLUMN invoice_line_items.taxable_amount IS 'Amount after discount, used for tax calculation';
