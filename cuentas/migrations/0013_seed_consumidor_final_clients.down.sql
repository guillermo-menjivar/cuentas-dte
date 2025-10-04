DROP TRIGGER IF EXISTS trigger_create_consumidor_final ON companies;
DROP FUNCTION IF EXISTS create_consumidor_final_for_company();
DELETE FROM clients WHERE nit = 9999999999999;
