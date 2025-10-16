-- Create consumidor final client for each existing company
INSERT INTO clients (
    company_id,
    nit,
    ncr,
    business_name,
    legal_business_name,
    giro,
    tipo_contribuyente,
    tipo_persona,
    full_address,
    country_code,
    department_code,
    municipality_code,
    active
)
SELECT 
    id as company_id,
    '9999999999999' as nit,
    '999999' as ncr,  -- Added NCR
    'Consumidor Final' as business_name,
    'Consumidor Final' as legal_business_name,
    'Consumidor Final' as giro,
    'Consumidor Final' as tipo_contribuyente,
    '1' as tipo_persona,
    'N/A' as full_address,
    'SV' as country_code,
    '06' as department_code,
    '01' as municipality_code,
    true as active
FROM companies
ON CONFLICT DO NOTHING;

-- Create trigger to automatically create consumidor final for new companies
CREATE OR REPLACE FUNCTION create_consumidor_final_for_company()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO clients (
        company_id,
        dui,
        business_name,
        legal_business_name,
        giro,
        tipo_contribuyente,
        tipo_persona,
        full_address,
        country_code,
        department_code,
        municipality_code,
        active
    ) VALUES (
        NEW.id,
        999999999,  -- Generic DUI as bigint
        'Consumidor Final',
        'Consumidor Final',
        'Consumidor Final',
        'Consumidor Final',
        '1',
        'N/A',
        'SV',
        '06',
        '01',
        true
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
