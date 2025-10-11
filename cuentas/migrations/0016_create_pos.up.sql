CREATE TABLE point_of_sale (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    establishment_id UUID NOT NULL REFERENCES establishments(id) ON DELETE CASCADE,
    nombre VARCHAR(100) NOT NULL,
    cod_punto_venta VARCHAR(15),    
    latitude DECIMAL(10, 8),        -- GPS latitude for portable POS
    longitude DECIMAL(11, 8),       -- GPS longitude for portable POS
    is_portable BOOLEAN DEFAULT false,
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_pos_establishment ON point_of_sale(establishment_id);
CREATE INDEX idx_pos_active ON point_of_sale(establishment_id, active);
