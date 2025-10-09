-- Create payments table to track all payments against invoices
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE RESTRICT,
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE RESTRICT,
    
    -- Payment details
    amount DECIMAL(15, 2) NOT NULL CHECK (amount > 0),
    payment_method VARCHAR(2) NOT NULL,   -- cat_012 formas de pago code
    payment_reference VARCHAR(100),        -- Optional reference (check number, transaction ID, etc.)
    
    -- Timestamps
    payment_date TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW(),
    created_by UUID,  -- Future: user who recorded payment
    
    -- Audit
    notes TEXT
);

-- Indexes for fast queries
CREATE INDEX idx_payments_company ON payments(company_id);
CREATE INDEX idx_payments_date ON payments(payment_date);
CREATE INDEX idx_payments_method ON payments(payment_method);

-- Add comments for documentation
COMMENT ON TABLE payments IS 'Record of all payments received against invoices';
COMMENT ON COLUMN payments.payment_method IS 'Payment method code from cat_012 (formas de pago)';
COMMENT ON COLUMN payments.amount IS 'Payment amount - must be positive';
COMMENT ON COLUMN payments.payment_reference IS 'Optional reference like check number, transaction ID, or receipt number';
