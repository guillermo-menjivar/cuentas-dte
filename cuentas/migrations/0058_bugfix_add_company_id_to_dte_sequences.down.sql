-- Revert: Remove company_id from dte_sequences

-- Step 1: Drop index
DROP INDEX IF EXISTS idx_dte_sequences_company;

-- Step 2: Drop new primary key
ALTER TABLE dte_sequences
DROP CONSTRAINT dte_sequences_pkey;

-- Step 3: Restore old primary key
ALTER TABLE dte_sequences
ADD CONSTRAINT dte_sequences_pkey
PRIMARY KEY (point_of_sale_id, tipo_dte);

-- Step 4: Drop foreign key
ALTER TABLE dte_sequences
DROP CONSTRAINT dte_sequences_company_id_fkey;

-- Step 5: Drop column
ALTER TABLE dte_sequences
DROP COLUMN company_id;
