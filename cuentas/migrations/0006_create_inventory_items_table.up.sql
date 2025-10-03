CREATE TABLE IF NOT EXISTS inventory_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    
    tipo_item VARCHAR(1) NOT NULL,
    
    sku VARCHAR(100) NOT NULL,
    codigo_barras VARCHAR(100),
    
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    manufacturer VARCHAR(255),
    image_url TEXT,
    
    cost_price DECIMAL(15,2),
    unit_price DECIMAL(15,2) NOT NULL,
    unit_of_measure VARCHAR(50) NOT NULL,
    
    color VARCHAR(50),
    
    track_inventory BOOLEAN DEFAULT true,
    current_stock DECIMAL(15,3) DEFAULT 0,
    minimum_stock DECIMAL(15,3) DEFAULT 0,
    
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT unique_company_sku UNIQUE (company_id, sku),
    CONSTRAINT unique_company_barcode UNIQUE (company_id, codigo_barras),
    CONSTRAINT check_tipo_item_valid CHECK (tipo_item IN ('1', '2')),
    CONSTRAINT check_prices_positive CHECK (unit_price >= 0 AND (cost_price IS NULL OR cost_price >= 0))
);

CREATE INDEX idx_inventory_items_company_id ON inventory_items(company_id);
CREATE INDEX idx_inventory_items_sku ON inventory_items(company_id, sku);
CREATE INDEX idx_inventory_items_barcode ON inventory_items(company_id, codigo_barras) WHERE codigo_barras IS NOT NULL;
CREATE INDEX idx_inventory_items_active ON inventory_items(active);
CREATE INDEX idx_inventory_items_tipo ON inventory_items(tipo_item);

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
