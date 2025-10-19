-- Migration: Add Nota de Débito/Crédito support
-- Version: 001_add_nota_debito_support
-- Date: 2025-10-19

-- Add parent invoice type tracking
ALTER TABLE invoices 
ADD COLUMN parent_invoice_type VARCHAR(2);  -- '03' for CCF, '07' for Liquidación

COMMENT ON COLUMN invoices.parent_invoice_type IS 
'Type of parent document this invoice adjusts. Used for Nota de Débito/Crédito. Values: 03=CCF, 07=Liquidación';

-- Create related documents table
CREATE TABLE invoice_related_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    related_document_type VARCHAR(2) NOT NULL,              -- '03'=CCF, '07'=Liquidación
    related_document_generation_type INT NOT NULL,          -- 1=physical, 2=electronic
    related_document_number VARCHAR(36) NOT NULL,           -- UUID for electronic, physical number for paper
    related_document_date DATE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT chk_related_doc_type CHECK (related_document_type IN ('03', '07')),
    CONSTRAINT chk_related_gen_type CHECK (related_document_generation_type IN (1, 2)),
    CONSTRAINT chk_electronic_uuid CHECK (
        related_document_generation_type = 1 OR 
        related_document_number ~ '^[A-F0-9]{8}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{12}$'
    ),
    
    -- Ensure company_id matches invoice's company_id
    CONSTRAINT fk_company_invoice CHECK (
        company_id = (SELECT company_id FROM invoices WHERE id = invoice_id)
    ),
    
    -- Unique: One invoice cannot reference same document twice
    UNIQUE(company_id, invoice_id, related_document_number)
);

-- Indexes for performance
CREATE INDEX idx_related_docs_company_invoice ON invoice_related_documents(company_id, invoice_id);
CREATE INDEX idx_related_docs_invoice ON invoice_related_documents(invoice_id);
CREATE INDEX idx_related_docs_company ON invoice_related_documents(company_id);
CREATE INDEX idx_related_docs_number ON invoice_related_documents(related_document_number);

-- Comments for documentation
COMMENT ON TABLE invoice_related_documents IS 
'Stores references to parent documents for Nota de Débito/Crédito. Max 50 related docs per invoice, max 2000 total items.';

COMMENT ON COLUMN invoice_related_documents.related_document_type IS 
'Type of related document: 03=CCF, 07=Comprobante de Liquidación';

COMMENT ON COLUMN invoice_related_documents.related_document_generation_type IS 
'How document was generated: 1=Physical/Paper, 2=Electronic/Digital';

COMMENT ON COLUMN invoice_related_documents.related_document_number IS 
'UUID for electronic docs (type 2), physical number for paper docs (type 1)';

-- Add line item document reference (for linking items to specific related docs)
ALTER TABLE invoice_line_items 
ADD COLUMN related_document_ref VARCHAR(36);

COMMENT ON COLUMN invoice_line_items.related_document_ref IS 
'References which related document this line item adjusts. Used in Nota de Débito/Crédito.';

CREATE INDEX idx_line_items_related_ref ON invoice_line_items(related_document_ref);

-- Trigger to auto-update updated_at
CREATE OR REPLACE FUNCTION update_invoice_related_documents_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_invoice_related_documents_timestamp
    BEFORE UPDATE ON invoice_related_documents
    FOR EACH ROW
    EXECUTE FUNCTION update_invoice_related_documents_updated_at();
