-- Add company_id to dte_sequences for proper data isolation
-- and easier querying by company

-- Step 1: Add company_id column (nullable first)
ALTER TABLE dte_sequences
ADD COLUMN company_id UUID;

-- Step 2: Backfill company_id from existing relationships
UPDATE dte_sequences ds
SET company_id = e.company_id
FROM point_of_sale pos
JOIN establishments e ON pos.establishment_id = e.id
WHERE ds.point_of_sale_id = pos.id;

-- Step 3: Make company_id NOT NULL after backfill
ALTER TABLE dte_sequences
ALTER COLUMN company_id SET NOT NULL;

-- Step 4: Add foreign key constraint
ALTER TABLE dte_sequences
ADD CONSTRAINT dte_sequences_company_id_fkey
FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE;

-- Step 5: Drop old primary key
ALTER TABLE dte_sequences
DROP CONSTRAINT dte_sequences_pkey;

-- Step 6: Create new primary key including company_id
ALTER TABLE dte_sequences
ADD CONSTRAINT dte_sequences_pkey
PRIMARY KEY (company_id, point_of_sale_id, tipo_dte);

-- Step 7: Add index for company-level queries
CREATE INDEX idx_dte_sequences_company ON dte_sequences(company_id);
