-- migrations/XXX_change_nit_ncr_to_varchar.down.sql

-- Drop indexes
DROP INDEX IF EXISTS idx_companies_nit;
DROP INDEX IF EXISTS idx_companies_ncr;

-- Change back to BIGINT
ALTER TABLE companies ALTER COLUMN nit TYPE BIGINT USING nit::BIGINT;
ALTER TABLE companies ALTER COLUMN ncr TYPE BIGINT USING ncr::BIGINT;

-- Recreate indexes
CREATE INDEX idx_companies_nit ON companies(nit);
CREATE INDEX idx_companies_ncr ON companies(ncr);
