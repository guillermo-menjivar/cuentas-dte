ALTER TABLE invoices ADD COLUMN dte_type VARCHAR(100);
COMMENT ON COLUMN invoices.dte_type IS 'Document the type of DTE of the invoice';
