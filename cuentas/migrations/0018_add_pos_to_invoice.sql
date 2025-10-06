ALTER TABLE invoices 
ADD COLUMN point_of_sale_id UUID REFERENCES point_of_sale(id);

-- For existing invoices, we'll need to handle this separately
-- For now, allow NULL, then later make NOT NULL after migration
