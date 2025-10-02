-- Create inventory_item_taxes table
CREATE TABLE IF NOT EXISTS inventory_item_taxes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    item_id UUID NOT NULL REFERENCES inventory_items(id) ON DELETE CASCADE,
    tributo_code VARCHAR(10) NOT NULL,
    
    -- For tipo 3 (Ambos) items - specify which portion this tax applies to
    applies_to_goods BOOLEAN DEFAULT true,
    applies_to_services BOOLEAN DEFAULT true,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT unique_item_tributo UNIQUE (item_id, tributo_code)
);

-- Indexes
CREATE INDEX idx_item_taxes_item_id ON inventory_item_taxes(item_id);

-- Comments
COMMENT ON TABLE inventory_item_taxes IS 'Tax configuration for inventory items';
COMMENT ON COLUMN inventory_item_taxes.tributo_code IS 'Tax code from CAT-015 (e.g., S1.20 for IVA 13%)';
COMMENT ON COLUMN inventory_item_taxes.applies_to_goods IS 'For tipo 3 (Ambos) items: whether this tax applies to the goods portion';
COMMENT ON COLUMN inventory_item_taxes.applies_to_services IS 'For tipo 3 (Ambos) items: whether this tax applies to the services portion';
