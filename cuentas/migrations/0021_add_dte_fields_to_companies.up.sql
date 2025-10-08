-- Add DTE-related fields to companies table
ALTER TABLE companies 
ADD COLUMN cod_actividad VARCHAR(6),
ADD COLUMN desc_actividad VARCHAR(150),
ADD COLUMN nombre_comercial VARCHAR(150),
ADD COLUMN dte_ambiente VARCHAR(2) DEFAULT '00',
ADD COLUMN firmador_username VARCHAR(100),
ADD COLUMN firmador_password_ref VARCHAR(255);

-- Add comment explaining dte_ambiente values
COMMENT ON COLUMN companies.dte_ambiente IS '00 = test/sandbox, 01 = production';
