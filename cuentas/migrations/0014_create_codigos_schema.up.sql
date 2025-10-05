-- Create codigos schema
CREATE SCHEMA IF NOT EXISTS codigos;

-- Create tributos table
CREATE TABLE IF NOT EXISTS codigos.tributos (
    codigo VARCHAR(10) PRIMARY KEY,
    descripcion VARCHAR(255) NOT NULL,
    porcentaje DECIMAL(8,6) NOT NULL,
    tipo VARCHAR(50)
);

CREATE INDEX idx_tributos_codigo ON codigos.tributos(codigo);

COMMENT ON TABLE codigos.tributos IS 'Catalog of tax codes (tributos) from Ministerio de Hacienda';
COMMENT ON COLUMN codigos.tributos.porcentaje IS 'Tax percentage (e.g., 13.00 for 13%)';
