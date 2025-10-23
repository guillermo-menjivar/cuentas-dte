-- =====================================================
-- Migration 35 UP: Add CQRS tables for inventory cost tracking
-- =====================================================

-- Create inventory_events table (Event Log - Write Only)
CREATE TABLE IF NOT EXISTS inventory_events (
    event_id BIGSERIAL PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    item_id UUID NOT NULL REFERENCES inventory_items(id) ON DELETE CASCADE,
    
    -- Event details
    event_type VARCHAR(50) NOT NULL,
    event_timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    aggregate_version INT NOT NULL,
    
    -- Transaction data
    quantity DECIMAL(15,4) NOT NULL,
    unit_cost DECIMAL(15,4) NOT NULL,
    total_cost DECIMAL(15,2) NOT NULL,
    
    -- State tracking (for cost history)
    balance_quantity_after DECIMAL(15,4) NOT NULL,
    balance_total_cost_after DECIMAL(15,2) NOT NULL,
    moving_avg_cost_before DECIMAL(15,4) NOT NULL,
    moving_avg_cost_after DECIMAL(15,4) NOT NULL,
    
    -- References
    reference_type VARCHAR(50),
    reference_id UUID,
    correlation_id UUID,
    
    -- Complete context
    event_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    
    -- Audit
    notes TEXT,
    created_by_user_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT check_event_type_valid CHECK (
        event_type IN ('PURCHASE', 'SALE', 'RETURN', 'ADJUSTMENT', 'INITIAL')
    ),
    CONSTRAINT check_quantity_positive CHECK (quantity > 0),
    CONSTRAINT check_balance_non_negative CHECK (balance_quantity_after >= 0),
    CONSTRAINT check_balance_cost_non_negative CHECK (balance_total_cost_after >= 0),
    CONSTRAINT unique_aggregate_version UNIQUE (company_id, item_id, aggregate_version)
);

-- Indexes for inventory_events
CREATE INDEX idx_inventory_events_company_item ON inventory_events(company_id, item_id, created_at DESC);
CREATE INDEX idx_inventory_events_company_type ON inventory_events(company_id, event_type, created_at DESC);
CREATE INDEX idx_inventory_events_reference ON inventory_events(reference_type, reference_id);
CREATE INDEX idx_inventory_events_created_at ON inventory_events(created_at DESC);
CREATE INDEX idx_inventory_events_correlation ON inventory_events(correlation_id) WHERE correlation_id IS NOT NULL;

-- Create inventory_state table (Current Snapshot - Read Optimized)
CREATE TABLE IF NOT EXISTS inventory_state (
    company_id UUID NOT NULL,
    item_id UUID NOT NULL,
    
    -- Current state
    current_quantity DECIMAL(15,4) NOT NULL DEFAULT 0,
    current_total_cost DECIMAL(15,2) NOT NULL DEFAULT 0,
    current_avg_cost DECIMAL(15,4) GENERATED ALWAYS AS (
        CASE 
            WHEN current_quantity > 0 
            THEN current_total_cost / current_quantity
            ELSE 0
        END
    ) STORED,
    
    -- Metadata
    last_event_id BIGINT,
    aggregate_version INT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Primary key
    PRIMARY KEY (company_id, item_id),
    
    -- Foreign keys
    CONSTRAINT fk_inventory_state_company FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE,
    CONSTRAINT fk_inventory_state_item FOREIGN KEY (item_id) REFERENCES inventory_items(id) ON DELETE CASCADE,
    
    -- Constraints
    CONSTRAINT check_state_quantity_non_negative CHECK (current_quantity >= 0),
    CONSTRAINT check_state_cost_non_negative CHECK (current_total_cost >= 0)
);

-- Indexes for inventory_state
CREATE INDEX idx_inventory_state_company ON inventory_state(company_id);
CREATE INDEX idx_inventory_state_in_stock ON inventory_state(company_id, current_quantity) WHERE current_quantity > 0;
CREATE INDEX idx_inventory_state_updated ON inventory_state(company_id, updated_at DESC);

-- Add comment explaining the tables
COMMENT ON TABLE inventory_events IS 'Immutable event log for all inventory movements. Never UPDATE or DELETE from this table.';
COMMENT ON TABLE inventory_state IS 'Current inventory snapshot derived from events. Optimized for fast queries.';
COMMENT ON COLUMN inventory_state.current_avg_cost IS 'Moving average cost auto-calculated from current_total_cost / current_quantity';
