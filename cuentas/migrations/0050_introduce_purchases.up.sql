-- Migration: Create purchases tables
-- Description: Generic purchases table to handle FSE (Type 14) and other purchase DTEs
-- Similar to invoices table but for purchase transactions

-- Create purchases table (generic for all purchase types)
CREATE TABLE purchases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    establishment_id UUID NOT NULL REFERENCES establishments(id),
    point_of_sale_id UUID NOT NULL REFERENCES points_of_sale(id),
    
    -- Purchase tracking
    purchase_number TEXT NOT NULL,
    purchase_type TEXT NOT NULL, -- 'fse', 'regular', 'import', 'other'
    purchase_date DATE NOT NULL,
    
    -- Supplier reference (NULL for FSE with embedded supplier info)
    supplier_id UUID REFERENCES clients(id), -- Reusing clients table for suppliers
    
    -- FSE-specific: Informal supplier info (when supplier_id is NULL)
    supplier_name TEXT,
    supplier_document_type TEXT, -- '36' (NIT), '13' (DUI), '37' (Otro), etc.
    supplier_document_number TEXT,
    supplier_nrc TEXT,
    supplier_activity_code TEXT,
    supplier_activity_desc TEXT,
    supplier_address_dept TEXT,
    supplier_address_muni TEXT,
    supplier_address_complement TEXT,
    supplier_phone TEXT,
    supplier_email TEXT,
    
    -- Financial amounts
    subtotal NUMERIC(12,2) NOT NULL,
    total_discount NUMERIC(12,2) DEFAULT 0,
    discount_percentage NUMERIC(5,2) DEFAULT 0,
    total_taxes NUMERIC(12,2) DEFAULT 0, -- For regular purchases with IVA
    iva_retained NUMERIC(12,2) DEFAULT 0, -- IVA retenido
    income_tax_retained NUMERIC(12,2) DEFAULT 0, -- Retención de renta
    total NUMERIC(12,2) NOT NULL,
    currency TEXT DEFAULT 'USD',
    
    -- Payment information
    payment_condition INT, -- 1=Contado, 2=Crédito, 3=Otro
    payment_method TEXT, -- '01'=Efectivo, '02'=Cheque, etc.
    payment_reference TEXT,
    payment_term TEXT, -- '01', '02', '03' (for credit payments)
    payment_period INT, -- Period in days/months
    payment_status TEXT DEFAULT 'pending', -- 'pending', 'paid', 'partial'
    amount_paid NUMERIC(12,2) DEFAULT 0,
    balance_due NUMERIC(12,2) DEFAULT 0,
    due_date DATE,
    
    -- DTE fields
    dte_numero_control TEXT UNIQUE,
    dte_status TEXT, -- 'pending', 'approved', 'rejected'
    dte_hacienda_response JSONB,
    dte_sello_recibido TEXT,
    dte_submitted_at TIMESTAMP WITH TIME ZONE,
    dte_type TEXT, -- '14' for FSE, future types
    
    -- Status tracking
    status TEXT DEFAULT 'draft', -- 'draft', 'finalized', 'voided'
    
    -- Audit fields
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    finalized_at TIMESTAMP WITH TIME ZONE,
    voided_at TIMESTAMP WITH TIME ZONE,
    created_by UUID REFERENCES users(id),
    voided_by UUID REFERENCES users(id),
    
    notes TEXT,
    
    -- Constraints
    CONSTRAINT purchases_status_check CHECK (status IN ('draft', 'finalized', 'voided')),
    CONSTRAINT purchases_type_check CHECK (purchase_type IN ('fse', 'regular', 'import', 'other')),
    CONSTRAINT purchases_payment_status_check CHECK (payment_status IN ('pending', 'paid', 'partial')),
    
    -- Business rule: FSE must have embedded supplier info, regular purchases must have supplier_id
    CONSTRAINT purchases_fse_supplier_check CHECK (
        (purchase_type = 'fse' AND supplier_id IS NULL AND supplier_name IS NOT NULL) OR
        (purchase_type != 'fse')
    )
);

-- Create purchase line items table
CREATE TABLE purchase_line_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    purchase_id UUID NOT NULL REFERENCES purchases(id) ON DELETE CASCADE,
    line_number INT NOT NULL,
    
    -- Item reference (NULL for FSE free-form items)
    item_id UUID REFERENCES inventory_items(id),
    
    -- Item information (required for FSE, can reference inventory for regular purchases)
    item_code TEXT,
    item_name TEXT NOT NULL,
    item_description TEXT,
    item_type INT NOT NULL, -- 1=Bien, 2=Servicio, 3=Ambos
    item_tipo_item TEXT, -- Maps to Hacienda tipo item codes
    unit_of_measure TEXT NOT NULL, -- Hacienda unit codes
    
    -- Quantities and pricing
    quantity NUMERIC(12,4) NOT NULL,
    unit_price NUMERIC(12,2) NOT NULL,
    
    -- Line amounts
    line_subtotal NUMERIC(12,2) NOT NULL,
    discount_percentage NUMERIC(5,2) DEFAULT 0,
    discount_amount NUMERIC(12,2) DEFAULT 0,
    taxable_amount NUMERIC(12,2) NOT NULL,
    total_taxes NUMERIC(12,2) DEFAULT 0,
    line_total NUMERIC(12,2) NOT NULL,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT purchase_line_items_unique_line UNIQUE (purchase_id, line_number)
);

