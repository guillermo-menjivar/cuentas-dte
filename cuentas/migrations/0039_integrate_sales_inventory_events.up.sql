ALTER TABLE inventory_events
ADD COLUMN sale_price DECIMAL(15,2),              -- Unit sale price (before discount)
ADD COLUMN discount_amount DECIMAL(15,2),         -- Total discount applied (in dollars)
ADD COLUMN net_sale_price DECIMAL(15,2),          -- Price after discount, before tax
ADD COLUMN tax_exempt BOOLEAN DEFAULT false,      -- Is this sale tax exempt?
ADD COLUMN tax_rate DECIMAL(5,4),                 -- Tax rate applied (e.g., 0.1300 for 13%)
ADD COLUMN tax_amount DECIMAL(15,2),              -- IVA amount collected
ADD COLUMN invoice_id UUID,                       -- Reference to invoices table
ADD COLUMN invoice_line_id UUID,                  -- Reference to invoice_line_items table
ADD COLUMN customer_name VARCHAR(255),            -- Customer name (from invoice)
ADD COLUMN customer_nit VARCHAR(20),              -- Customer NIT (if available)
ADD COLUMN customer_tax_exempt BOOLEAN DEFAULT false; -- Is customer tax exempt (consumidor final)?
