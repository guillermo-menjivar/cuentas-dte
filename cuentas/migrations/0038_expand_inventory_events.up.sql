ALTER TABLE inventory_events
ADD COLUMN document_type VARCHAR(2),              -- "01" (Factura) or "03" (CCF)
ADD COLUMN document_number VARCHAR(100),          -- Document correlativo
ADD COLUMN supplier_name VARCHAR(255),            -- Provider name
ADD COLUMN supplier_nit VARCHAR(20),              -- Provider NIT (required for CCF)
ADD COLUMN supplier_nationality VARCHAR(50),      -- "Nacional" or "Extranjero"
ADD COLUMN cost_source_ref TEXT;                  -- Reference to purchase book/cost source

-- Add index for document lookups
CREATE INDEX IF NOT EXISTS idx_inventory_events_document 
ON inventory_events(company_id, document_type, document_number)
WHERE document_type IS NOT NULL;

-- Add comments
COMMENT ON COLUMN inventory_events.document_type IS 'Type of tax document: 01 (Factura) or 03 (CCF)';
COMMENT ON COLUMN inventory_events.document_number IS 'Document number from supplier';
COMMENT ON COLUMN inventory_events.supplier_name IS 'Name of supplier/provider';
COMMENT ON COLUMN inventory_events.supplier_nit IS 'NIT of supplier (required for CCF type 03)';
COMMENT ON COLUMN inventory_events.supplier_nationality IS 'Nacional or Extranjero (currently always Nacional)';
COMMENT ON COLUMN inventory_events.cost_source_ref IS 'Reference to purchase book or cost source (e.g., Libro de Compras Folio 45)';
