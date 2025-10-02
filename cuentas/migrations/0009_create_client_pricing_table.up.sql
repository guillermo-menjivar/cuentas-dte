-- Create client_pricing table
CREATE TABLE IF NOT EXISTS client_pricing (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    item_id UUID NOT NULL REFERENCES inventory_items(id) ON DELETE CASCADE,
    
    price_type VARCHAR(20) NOT NULL,
    value DECIMAL(15,2) NOT NULL,
    is_percentage BOOLEAN DEFAULT false,
    
    valid_from DATE,
    valid_until DATE,
    active BOOLEAN DEFAULT true,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Only one active price per client/item combination
    CONSTRAINT check_price_type CHECK (price_type IN ('discount', 'upcharge', 'fixed_price'))
);

-- Partial unique index for active records only
CREATE UNIQUE INDEX unique_client_item_active_pricing 
    ON client_pricing(client_id, item_id) 
    WHERE active = true;

-- Indexes
CREATE INDEX idx_client_pricing_client ON client_pricing(client_id);
CREATE INDEX idx_client_pricing_item ON client_pricing(item_id);
CREATE INDEX idx_client_pricing_active ON client_pricing(active) WHERE active = true;

-- Trigger to update updated_at timestamp
CREATE TRIGGER trigger_update_client_pricing_updated_at
    BEFORE UPDATE ON client_pricing
    FOR EACH ROW
    EXECUTE FUNCTION update_clients_updated_at();

-- Comments
COMMENT ON TABLE client_pricing IS 'Client-specific pricing overrides (discounts, upcharges, or fixed prices)';
COMMENT ON COLUMN client_pricing.price_type IS 'Type of pricing: discount, upcharge, or fixed_price';
COMMENT ON COLUMN client_pricing.value IS 'Amount or percentage depending on is_percentage flag';
COMMENT ON COLUMN client_pricing.is_percentage IS 'If true, value is a percentage; if false, value is a fixed amount';
