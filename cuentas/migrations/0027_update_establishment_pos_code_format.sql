-- migrations/XXX_update_establishment_pos_code_format.sql

-- ============================================
-- Update establishments table
-- ============================================

-- Make cod_establecimiento NOT NULL
ALTER TABLE establishments 
ALTER COLUMN cod_establecimiento SET NOT NULL;

-- Add check constraint for format: M### or S### or B### or P###
ALTER TABLE establishments 
ADD CONSTRAINT chk_cod_establecimiento_format 
CHECK (cod_establecimiento ~ '^[MSBP][0-9]{3}$');

-- Add unique constraint (per company)
ALTER TABLE establishments 
ADD CONSTRAINT uq_establishments_company_code 
UNIQUE (company_id, cod_establecimiento);

-- ============================================
-- Update point_of_sale table
-- ============================================

-- Make cod_punto_venta NOT NULL
ALTER TABLE point_of_sale 
ALTER COLUMN cod_punto_venta SET NOT NULL;

-- Add check constraint for format: P###
ALTER TABLE point_of_sale 
ADD CONSTRAINT chk_cod_punto_venta_format 
CHECK (cod_punto_venta ~ '^P[0-9]{3}$');

-- Add unique constraint (per establishment)
ALTER TABLE point_of_sale 
ADD CONSTRAINT uq_pos_establishment_code 
UNIQUE (establishment_id, cod_punto_venta);

-- ============================================
-- Add comments for documentation
-- ============================================

COMMENT ON COLUMN establishments.cod_establecimiento IS 
'Establishment code format:
  M### = Matriz (Casa Matriz)
  S### = Sucursal (Branch)
  B### = Bodega (Warehouse)
  P### = Patio (Yard)
Examples: M001, S042, B003, P005';

COMMENT ON COLUMN point_of_sale.cod_punto_venta IS 
'Point of sale code in format P###. Example: P001, P042';
