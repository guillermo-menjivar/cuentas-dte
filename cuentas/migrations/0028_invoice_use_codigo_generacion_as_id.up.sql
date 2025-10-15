-- Drop all foreign keys pointing to invoices.id
ALTER TABLE invoice_line_items DROP CONSTRAINT invoice_line_items_invoice_id_fkey;
ALTER TABLE payments DROP CONSTRAINT payments_invoice_id_fkey;
ALTER TABLE invoices DROP CONSTRAINT invoices_references_invoice_id_fkey;
ALTER TABLE invoice_payments DROP CONSTRAINT invoice_payments_invoice_id_fkey;
ALTER TABLE dte_submission_attempts DROP CONSTRAINT dte_submission_attempts_invoice_id_fkey;

-- Change all ID columns to VARCHAR(36)
ALTER TABLE invoices ALTER COLUMN id TYPE VARCHAR(36);
ALTER TABLE invoices ALTER COLUMN references_invoice_id TYPE VARCHAR(36);
ALTER TABLE invoice_line_items ALTER COLUMN invoice_id TYPE VARCHAR(36);
ALTER TABLE payments ALTER COLUMN invoice_id TYPE VARCHAR(36);
ALTER TABLE invoice_payments ALTER COLUMN invoice_id TYPE VARCHAR(36);
ALTER TABLE dte_submission_attempts ALTER COLUMN invoice_id TYPE VARCHAR(36);

-- Recreate all foreign keys
ALTER TABLE invoice_line_items ADD CONSTRAINT invoice_line_items_invoice_id_fkey FOREIGN KEY (invoice_id) REFERENCES invoices(id) ON DELETE CASCADE;
ALTER TABLE payments ADD CONSTRAINT payments_invoice_id_fkey FOREIGN KEY (invoice_id) REFERENCES invoices(id) ON DELETE CASCADE;
ALTER TABLE invoices ADD CONSTRAINT invoices_references_invoice_id_fkey FOREIGN KEY (references_invoice_id) REFERENCES invoices(id);
ALTER TABLE invoice_payments ADD CONSTRAINT invoice_payments_invoice_id_fkey FOREIGN KEY (invoice_id) REFERENCES invoices(id) ON DELETE CASCADE;
ALTER TABLE dte_submission_attempts ADD CONSTRAINT dte_submission_attempts_invoice_id_fkey FOREIGN KEY (invoice_id) REFERENCES invoices(id) ON DELETE CASCADE;
