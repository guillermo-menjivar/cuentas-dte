-- Add CCF-required fields to clients table
ALTER TABLE clients 
ADD COLUMN cod_actividad VARCHAR(6),
ADD COLUMN desc_actividad VARCHAR(150),
ADD COLUMN nombre_comercial VARCHAR(150),
ADD COLUMN telefono VARCHAR(30),
ADD COLUMN correo VARCHAR(100);

-- Add index on correo for lookups
CREATE INDEX idx_clients_correo ON clients(correo) WHERE correo IS NOT NULL;

-- Add comment explaining these are for CCF (Crédito Fiscal)
COMMENT ON COLUMN clients.cod_actividad IS 'Código de actividad económica (required for CCF - tipo_persona=2)';
COMMENT ON COLUMN clients.desc_actividad IS 'Descripción de actividad económica (required for CCF - tipo_persona=2)';
COMMENT ON COLUMN clients.nombre_comercial IS 'Nombre comercial del cliente (optional)';
COMMENT ON COLUMN clients.telefono IS 'Número de teléfono del cliente';
COMMENT ON COLUMN clients.correo IS 'Correo electrónico (required for CCF - tipo_persona=2)';
