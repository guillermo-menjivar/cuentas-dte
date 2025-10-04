-- Add credit tracking fields to clients table
ALTER TABLE clients ADD COLUMN credit_limit DECIMAL(15,2) DEFAULT 0;
ALTER TABLE clients ADD COLUMN current_balance DECIMAL(15,2) DEFAULT 0;
ALTER TABLE clients ADD COLUMN credit_status VARCHAR(20) DEFAULT 'good_standing';

-- Add check constraint for credit status
ALTER TABLE clients ADD CONSTRAINT check_credit_status 
    CHECK (credit_status IN ('good_standing', 'over_limit', 'suspended'));

-- Add comments
COMMENT ON COLUMN clients.credit_limit IS 'Maximum credit amount allowed for this client';
COMMENT ON COLUMN clients.current_balance IS 'Current outstanding balance (total owed across all invoices)';
COMMENT ON COLUMN clients.credit_status IS 'Credit standing: good_standing, over_limit, suspended';
