ALTER TABLE companies
ADD COLUMN departamento VARCHAR(2),
ADD COLUMN municipio VARCHAR(2),
ADD COLUMN complemento_direccion VARCHAR(200),
ADD COLUMN telefono VARCHAR(30);

-- Add comments
COMMENT ON COLUMN companies.departamento IS 'Código de departamento (01-14) según catálogo MH';
COMMENT ON COLUMN companies.municipio IS 'Código de municipio (01-XX) según departamento';
COMMENT ON COLUMN companies.complemento_direccion IS 'Dirección completa del emisor';
COMMENT ON COLUMN companies.telefono IS 'Teléfono del emisor para DTE';
