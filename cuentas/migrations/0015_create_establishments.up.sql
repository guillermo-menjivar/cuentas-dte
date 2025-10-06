-- Establishments table
CREATE TABLE establishments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id),
    tipo_establecimiento VARCHAR(2) NOT NULL, -- 01,02,04,07,20
    nombre VARCHAR(100) NOT NULL,
    cod_establecimiento_mh VARCHAR(4), -- Assigned by MH
    cod_establecimiento VARCHAR(10),   -- Company's own code
    -- Address (for DTE emisor.direccion)
    departamento VARCHAR(2) NOT NULL,
    municipio VARCHAR(2) NOT NULL,
    complemento_direccion TEXT NOT NULL,
    telefono VARCHAR(30) NOT NULL,
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Point of Sale terminals
CREATE TABLE point_of_sale (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    establishment_id UUID NOT NULL REFERENCES establishments(id),
    nombre VARCHAR(100) NOT NULL,
    cod_punto_venta_mh VARCHAR(4),    -- Assigned by MH
    cod_punto_venta VARCHAR(15),      -- Company's own code
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW()
);

-- DTE sequences per POS+tipoDte
CREATE TABLE dte_sequences (
    point_of_sale_id UUID NOT NULL,
    tipo_dte VARCHAR(2) NOT NULL,
    last_sequence BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (point_of_sale_id, tipo_dte)
);
