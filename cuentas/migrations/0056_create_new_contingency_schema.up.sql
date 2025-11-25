-- ============================================================================
-- New Contingency Schema - Matches final design
-- ============================================================================

-- 1. Create contingency_periods (no dependencies)
CREATE TABLE contingency_periods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    company_id UUID NOT NULL,
    establishment_id UUID NOT NULL,
    point_of_sale_id UUID NOT NULL,
    ambiente VARCHAR(2) NOT NULL CHECK (ambiente IN ('00', '01')),
    
    f_inicio DATE NOT NULL,
    h_inicio TIME NOT NULL,
    f_fin DATE,
    h_fin TIME,
    
    tipo_contingencia INT NOT NULL CHECK (tipo_contingencia BETWEEN 1 AND 5),
    motivo_contingencia TEXT,
    
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'reporting', 'completed')),
    processing BOOLEAN DEFAULT false,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Create contingency_events (depends on periods)
CREATE TABLE contingency_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contingency_period_id UUID NOT NULL REFERENCES contingency_periods(id) ON DELETE CASCADE,
    
    codigo_generacion VARCHAR(36) NOT NULL,
    
    company_id UUID NOT NULL,
    establishment_id UUID NOT NULL,
    point_of_sale_id UUID NOT NULL,
    ambiente VARCHAR(2) NOT NULL,
    
    event_json JSONB NOT NULL,
    event_signed TEXT NOT NULL,
    
    estado VARCHAR(20),
    sello_recibido TEXT,
    hacienda_response JSONB,
    
    submitted_at TIMESTAMPTZ,
    accepted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. Create lotes (depends on events)
CREATE TABLE lotes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contingency_event_id UUID NOT NULL REFERENCES contingency_events(id) ON DELETE CASCADE,
    
    codigo_lote VARCHAR(100),
    
    company_id UUID NOT NULL,
    establishment_id UUID NOT NULL,
    point_of_sale_id UUID NOT NULL,
    
    dte_count INT NOT NULL,
    
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'submitted', 'completed')),
    processing BOOLEAN DEFAULT false,
    
    hacienda_response JSONB,
    
    submitted_at TIMESTAMPTZ,
    last_polled_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Comments
COMMENT ON TABLE contingency_periods IS 'Tracks contingency periods (outage episodes) per POS';
COMMENT ON TABLE contingency_events IS 'Eventos de Contingencia submitted to Hacienda';
COMMENT ON TABLE lotes IS 'Batches of DTEs submitted during contingency recovery';
