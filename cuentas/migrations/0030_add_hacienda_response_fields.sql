ALTER TABLE invoices ADD COLUMN dte_sello_recibido VARCHAR(100);
ALTER TABLE invoices ADD COLUMN dte_fecha_procesamiento TIMESTAMP WITH TIME ZONE;
ALTER TABLE invoices ADD COLUMN dte_observaciones TEXT[];

COMMENT ON COLUMN invoices.dte_sello_recibido IS 'Digital seal received from Hacienda after successful processing';
COMMENT ON COLUMN invoices.dte_fecha_procesamiento IS 'Timestamp when Hacienda processed the DTE';
COMMENT ON COLUMN invoices.dte_observaciones IS 'Observations/warnings returned by Hacienda';
