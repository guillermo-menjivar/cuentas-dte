-- ============================================================================
-- Migration 004: Create Triggers for Contingency
-- Description: Database triggers for data consistency
-- ============================================================================

-- Updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply updated_at to periods
CREATE TRIGGER update_contingency_periods_updated_at
    BEFORE UPDATE ON contingency_periods
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Apply updated_at to lotes
CREATE TRIGGER update_lotes_updated_at
    BEFORE UPDATE ON lotes
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Ambiente consistency check
CREATE OR REPLACE FUNCTION check_ambiente_consistency()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.contingency_period_id IS NOT NULL THEN
        IF NOT EXISTS (
            SELECT 1 FROM contingency_periods 
            WHERE id = NEW.contingency_period_id 
            AND ambiente = NEW.ambiente
        ) THEN
            RAISE EXCEPTION 'Invoice ambiente (%) does not match period ambiente', NEW.ambiente;
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply ambiente check to invoices
CREATE TRIGGER check_invoice_ambiente
    BEFORE INSERT OR UPDATE ON invoices
    FOR EACH ROW
    WHEN (NEW.contingency_period_id IS NOT NULL)
    EXECUTE FUNCTION check_ambiente_consistency();
