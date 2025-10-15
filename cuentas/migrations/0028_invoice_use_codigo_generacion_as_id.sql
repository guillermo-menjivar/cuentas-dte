-- migrations/0028_invoice_use_codigo_generacion_as_id.sql

-- Step 1: Add new id column (varchar)
ALTER TABLE invoices ADD COLUMN new_id VARCHAR(36);

-- Step 2: Populate new_id with uppercase UUIDs for existing records
UPDATE invoices SET new_id = UPPER(id::text);

-- Step 3: Make new_id NOT NULL
ALTER TABLE invoices ALTER COLUMN new_id SET NOT NULL;

-- Step 4: Drop foreign key constraints that reference invoices.id
ALTER TABLE invoice_line_items DROP CONSTRAINT IF EXISTS invoice_line_items_invoice_id_fkey;
ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_invoice_id_fkey;

-- Step 5: Drop old primary key
ALTER TABLE invoices DROP CONSTRAINT invoices_pkey;

-- Step 6: Add new primary key on new_id
ALTER TABLE invoices ADD PRIMARY KEY (new_id);

-- Step 7: Drop old id column
ALTER TABLE invoices DROP COLUMN id;

-- Step 8: Rename new_id to id
ALTER TABLE invoices RENAME COLUMN new_id TO id;

-- Step 9: Recreate foreign key constraints
ALTER TABLE invoice_line_items 
ADD CONSTRAINT invoice_line_items_invoice_id_fkey 
FOREIGN KEY (invoice_id) REFERENCES invoices(id) ON DELETE CASCADE;

ALTER TABLE payments 
ADD CONSTRAINT payments_invoice_id_fkey 
FOREIGN KEY (invoice_id) REFERENCES invoices(id) ON DELETE CASCADE;

-- Step 10: Update references_invoice_id to VARCHAR(36)
ALTER TABLE invoices ALTER COLUMN references_invoice_id TYPE VARCHAR(36);

-- Step 11: Drop dte_codigo_generacion column
ALTER TABLE invoices DROP COLUMN IF EXISTS dte_codigo_generacion;

-- Step 12: Drop and recreate the finalized constraint
ALTER TABLE invoices DROP CONSTRAINT IF EXISTS check_finalized_has_dte;
ALTER TABLE invoices 
ADD CONSTRAINT check_finalized_has_dte CHECK (
    (status = 'draft') OR
    (status IN ('finalized', 'void') AND dte_numero_control IS NOT NULL)
);

-- Step 13: Drop old index on dte_codigo_generacion
DROP INDEX IF EXISTS idx_invoices_dte_codigo_unique;

-- Step 14: Update comments
COMMENT ON COLUMN invoices.id IS 'Código de Generación (uppercase UUID) - serves as both invoice ID and DTE identifier for Hacienda';
