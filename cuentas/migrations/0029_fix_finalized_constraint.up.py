BEGIN;

-- Drop the old constraint
ALTER TABLE invoices 
DROP CONSTRAINT IF EXISTS check_finalized_has_dte;

-- Add the corrected constraint
ALTER TABLE invoices
ADD CONSTRAINT check_finalized_has_dte CHECK (
    (status = 'draft') OR
    (status IN ('finalized', 'void') AND dte_numero_control IS NOT NULL)
);

COMMIT;
