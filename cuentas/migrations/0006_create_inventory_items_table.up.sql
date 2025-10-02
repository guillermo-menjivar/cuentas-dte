-- Create inventory_items table
CREATE TABLE IF NOT EXISTS inventory_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    
    -- Item classification
    tipo_item VARCHAR(1) NOT NULL,
    
    -- Identification
    sku VARCHAR(100) NOT NULL,
    codigo_barras VARCHAR(100),
    
    -- Basic info
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Product details
    manufacturer VARCHAR(255),
    image_url TEXT,
    
    -- For tipo 3 (Ambos) - percentage split
    goods_percentage DECIMAL(5,2),
    services_percentage DECIMAL(5,2),
    
    -- Pricing (tax-exclusive)
    cost_price DECIMAL(15,2),
    unit_price DECIMAL(15,2) NOT NULL,
    unit_of_measure VARCHAR(50) NOT NULL,
    
    -- Variant
    color VARCHAR(50),
    
    -- Inventory tracking
    track_inventory BOOLEAN DEFAULT true,
    current_stock DECIMAL(15,3) DEFAULT 0,
    minimum_stock DECIMAL(15,3) DEFAULT 0,
    
    -- Metadata
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT unique_company_sku UNIQUE (company_id, sku),
    CONSTRAINT unique_company_barcode UNIQUE (company_id, codigo_barras),
    CONSTRAINT check_tipo_item_valid CHECK (tipo_item IN ('1', '2', '3', '4')),
    CONSTRAINT check_ambos_percentages CHECK (
        (tipo_item != '3') OR 
        (goods_percentage IS NOT NULL AND services_percentage IS NOT NULL 
         AND goods_percentage + services_percentage = 100)
    ),
    CONSTRAINT check_prices_positive CHECK (unit_price >= 0 AND (cost_price IS NULL OR cost_price >= 0))
);

-- Indexes
CREATE INDEX idx_inventory_items_company_id ON inventory_items(company_id);
CREATE INDEX idx_inventory_items_sku ON inventory_items(company_id, sku);
CREATE INDEX idx_inventory_items_barcode ON inventory_items(company_id, codigo_barras) WHERE codigo_barras IS NOT NULL;
CREATE INDEX idx_inventory_items_active ON inventory_items(active);
CREATE INDEX idx_inventory_items_tipo ON inventory_items(tipo_item);
CREATE INDEX idx_inventory_items_stock_alerts ON inventory_items(company_id, current_stock) WHERE track_inventory = true AND active = true;

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_inventory_items_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_inventory_items_updated_at
    BEFORE UPDATE ON inventory_items
    FOR EACH ROW
    EXECUTE FUNCTION update_inventory_items_updated_at();

-- Comments
COMMENT ON TABLE inventory_items IS 'Inventory items including goods, services, and both';
COMMENT ON COLUMN inventory_items.tipo_item IS 'Type of item: 1=Bienes, 2=Servicios, 3=Ambos, 4=Otros tributos por ítem';
COMMENT ON COLUMN inventory_items.goods_percentage IS 'For tipo 3 (Ambos): percentage that is goods (must sum to 100 with services_percentage)';
COMMENT ON COLUMN inventory_items.services_percentage IS 'For tipo 3 (Ambos): percentage that is services (must sum to 100 with goods_percentage)';
COMMENT ON COLUMN inventory_items.cost_price IS 'Cost to purchase/produce the item (nullable for services)';
COMMENT ON COLUMN inventory_items.unit_price IS 'Sale price per unit (tax-exclusive)';
COMMENT ON COLUMN inventory_items.current_stock IS 'Current stock level (can be negative to track oversold items)';
