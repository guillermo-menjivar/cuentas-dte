DROP INDEX IF EXISTS idx_invoices_pos;
DROP INDEX IF EXISTS idx_invoices_establishment;

ALTER TABLE invoices 
DROP COLUMN IF EXISTS point_of_sale_id,
DROP COLUMN IF EXISTS establishment_id;
