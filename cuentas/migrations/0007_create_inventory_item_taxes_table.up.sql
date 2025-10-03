CREATE TABLE IF NOT EXISTS inventory_item_taxes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    item_id UUID NOT NULL REFERENCES inventory_items(id) ON DELETE CASCADE,
    tributo_code VARCHAR(10) NOT NULL,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT unique_item_tributo UNIQUE (item_id, tributo_code)
);

CREATE INDEX idx_item_taxes_item_id ON inventory_item_taxes(item_id);
