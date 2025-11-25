-- Remove ambiente consistency trigger (invoices doesn't have ambiente column)
DROP TRIGGER IF EXISTS check_invoice_ambiente ON invoices;
DROP FUNCTION IF EXISTS check_ambiente_consistency();
