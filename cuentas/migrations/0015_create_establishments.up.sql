CREATE TABLE establishments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    tipo_establecimiento VARCHAR(2) NOT NULL,
    nombre VARCHAR(100) NOT NULL,
    cod_establecimiento_mh VARCHAR(4), -- From MH, nullable until assigned
    cod_establecimiento VARCHAR(10),   -- Company's internal code
    -- Address for DTE
    departamento VARCHAR(2) NOT NULL,
    municipio VARCHAR(2) NOT NULL,
    complemento_direccion TEXT NOT NULL,
    telefono VARCHAR(30) NOT NULL,
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_establishments_company ON establishments(company_id);
CREATE INDEX idx_establishments_active ON establishments(company_id, active);
