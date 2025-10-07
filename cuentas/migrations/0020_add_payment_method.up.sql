ALTER TABLE invoices ADD COLUMN payment_method VARCHAR(2);

-- Add check constraint
ALTER TABLE invoices ADD CONSTRAINT check_payment_method 
CHECK (payment_method IS NULL OR payment_method IN (
    '01', '02', '03', '04', '05', '08', '09', '11', '12', '13', '14', '99'
));
