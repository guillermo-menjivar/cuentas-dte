-- Create invoice_payments table
CREATE TABLE IF NOT EXISTS invoice_payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES invoices(id),
    
    payment_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    payment_method VARCHAR(50) NOT NULL,
    amount DECIMAL(15,2) NOT NULL,
    
    reference_number VARCHAR(100),
    notes TEXT,
    
    created_by UUID,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT check_payment_amount_positive CHECK (amount > 0),
    CONSTRAINT check_payment_method CHECK (payment_method IN ('cash', 'card', 'transfer', 'check', 'other'))
);

-- Indexes
CREATE INDEX idx_payments_invoice ON invoice_payments(invoice_id);
CREATE INDEX idx_payments_date ON invoice_payments(payment_date);

-- Comments
COMMENT ON TABLE invoice_payments IS 'Payment records for invoices';
COMMENT ON COLUMN invoice_payments.payment_method IS 'Payment method: cash, card, transfer, check, other';
COMMENT ON COLUMN invoice_payments.reference_number IS 'Check number, transfer ID, transaction reference, etc.';
