ALTER TABLE invoices 
ADD COLUMN destination_establishment_id UUID REFERENCES establishments(id);

-- Add index for queries
CREATE INDEX idx_invoices_destination_establishment 
ON invoices(destination_establishment_id) 
WHERE destination_establishment_id IS NOT NULL;