-- Create purchase line item taxes table
CREATE TABLE purchase_line_item_taxes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    line_item_id UUID NOT NULL REFERENCES purchase_line_items(id) ON DELETE CASCADE,
    tributo_code TEXT NOT NULL,
    tributo_name TEXT NOT NULL,
    tax_rate NUMERIC(5,4) NOT NULL, -- 0.13 for 13% IVA
    taxable_base NUMERIC(12,2) NOT NULL,
    tax_amount NUMERIC(12,2) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for purchases
CREATE INDEX idx_purchases_company ON purchases(company_id);
CREATE INDEX idx_purchases_establishment ON purchases(establishment_id);
CREATE INDEX idx_purchases_point_of_sale ON purchases(point_of_sale_id);
CREATE INDEX idx_purchases_supplier ON purchases(supplier_id) WHERE supplier_id IS NOT NULL;
CREATE INDEX idx_purchases_type ON purchases(purchase_type);
CREATE INDEX idx_purchases_status ON purchases(status);
CREATE INDEX idx_purchases_date ON purchases(purchase_date);
CREATE INDEX idx_purchases_created_at ON purchases(created_at);
CREATE INDEX idx_purchases_dte_numero ON purchases(dte_numero_control) WHERE dte_numero_control IS NOT NULL;
CREATE INDEX idx_purchases_dte_type ON purchases(dte_type) WHERE dte_type IS NOT NULL;
CREATE INDEX idx_purchases_payment_status ON purchases(payment_status);

-- Create indexes for purchase line items
CREATE INDEX idx_purchase_line_items_purchase ON purchase_line_items(purchase_id);
CREATE INDEX idx_purchase_line_items_item ON purchase_line_items(item_id) WHERE item_id IS NOT NULL;

-- Create indexes for purchase line item taxes
CREATE INDEX idx_purchase_line_item_taxes_line ON purchase_line_item_taxes(line_item_id);
CREATE INDEX idx_purchase_line_item_taxes_tributo ON purchase_line_item_taxes(tributo_code);

-- Extend dte_commit_log to support purchases
ALTER TABLE dte_commit_log 
ADD COLUMN purchase_id UUID REFERENCES purchases(id);

-- Add index for purchase_id in commit log
CREATE INDEX idx_dte_commit_log_purchase ON dte_commit_log(purchase_id) WHERE purchase_id IS NOT NULL;

-- Add check constraint to ensure either invoice_id or purchase_id is set (not both)
ALTER TABLE dte_commit_log
ADD CONSTRAINT dte_commit_log_doc_reference_check CHECK (
    (invoice_id IS NOT NULL AND purchase_id IS NULL) OR
    (invoice_id IS NULL AND purchase_id IS NOT NULL) OR
     (invoice_id IS NULL AND purchase_id IS NULL)
);

-- Add comments for documentation
COMMENT ON TABLE purchases IS 'Generic purchases table supporting FSE (Type 14) and other purchase DTEs';
COMMENT ON COLUMN purchases.purchase_type IS 'Type of purchase: fse (Factura Sujeto Excluido), regular (standard purchase), import, other';
COMMENT ON COLUMN purchases.supplier_id IS 'Foreign key to clients table (reused for suppliers). NULL for FSE with embedded supplier info';
COMMENT ON COLUMN purchases.supplier_name IS 'For FSE: Informal supplier name (required when supplier_id is NULL)';
COMMENT ON COLUMN purchases.iva_retained IS 'IVA retained from supplier (if applicable)';
COMMENT ON COLUMN purchases.income_tax_retained IS 'Income tax retained from supplier (if applicable)';
COMMENT ON COLUMN purchases.dte_type IS 'DTE type code: 14 for FSE, others for future purchase DTE types';

COMMENT ON TABLE purchase_line_items IS 'Line items for purchases. Can reference inventory_items or be free-form (for FSE)';
COMMENT ON COLUMN purchase_line_items.item_id IS 'Reference to inventory_items. NULL for FSE free-form items';
COMMENT ON COLUMN purchase_line_items.item_type IS '1=Bien (goods), 2=Servicio (service), 3=Ambos (both)';

COMMENT ON TABLE purchase_line_item_taxes IS 'Tax breakdown for purchase line items';

COMMENT ON COLUMN dte_commit_log.purchase_id IS 'Reference to purchases table for purchase DTEs (Type 14 FSE, etc). Mutually exclusive with invoice_id';
