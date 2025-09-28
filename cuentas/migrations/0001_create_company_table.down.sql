-- Drop trigger
DROP TRIGGER IF EXISTS update_companies_updated_at ON companies;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_companies_last_activity;
DROP INDEX IF EXISTS idx_companies_active;
DROP INDEX IF EXISTS idx_companies_email;
DROP INDEX IF EXISTS idx_companies_ncr;
DROP INDEX IF EXISTS idx_companies_nit;

-- Drop table
DROP TABLE IF EXISTS companies;
