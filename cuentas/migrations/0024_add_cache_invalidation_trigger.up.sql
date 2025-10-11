-- Function to notify application when company credentials are updated
CREATE OR REPLACE FUNCTION notify_company_credentials_updated()
RETURNS TRIGGER AS $$
BEGIN
    -- Only notify if firmador credentials changed
    IF (OLD.firmador_password_ref IS DISTINCT FROM NEW.firmador_password_ref) THEN
        PERFORM pg_notify(
            'company_credentials_updated',
            json_build_object(
                'company_id', NEW.id::text,
                'action', 'invalidate_cache'
            )::text
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger on companies table
CREATE TRIGGER company_credentials_update_trigger
    AFTER UPDATE ON companies
    FOR EACH ROW
    EXECUTE FUNCTION notify_company_credentials_updated();
