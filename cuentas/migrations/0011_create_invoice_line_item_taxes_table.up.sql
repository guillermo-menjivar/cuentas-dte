CREATE TABLE IF NOT EXISTS invoice_line_item_taxes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    line_item_id UUID NOT NULL REFERENCES invoice_line_items(id) ON DELETE CASCADE,
    
    -- Tax snapshot (complete state at transaction time)
    tributo_code VARCHAR(10) NOT NULL,
    tributo_name VARCHAR(255) NOT NULL,
    
    -- Rate and calculation
    tax_rate DECIMAL(8,6) NOT NULL,
    taxable_base DECIMAL(15,2) NOT NULL,
    tax_amount DECIMAL(15,2) NOT NULL,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT check_tax_rate_valid CHECK (tax_rate >= 0 AND tax_rate <= 1)
);

-- Indexes
CREATE INDEX idx_line_item_taxes_line ON invoice_line_item_taxes(line_item_id);
CREATE INDEX idx_line_item_taxes_code ON invoice_line_item_taxes(tributo_code);

-- Comments
COMMENT ON TABLE invoice_line_item_taxes IS 'Tax details per line item with rate snapshots';
COMMENT ON COLUMN invoice_line_item_taxes.tax_rate IS 'Tax rate at transaction time as decimal (e.g., 0.13 for 13%)';
COMMENT ON COLUMN invoice_line_item_taxes.taxable_base IS 'Amount this tax was calculated on';
COMMENT ON COLUMN invoice_line_item_taxes.tax_amount IS 'Calculated tax amount';
