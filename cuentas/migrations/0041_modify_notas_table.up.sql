-- Fix client_tipo_contribuyente column size in notas_debito table
-- Change from VARCHAR(10) to VARCHAR(100) to match clients table

ALTER TABLE notas_debito 
ALTER COLUMN client_tipo_contribuyente TYPE VARCHAR(100);
