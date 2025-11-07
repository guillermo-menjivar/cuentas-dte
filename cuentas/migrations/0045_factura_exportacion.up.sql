-- Migration 0045: Add Factura de Exportación (Type 11) Support
-- Adds export-specific fields to invoices table and creates export documents table

-- ============================================
-- 1. Add Export Fields to Invoices Table
-- ============================================

-- Emisor export fields
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS export_tipo_item_expor INTEGER;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS export_recinto_fiscal VARCHAR(2);
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS export_regimen VARCHAR(13);

-- Resumen export fields
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS export_incoterms_code VARCHAR(10);
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS export_incoterms_desc VARCHAR(150);
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS export_seguro DECIMAL(12,2);
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS export_flete DECIMAL(12,2);
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS export_observaciones TEXT;

-- Receptor international fields (for Type 11 only)
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS export_receptor_cod_pais VARCHAR(4);
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS export_receptor_nombre_pais VARCHAR(50);
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS export_receptor_tipo_documento VARCHAR(2);
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS export_receptor_num_documento VARCHAR(20);
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS export_receptor_complemento VARCHAR(300);

-- Add comments for documentation
COMMENT ON COLUMN invoices.export_tipo_item_expor IS 'Export item type: 1=Goods, 2=Services, 3=Both (Type 11 only)';
COMMENT ON COLUMN invoices.export_recinto_fiscal IS 'Fiscal zone code (Type 11 only, nullable)';
COMMENT ON COLUMN invoices.export_regimen IS 'Export regime (Type 11 only, nullable)';
COMMENT ON COLUMN invoices.export_incoterms_code IS 'INCOTERMS code (FOB, CIF, etc.) for Type 11';
COMMENT ON COLUMN invoices.export_incoterms_desc IS 'INCOTERMS description for Type 11';
COMMENT ON COLUMN invoices.export_seguro IS 'Insurance amount for Type 11';
COMMENT ON COLUMN invoices.export_flete IS 'Freight amount for Type 11';
COMMENT ON COLUMN invoices.export_observaciones IS 'Export observations for Type 11';
COMMENT ON COLUMN invoices.export_receptor_cod_pais IS 'International country code (9300-9907) for Type 11';
COMMENT ON COLUMN invoices.export_receptor_nombre_pais IS 'Country name for Type 11';
COMMENT ON COLUMN invoices.export_receptor_tipo_documento IS 'International document type (36,13,02,03,37) for Type 11';
COMMENT ON COLUMN invoices.export_receptor_num_documento IS 'International document number for Type 11';
COMMENT ON COLUMN invoices.export_receptor_complemento IS 'Free-form international address for Type 11';

-- ============================================
-- 2. Create Export Documents Table
-- ============================================

CREATE TABLE IF NOT EXISTS invoice_export_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id VARCHAR(36) NOT NULL,
    
    -- Document classification
    cod_doc_asociado INTEGER NOT NULL CHECK (cod_doc_asociado IN (1, 2, 3, 4)),
    desc_documento VARCHAR(100),
    detalle_documento VARCHAR(300),
    
    -- Transport information (required if cod_doc_asociado = 4)
    placa_trans VARCHAR(70),
    modo_transp INTEGER CHECK (modo_transp IS NULL OR modo_transp IN (1, 2, 3, 4, 5, 6, 7)),
    num_conductor VARCHAR(100),
    nombre_conductor VARCHAR(200),
    
    -- Audit
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Foreign key
    CONSTRAINT fk_invoice_export_docs 
        FOREIGN KEY (invoice_id) 
        REFERENCES invoices(id) 
        ON DELETE CASCADE
);

-- Index for performance
CREATE INDEX idx_export_docs_invoice ON invoice_export_documents(invoice_id);

-- Add comments
COMMENT ON TABLE invoice_export_documents IS 'Export documentation for Type 11 invoices (customs, transport, etc.)';
COMMENT ON COLUMN invoice_export_documents.cod_doc_asociado IS '1=Export authorization, 2=Customs doc, 3=Other, 4=Bill of lading/Transport';
COMMENT ON COLUMN invoice_export_documents.desc_documento IS 'Document identification/number';
COMMENT ON COLUMN invoice_export_documents.detalle_documento IS 'Document description';
COMMENT ON COLUMN invoice_export_documents.placa_trans IS 'Transport vehicle identification';
COMMENT ON COLUMN invoice_export_documents.modo_transp IS 'Transport mode: 1=Maritime, 2=Air, 3=Road, 4=Rail, 5=Postal, 6=Multimodal, 7=Fixed transport';
COMMENT ON COLUMN invoice_export_documents.num_conductor IS 'Driver document number';
COMMENT ON COLUMN invoice_export_documents.nombre_conductor IS 'Driver name';

