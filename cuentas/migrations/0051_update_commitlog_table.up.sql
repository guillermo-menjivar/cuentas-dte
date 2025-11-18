-- Drop the NOT NULL constraint from invoice_id
ALTER TABLE dte_commit_log 
ALTER COLUMN invoice_id DROP NOT NULL;

-- Drop the NOT NULL constraint from invoice_number (same issue)
ALTER TABLE dte_commit_log 
ALTER COLUMN invoice_number DROP NOT NULL;
