CREATE TABLE notas_credito_ccf_references (
    id UUID PRIMARY KEY,
    nota_credito_id UUID REFERENCES notas_credito,
    ccf_id UUID REFERENCES invoices(id),
    ccf_number VARCHAR(50),
    ccf_date DATE,
    
    UNIQUE(nota_credito_id, ccf_id)
);
