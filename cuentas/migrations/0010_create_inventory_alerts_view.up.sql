-- Create view for inventory alerts
CREATE VIEW inventory_alerts AS
SELECT 
    i.id,
    i.company_id,
    i.sku,
    i.name,
    i.current_stock,
    i.minimum_stock,
    CASE 
        WHEN i.current_stock < 0 THEN 'OVERSOLD'
        WHEN i.current_stock <= i.minimum_stock THEN 'LOW_STOCK'
        ELSE 'OK'
    END as alert_status,
    ABS(i.current_stock) as oversold_quantity
FROM inventory_items i
WHERE i.track_inventory = true 
  AND i.active = true
  AND (i.current_stock < 0 OR i.current_stock <= i.minimum_stock);

COMMENT ON VIEW inventory_alerts IS 'View showing items with low stock or oversold status';
