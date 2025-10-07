ALTER TABLE invoices DROP CONSTRAINT IF EXISTS check_payment_method;
ALTER TABLE invoices DROP COLUMN IF EXISTS payment_method;
