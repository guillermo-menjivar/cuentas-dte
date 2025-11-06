CREATE TABLE notas_credito (
    id UUID PRIMARY KEY,
    company_id, establishment_id, point_of_sale_id,
    
    -- Document identification
    nota_number VARCHAR(50),  -- e.g., "NC-00000001"
    nota_type VARCHAR(2) DEFAULT '05',
    
    -- Client info (snapshot)
    client_id, client_name, client_legal_name, client_nit, ...
    
    -- NEW: Credit-specific fields
    credit_reason VARCHAR(20) NOT NULL,
    -- Values: 'void', 'return', 'discount', 'defect', 'overbilling', 
    --         'correction', 'quality', 'cancellation', 'other'
    
    credit_description TEXT,
    is_full_annulment BOOLEAN DEFAULT FALSE,
    
    -- Financial (POSITIVE numbers)
    subtotal, total_discount, total_taxes, total,
    currency, payment_terms, payment_method,
    
    -- Status: draft → finalized → voided
    status VARCHAR(20) DEFAULT 'draft',
    
    -- DTE tracking
    dte_numero_control, dte_codigo_generacion, 
    dte_sello_recibido, dte_status, dte_hacienda_response,
    
    -- Timestamps
    created_at, finalized_at, voided_at,
    created_by, notes
);
