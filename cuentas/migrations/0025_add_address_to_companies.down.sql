-- migrations/XXX_add_company_address_fields.down.sql

ALTER TABLE companies
DROP COLUMN IF EXISTS departamento,
DROP COLUMN IF EXISTS municipio,
DROP COLUMN IF EXISTS complemento_direccion,
DROP COLUMN IF EXISTS telefono;
