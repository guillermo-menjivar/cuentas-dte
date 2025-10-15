ALTER TABLE invoices DROP CONSTRAINT invoices_references_invoice_id_fkey;
ALTER TABLE invoices ALTER COLUMN id TYPE VARCHAR(36);
ALTER TABLE invoices ALTER COLUMN references_invoice_id TYPE VARCHAR(36);
ALTER TABLE invoices ADD CONSTRAINT invoices_references_invoice_id_fkey FOREIGN KEY (references_invoice_id) REFERENCES invoices(id);


