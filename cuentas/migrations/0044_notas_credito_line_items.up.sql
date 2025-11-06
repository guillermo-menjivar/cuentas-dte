CREATE TABLE notas_credito_line_items (
    id UUID PRIMARY KEY,
    nota_credito_id UUID REFERENCES notas_credito,
    line_number INTEGER,
    
    -- References to CCF
    related_ccf_id UUID REFERENCES invoices(id),
    related_ccf_number VARCHAR(50),
    ccf_line_item_id UUID REFERENCES invoice_line_items(id),
    
    -- Original item snapshot
    original_item_sku, original_item_name,
    original_unit_price, original_quantity,
    original_item_tipo_item, original_unit_of_measure,
    
    -- NEW: Credit-specific fields
    quantity_credited DECIMAL(15,8) NOT NULL,
    -- How many units credited (can be partial: 3 of 10)
    
    credit_amount DECIMAL(15,2) NOT NULL,
    -- Per-unit credit (POSITIVE)
    
    credit_reason VARCHAR(200),
    
    -- Calculated totals
    line_subtotal, discount_amount, taxable_amount,
    total_taxes, line_total,
    
    CONSTRAINT positive_quantities CHECK (
        quantity_credited > 0 AND 
        quantity_credited <= original_quantity
    )
);
