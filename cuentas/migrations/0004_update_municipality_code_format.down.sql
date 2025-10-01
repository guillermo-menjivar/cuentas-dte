ALTER TABLE clients ALTER COLUMN municipality_code TYPE VARCHAR(4);

-- Remove comment
COMMENT ON COLUMN clients.municipality_code IS NULL;
