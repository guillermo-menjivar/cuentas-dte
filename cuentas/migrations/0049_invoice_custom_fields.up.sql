ALTER TABLE invoices 
ADD COLUMN custom_fields JSONB;

-- Add index for querying custom fields
CREATE INDEX idx_invoices_custom_fields ON invoices USING GIN (custom_fields);

-- Add comment
COMMENT ON COLUMN invoices.custom_fields IS 'Custom appendix fields (apendice) for DTE - array of {campo, etiqueta, valor}';
