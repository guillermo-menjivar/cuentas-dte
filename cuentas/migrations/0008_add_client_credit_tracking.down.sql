ALTER TABLE clients DROP CONSTRAINT IF EXISTS check_credit_status;
ALTER TABLE clients DROP COLUMN IF EXISTS credit_limit;
ALTER TABLE clients DROP COLUMN IF EXISTS current_balance;
ALTER TABLE clients DROP COLUMN IF EXISTS credit_status;
