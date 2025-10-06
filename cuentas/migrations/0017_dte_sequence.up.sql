CREATE TABLE dte_sequences (
    point_of_sale_id UUID NOT NULL REFERENCES point_of_sale(id) ON DELETE CASCADE,
    tipo_dte VARCHAR(2) NOT NULL,
    last_sequence BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (point_of_sale_id, tipo_dte)
);