-- ============================================
-- 3. Add Constraints
-- ============================================

-- Constraint: Type 11 invoices must have export_tipo_item_expor
ALTER TABLE invoices ADD CONSTRAINT check_type_11_has_export_fields
CHECK (
    (dte_type != '11') OR 
    (dte_type = '11' AND export_tipo_item_expor IS NOT NULL)
);

-- Constraint: If tipo_item_expor = 2 (services), recinto & regimen must be null
ALTER TABLE invoices ADD CONSTRAINT check_services_export_fields
CHECK (
    (export_tipo_item_expor != 2) OR 
    (export_tipo_item_expor = 2 AND export_recinto_fiscal IS NULL AND export_regimen IS NULL)
);

-- Constraint: tipo_item_expor must be 1, 2, or 3
ALTER TABLE invoices ADD CONSTRAINT check_tipo_item_expor_values
CHECK (
    export_tipo_item_expor IS NULL OR 
    export_tipo_item_expor IN (1, 2, 3)
);

-- Constraint: Type 11 must have international receptor info
ALTER TABLE invoices ADD CONSTRAINT check_type_11_has_receptor
CHECK (
    (dte_type != '11') OR 
    (dte_type = '11' AND export_receptor_cod_pais IS NOT NULL)
);

-- Constraint: Transport documents (type 4) must have transport fields
ALTER TABLE invoice_export_documents ADD CONSTRAINT check_transport_doc_fields
CHECK (
    (cod_doc_asociado != 4) OR 
    (cod_doc_asociado = 4 AND 
     placa_trans IS NOT NULL AND 
     modo_transp IS NOT NULL AND 
     num_conductor IS NOT NULL AND 
     nombre_conductor IS NOT NULL)
);

-- Constraint: Non-transport docs must NOT have transport fields
ALTER TABLE invoice_export_documents ADD CONSTRAINT check_non_transport_fields_null
CHECK (
    (cod_doc_asociado = 4) OR 
    (cod_doc_asociado IN (1, 2, 3) AND 
     placa_trans IS NULL AND 
     modo_transp IS NULL AND 
     num_conductor IS NULL AND 
     nombre_conductor IS NULL)
);

-- Constraint: Authorization/customs docs (1,2) must have desc & detalle
ALTER TABLE invoice_export_documents ADD CONSTRAINT check_auth_doc_fields
CHECK (
    (cod_doc_asociado NOT IN (1, 2)) OR 
    (cod_doc_asociado IN (1, 2) AND 
     desc_documento IS NOT NULL AND 
     detalle_documento IS NOT NULL)
);

-- ============================================
-- 4. Add Indexes for Export Queries
-- ============================================

CREATE INDEX idx_invoices_export_tipo ON invoices(export_tipo_item_expor) 
    WHERE export_tipo_item_expor IS NOT NULL;

CREATE INDEX idx_invoices_export_country ON invoices(export_receptor_cod_pais) 
    WHERE export_receptor_cod_pais IS NOT NULL;

CREATE INDEX idx_invoices_dte_type_11 ON invoices(dte_type, created_at) 
    WHERE dte_type = '11';

-- ============================================
-- 5. Update existing check constraint for payment_method
-- ============================================

-- Drop old constraint if exists
ALTER TABLE invoices DROP CONSTRAINT IF EXISTS check_payment_method;

-- Add updated constraint (same as before, just recreating for clarity)
ALTER TABLE invoices ADD CONSTRAINT check_payment_method 
CHECK (
    payment_method IS NULL OR 
    payment_method IN ('01', '02', '03', '04', '05', '08', '09', '11', '12', '13', '14', '99')
);

-- ============================================
-- Migration Complete
-- ============================================

-- Summary of changes:
-- ✅ Added 13 export columns to invoices table
-- ✅ Created invoice_export_documents table
-- ✅ Added 7 constraints for data integrity
-- ✅ Added 3 indexes for performance
-- ✅ Added comprehensive comments

-- Next steps:
-- 1. Update models to include export fields
-- 2. Add export validation to InvoiceService
-- 3. Create DTE builder for Type 11
-- 4. Update API handlers
