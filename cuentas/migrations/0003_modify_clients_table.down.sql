-- Drop partial unique indexes
DROP INDEX IF EXISTS unique_client_nit_per_company;
DROP INDEX IF EXISTS unique_client_dui_per_company;

-- Remove check constraints
ALTER TABLE clients DROP CONSTRAINT IF EXISTS check_at_least_one_id;
ALTER TABLE clients DROP CONSTRAINT IF EXISTS check_nit_requires_ncr;

-- Recreate original unique constraints
ALTER TABLE clients ADD CONSTRAINT unique_client_nit_per_company UNIQUE (company_id, nit);
ALTER TABLE clients ADD CONSTRAINT unique_client_dui_per_company UNIQUE (company_id, dui);

-- Revert to NOT NULL (this will fail if there are NULL values in the database)
ALTER TABLE clients ALTER COLUMN ncr SET NOT NULL;
ALTER TABLE clients ALTER COLUMN nit SET NOT NULL;
ALTER TABLE clients ALTER COLUMN dui SET NOT NULL;
