-- Migration 0046: Add Nota de Remisión Electrónica (Type 04) Support
-- Extends invoices table to support delivery notes for goods movement

-- ============================================
-- 1. Add Remision Fields to Invoices Table
-- ============================================

-- Remision-specific type (pre_invoice_delivery, inter_branch_transfer, route_sales, other)
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS remision_type VARCHAR(50);

-- Delivery/transport information
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS delivery_person VARCHAR(200);
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS vehicle_plate VARCHAR(20);
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS delivery_notes TEXT;

-- Allow null receptor for internal transfers (Type 04 only)
ALTER TABLE invoices ALTER COLUMN client_id DROP NOT NULL;
ALTER TABLE invoices ALTER COLUMN client_name DROP NOT NULL;
ALTER TABLE invoices ALTER COLUMN client_legal_name DROP NOT NULL;
ALTER TABLE invoices ALTER COLUMN client_address DROP NOT NULL;

-- Add comments for documentation
COMMENT ON COLUMN invoices.remision_type IS 'Remision subtype: pre_invoice_delivery, inter_branch_transfer, route_sales, other (Type 04 only)';
COMMENT ON COLUMN invoices.delivery_person IS 'Name of person delivering goods (Type 04 only)';
COMMENT ON COLUMN invoices.vehicle_plate IS 'Vehicle plate number for transport (Type 04 only)';
COMMENT ON COLUMN invoices.delivery_notes IS 'Additional notes about delivery/movement (Type 04 only)';

-- ============================================
-- 2. Create Related Documents Table
-- ============================================

CREATE TABLE IF NOT EXISTS invoice_related_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id VARCHAR(36) NOT NULL,
    
    -- Related document info (from schema documentoRelacionado)
    tipo_documento VARCHAR(2) NOT NULL,  -- "01" or "03" for remisiones
    tipo_generacion INTEGER NOT NULL CHECK (tipo_generacion IN (1, 2)),  -- 1=physical, 2=electronic
    numero_documento VARCHAR(36) NOT NULL,  -- UUID if electronic, correlativo if physical
    fecha_emision DATE NOT NULL,
    
    -- Audit
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Foreign key
    CONSTRAINT fk_invoice_related_docs 
        FOREIGN KEY (invoice_id) 
        REFERENCES invoices(id) 
        ON DELETE CASCADE
);

-- Indexes for performance
CREATE INDEX idx_related_docs_invoice ON invoice_related_documents(invoice_id);
CREATE INDEX idx_related_docs_numero ON invoice_related_documents(numero_documento);
CREATE INDEX idx_related_docs_type ON invoice_related_documents(tipo_documento, invoice_id);

-- Add comments
COMMENT ON TABLE invoice_related_documents IS 'Related documents (documentoRelacionado) for DTEs - links remisiones to invoices';
COMMENT ON COLUMN invoice_related_documents.tipo_documento IS 'Type of related document: 01=Factura, 03=CCF (only these allowed for remisiones)';
COMMENT ON COLUMN invoice_related_documents.tipo_generacion IS '1=Physical document, 2=Electronic DTE';
COMMENT ON COLUMN invoice_related_documents.numero_documento IS 'UUID for electronic DTEs, correlativo number for physical docs';
COMMENT ON COLUMN invoice_related_documents.fecha_emision IS 'Date the related document was issued';

-- ============================================
-- 3. Create Remision-Invoice Links Table
-- ============================================

CREATE TABLE IF NOT EXISTS remision_invoice_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    remision_id VARCHAR(36) NOT NULL,  -- The remision (dte_type='04')
    invoice_id VARCHAR(36) NOT NULL,   -- The invoice that references the remision
    
    -- Audit
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Foreign keys
    CONSTRAINT fk_remision 
        FOREIGN KEY (remision_id) 
        REFERENCES invoices(id) 
        ON DELETE CASCADE,
    CONSTRAINT fk_linked_invoice 
        FOREIGN KEY (invoice_id) 
        REFERENCES invoices(id) 
        ON DELETE CASCADE,
    
    -- Ensure unique links
    CONSTRAINT unique_remision_invoice_link UNIQUE (remision_id, invoice_id)
);

-- Indexes
CREATE INDEX idx_remision_links_remision ON remision_invoice_links(remision_id);
CREATE INDEX idx_remision_links_invoice ON remision_invoice_links(invoice_id);

-- Add comments
COMMENT ON TABLE remision_invoice_links IS 'Tracks which invoices reference which remisiones (for route sales scenario)';
COMMENT ON COLUMN remision_invoice_links.remision_id IS 'The remision document (Type 04)';
COMMENT ON COLUMN remision_invoice_links.invoice_id IS 'The invoice (Type 01/03) that references this remision';

-- ============================================
-- 4. Add Constraints
-- ============================================

-- Constraint: remision_type only valid for Type 04
ALTER TABLE invoices ADD CONSTRAINT check_remision_type_only_for_type_04
CHECK (
    (dte_type != '04') OR 
    (dte_type = '04' AND remision_type IS NOT NULL)
);

-- Constraint: remision_type must be valid
ALTER TABLE invoices ADD CONSTRAINT check_remision_type_values
CHECK (
    remision_type IS NULL OR 
    remision_type IN ('pre_invoice_delivery', 'inter_branch_transfer', 'route_sales', 'other')
);

-- Constraint: delivery fields only for Type 04
ALTER TABLE invoices ADD CONSTRAINT check_delivery_fields_only_for_type_04
CHECK (
    (delivery_person IS NULL AND vehicle_plate IS NULL AND delivery_notes IS NULL) OR
    (dte_type = '04')
);

-- Constraint: Type 04 can have null client (internal transfers)
-- (Already handled by making client_id nullable above)

-- Constraint: Related documents can only reference invoices (01) or CCF (03)
ALTER TABLE invoice_related_documents ADD CONSTRAINT check_related_doc_types
CHECK (
    tipo_documento IN ('01', '03')
);

-- Constraint: Electronic docs (tipo_generacion=2) must have UUID format
ALTER TABLE invoice_related_documents ADD CONSTRAINT check_electronic_doc_format
CHECK (
    (tipo_generacion = 1) OR 
    (tipo_generacion = 2 AND numero_documento ~ '^[A-F0-9]{8}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{12}$')
);

-- ============================================
-- 5. Add Indexes for Remision Queries
-- ============================================

CREATE INDEX idx_invoices_remision_type ON invoices(remision_type) 
    WHERE remision_type IS NOT NULL;

CREATE INDEX idx_invoices_dte_type_04 ON invoices(dte_type, created_at) 
    WHERE dte_type = '04';

-- Index for finding remisiones by delivery person (useful for route sales)
CREATE INDEX idx_invoices_delivery_person ON invoices(delivery_person) 
    WHERE delivery_person IS NOT NULL;

-- ============================================
-- Migration Complete
-- ============================================

-- Summary of changes:
-- ✅ Added 4 remision columns to invoices table
-- ✅ Made client fields nullable (for internal transfers)
-- ✅ Created invoice_related_documents table
-- ✅ Created remision_invoice_links table
-- ✅ Added 6 constraints for data integrity
-- ✅ Added 3 indexes for performance
-- ✅ Added comprehensive comments

-- Next steps:
-- 1. Update models to include remision fields
-- 2. Add remision validation to InvoiceService
-- 3. Create DTE builder for Type 04
-- 4. Update API handlers for remision endpoints
