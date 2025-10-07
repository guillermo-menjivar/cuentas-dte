ALTER TABLE invoices 
ADD COLUMN establishment_id UUID REFERENCES establishments(id),
ADD COLUMN point_of_sale_id UUID REFERENCES point_of_sale(id);

-- Index for queries by establishment
CREATE INDEX idx_invoices_establishment ON invoices(establishment_id);

-- Index for queries by POS
CREATE INDEX idx_invoices_pos ON invoices(point_of_sale_id);
