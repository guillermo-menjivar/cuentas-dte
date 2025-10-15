-- migrations/0029_fix_finalized_constraint.sql

ALTER TABLE invoices DROP CONSTRAINT check_finalized_has_dte;

ALTER TABLE invoices ADD CONSTRAINT check_finalized_has_dte CHECK (
    (status = 'draft') OR
    (status IN ('finalized', 'void') AND dte_numero_control IS NOT NULL)
);
