-- Remove DTE-related fields from companies table
ALTER TABLE companies 
DROP COLUMN IF EXISTS cod_actividad,
DROP COLUMN IF EXISTS desc_actividad,
DROP COLUMN IF EXISTS nombre_comercial,
DROP COLUMN IF EXISTS dte_ambiente,
DROP COLUMN IF EXISTS firmador_username,
DROP COLUMN IF EXISTS firmador_password_ref;
