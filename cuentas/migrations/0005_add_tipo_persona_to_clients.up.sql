-- Add tipo_persona column to clients table
ALTER TABLE clients ADD COLUMN tipo_persona VARCHAR(1) NOT NULL DEFAULT '1';

-- Add check constraint to ensure valid values
ALTER TABLE clients ADD CONSTRAINT check_valid_tipo_persona 
    CHECK (tipo_persona IN ('1', '2'));

-- Add comment
COMMENT ON COLUMN clients.tipo_persona IS 'Type of person: 1 = Natural Person (Individual), 2 = Juridical Person (Company/Legal Entity)';
