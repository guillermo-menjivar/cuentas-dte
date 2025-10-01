-- Update municipality_code column to support dot notation format (DD.MM)
ALTER TABLE clients ALTER COLUMN municipality_code TYPE VARCHAR(5);

-- Add comment explaining the format
COMMENT ON COLUMN clients.municipality_code IS 'Municipality code in format DD.MM where DD is department code and MM is municipality code (e.g., 06.23 for San Salvador Centro)';
