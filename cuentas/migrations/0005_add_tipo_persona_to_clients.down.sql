-- Remove tipo_persona column
ALTER TABLE clients DROP CONSTRAINT IF EXISTS check_valid_tipo_persona;
ALTER TABLE clients DROP COLUMN IF EXISTS tipo_persona;
