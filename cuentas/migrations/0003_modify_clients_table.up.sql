-- Make ncr, nit, and dui nullable since they're optional
ALTER TABLE clients ALTER COLUMN ncr DROP NOT NULL;
ALTER TABLE clients ALTER COLUMN nit DROP NOT NULL;
ALTER TABLE clients ALTER COLUMN dui DROP NOT NULL;

-- Drop existing unique constraints
ALTER TABLE clients DROP CONSTRAINT IF EXISTS unique_client_nit_per_company;
ALTER TABLE clients DROP CONSTRAINT IF EXISTS unique_client_dui_per_company;

-- Add check constraint to ensure at least one identification is provided
ALTER TABLE clients ADD CONSTRAINT check_at_least_one_id 
    CHECK (ncr IS NOT NULL OR nit IS NOT NULL OR dui IS NOT NULL);

-- Add check constraint to ensure if NIT is provided, NCR must also be provided
ALTER TABLE clients ADD CONSTRAINT check_nit_requires_ncr 
    CHECK ((nit IS NULL AND ncr IS NULL) OR (nit IS NOT NULL AND ncr IS NOT NULL));

-- Recreate unique constraints that work with NULL values
CREATE UNIQUE INDEX unique_client_nit_per_company ON clients(company_id, nit) WHERE nit IS NOT NULL;
CREATE UNIQUE INDEX unique_client_dui_per_company ON clients(company_id, dui) WHERE dui IS NOT NULL;
