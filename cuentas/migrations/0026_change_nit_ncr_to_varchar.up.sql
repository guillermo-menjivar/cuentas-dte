-- Drop existing constraints and indexes
DROP INDEX IF EXISTS idx_companies_nit;
DROP INDEX IF EXISTS idx_companies_ncr;

-- Change column types
ALTER TABLE companies ALTER COLUMN nit TYPE VARCHAR(14);
ALTER TABLE companies ALTER COLUMN ncr TYPE VARCHAR(8);

-- Recreate indexes
CREATE INDEX idx_companies_nit ON companies(nit);
CREATE INDEX idx_companies_ncr ON companies(ncr);
