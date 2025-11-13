-- Make client_id nullable for internal transfers (remisiones without external client)
ALTER TABLE dte_commit_log 
ALTER COLUMN client_id DROP NOT NULL;

-- Add comment explaining when it can be NULL
COMMENT ON COLUMN dte_commit_log.client_id IS 'Client ID - NULL for internal transfers (Type 04 remisiones between establishments)';
