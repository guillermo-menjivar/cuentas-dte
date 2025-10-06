CREATE TABLE point_of_sale (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    establishment_id UUID NOT NULL REFERENCES establishments(id) ON DELETE CASCADE,
    nombre VARCHAR(100) NOT NULL,
    cod_punto_venta_mh VARCHAR(4),  -- From MH, nullable until assigned
    cod_punto_venta VARCHAR(15),    -- Company's internal code
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_pos_establishment ON point_of_sale(establishment_id);
CREATE INDEX idx_pos_active ON point_of_sale(establishment_id, active);
