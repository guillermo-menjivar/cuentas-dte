-- migrations/0032_alter_dte_commit_log_add_id_pk.sql

-- Step 1: Drop the existing primary key constraint
ALTER TABLE dte_commit_log DROP CONSTRAINT dte_commit_log_pkey;

-- Step 2: Add a new UUID column as the primary key
ALTER TABLE dte_commit_log ADD COLUMN id UUID DEFAULT gen_random_uuid();

-- Step 3: Backfill existing rows with UUIDs (in case there's existing data)
UPDATE dte_commit_log SET id = gen_random_uuid() WHERE id IS NULL;

-- Step 4: Make id NOT NULL and set as primary key
ALTER TABLE dte_commit_log ALTER COLUMN id SET NOT NULL;
ALTER TABLE dte_commit_log ADD PRIMARY KEY (id);

-- Step 5: codigo_generacion is no longer unique (one nota can reference multiple CCFs)
-- Create index on codigo_generacion for queries
CREATE INDEX idx_dte_commit_log_codigo_gen ON dte_commit_log(codigo_generacion);

-- Step 6: Create composite index for efficient queries by nota and invoice
CREATE INDEX idx_dte_commit_log_codigo_invoice ON dte_commit_log(codigo_generacion, invoice_id);

-- Step 7: Update comment to reflect new structure
COMMENT ON COLUMN dte_commit_log.id IS 'Internal primary key - auto-generated UUID';
COMMENT ON COLUMN dte_commit_log.codigo_generacion IS 'Código de Generación from Hacienda - shared by all CCFs in a single nota de débito';
COMMENT ON COLUMN dte_commit_log.invoice_id IS 'Specific CCF (invoice) being referenced or adjusted by this DTE submission';
