ALTER TABLE inventory_items 
ADD COLUMN is_tax_exempt BOOLEAN DEFAULT false NOT NULL;

CREATE INDEX idx_inventory_items_tax_exempt 
ON inventory_items(company_id, is_tax_exempt) 
WHERE is_tax_exempt = true;
