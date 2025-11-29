#!/bin/bash

# Check if company_id was provided
if [ -z "$1" ]; then
    echo "Usage: ./test_clients.sh <company_id>"
    echo "Example: ./test_clients.sh 550e8400-e29b-41d4-a716-446655440000"
    exit 1
fi

# Set variables
BASE_URL="http://localhost:8080"
COMPANY_ID="$1"

# Color codes for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Client Creation Tests ===${NC}"
echo -e "Company ID: ${COMPANY_ID}\n"

# Test 1: Create client with NIT and NCR (business entity - Persona Jurídica)
echo -e "${GREEN}Test 1: Creating Persona Jurídica with NIT and NCR - San Salvador Centro${NC}"
curl -X POST "${BASE_URL}/v1/clients" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "nit": "0614-123456-001-2",
    "ncr": "12345-6",
    "business_name": "Comercial San Salvador S.A. de C.V.",
    "legal_business_name": "Comercial San Salvador Sociedad Anonima de Capital Variable",
    "giro": "Venta al por mayor de productos alimenticios",
    "tipo_contribuyente": "Gran Contribuyente",
    "full_address": "Av. La Revolucion, Edificio Torre Futura, Piso 5, San Salvador",
    "country_code": "SV",
    "tipo_persona": "2",
    "department_code": "06",
    "municipality_code": "23"
  }' | jq '.'

echo -e "\n---\n"

# Test 2: Create another Persona Jurídica with NIT and NCR
echo -e "${GREEN}Test 2: Creating Persona Jurídica with NIT and NCR - San Salvador Este${NC}"
curl -X POST "${BASE_URL}/v1/clients" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "nit": "1402-050689-102-5",
    "ncr": "234567-8",
    "business_name": "Inversiones El Progreso S.A.",
    "legal_business_name": "Inversiones El Progreso Sociedad Anonima",
    "giro": "Actividades de construccion",
    "tipo_contribuyente": "Mediano Contribuyente",
    "full_address": "Boulevard del Hipodromo #450, Colonia San Benito, San Salvador",
    "country_code": "SV",
    "tipo_persona": "2",
    "department_code": "06",
    "municipality_code": "22"
  }' | jq '.'

echo -e "\n---\n"

# Test 3: Create Persona Natural with DUI only
echo -e "${GREEN}Test 3: Creating Persona Natural with DUI only - San Salvador Norte${NC}"
curl -X POST "${BASE_URL}/v1/clients" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "dui": "03456789-0",
    "business_name": "Juan Carlos Martinez",
    "legal_business_name": "Juan Carlos Martinez Lopez",
    "giro": "Servicios profesionales de consultoria",
    "tipo_contribuyente": "Pequeño Contribuyente",
    "full_address": "Colonia Escalon, Calle Los Bambues #234, San Salvador",
    "country_code": "SV",
    "tipo_persona": "1",
    "department_code": "06",
    "municipality_code": "20"
  }' | jq '.'

echo -e "\n---\n"

# Test 4: Create another Persona Natural with DUI only
echo -e "${GREEN}Test 4: Creating Persona Natural with DUI only - San Salvador Sur${NC}"
curl -X POST "${BASE_URL}/v1/clients" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "dui": "04567890-1",
    "business_name": "Maria Elena Rodriguez",
    "legal_business_name": "Maria Elena Rodriguez de Gonzalez",
    "giro": "Comercio al por menor de ropa y accesorios",
    "tipo_contribuyente": "Pequeño Contribuyente",
    "full_address": "Mercado Central, Local 45, San Salvador",
    "country_code": "SV",
    "tipo_persona": "1",
    "department_code": "06",
    "municipality_code": "24"
  }' | jq '.'

echo -e "\n---\n"

# Test 5: Create Persona Natural with DUI, NIT, and NCR (professional with company registration)
echo -e "${GREEN}Test 5: Creating Persona Natural with DUI, NIT, and NCR - La Libertad Este${NC}"
curl -X POST "${BASE_URL}/v1/clients" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "dui": "05678901-2",
    "nit": "0614-987654-001-8",
    "ncr": "345678-9",
    "business_name": "Roberto Alvarez Consultores",
    "legal_business_name": "Roberto Antonio Alvarez Consultores",
    "giro": "Servicios de contabilidad y auditoria",
    "tipo_contribuyente": "Mediano Contribuyente",
    "full_address": "Centro Comercial Galerias, Local 102, Santa Tecla",
    "country_code": "SV",
    "tipo_persona": "1",
    "department_code": "05",
    "municipality_code": "26"
  }' | jq '.'

echo -e "\n---\n"

# Test 6: Create Persona Natural with DUI only
echo -e "${GREEN}Test 6: Creating Persona Natural with DUI only - Santa Ana Centro${NC}"
curl -X POST "${BASE_URL}/v1/clients" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "dui": "06789012-3",
    "business_name": "Ana Sofia Hernandez",
    "legal_business_name": "Ana Sofia Hernandez Melendez",
    "giro": "Servicios de peluqueria y tratamientos de belleza",
    "tipo_contribuyente": "Pequeño Contribuyente",
    "full_address": "Calle Principal, Santa Ana",
    "country_code": "SV",
    "tipo_persona": "1",
    "department_code": "02",
    "municipality_code": "15"
  }' | jq '.'

echo -e "\n---\n"

# Test 7: Create Persona Jurídica - large corporation
echo -e "${GREEN}Test 7: Creating Persona Jurídica with NIT and NCR - Usulután Norte${NC}"
curl -X POST "${BASE_URL}/v1/clients" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "nit": "0614-111222-103-4",
    "ncr": "456789-0",
    "business_name": "Distribuidora Nacional S.A. de C.V.",
    "legal_business_name": "Distribuidora Nacional Sociedad Anonima de Capital Variable",
    "giro": "Distribucion de productos farmaceuticos",
    "tipo_contribuyente": "Gran Contribuyente",
    "full_address": "Carretera Principal, Usulután",
    "country_code": "SV",
    "tipo_persona": "2",
    "department_code": "11",
    "municipality_code": "24"
  }' | jq '.'

echo -e "\n${BLUE}=== Error Case Tests ===${NC}\n"

# Test 8: Error - Missing tipo_persona (should fail)
echo -e "${RED}Test 8: Error case - Missing tipo_persona (should fail)${NC}"
curl -X POST "${BASE_URL}/v1/clients" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "dui": "07890123-4",
    "business_name": "Test Business",
    "legal_business_name": "Test Business Legal Name",
    "giro": "Test giro",
    "tipo_contribuyente": "Test",
    "full_address": "Test Address",
    "country_code": "SV",
    "department_code": "06",
    "municipality_code": "23"
  }' | jq '.'

echo -e "\n---\n"

# Test 9: Error - Invalid tipo_persona (should fail)
echo -e "${RED}Test 9: Error case - Invalid tipo_persona '3' (should fail)${NC}"
curl -X POST "${BASE_URL}/v1/clients" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "dui": "08901234-5",
    "business_name": "Test Business",
    "legal_business_name": "Test Business Legal Name",
    "giro": "Test giro",
    "tipo_contribuyente": "Test",
    "full_address": "Test Address",
    "country_code": "SV",
    "tipo_persona": "3",
    "department_code": "06",
    "municipality_code": "23"
  }' | jq '.'

echo -e "\n---\n"

# Test 10: Error - NCR without NIT (should fail)
echo -e "${RED}Test 10: Error case - NCR without NIT (should fail)${NC}"
curl -X POST "${BASE_URL}/v1/clients" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "ncr": "567890-1",
    "business_name": "Test Business",
    "legal_business_name": "Test Business Legal Name",
    "giro": "Test giro",
    "tipo_contribuyente": "Test",
    "full_address": "Test Address",
    "country_code": "SV",
    "tipo_persona": "2",
    "department_code": "06",
    "municipality_code": "23"
  }' | jq '.'

echo -e "\n---\n"

# Test 11: Error - NIT without NCR (should fail)
echo -e "${RED}Test 11: Error case - NIT without NCR (should fail)${NC}"
curl -X POST "${BASE_URL}/v1/clients" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "nit": "0614-123456-001-2",
    "business_name": "Test Business",
    "legal_business_name": "Test Business Legal Name",
    "giro": "Test giro",
    "tipo_contribuyente": "Test",
    "full_address": "Test Address",
    "country_code": "SV",
    "tipo_persona": "2",
    "department_code": "06",
    "municipality_code": "23"
  }' | jq '.'

echo -e "\n---\n"

# Test 12: Error - No identification (should fail)
echo -e "${RED}Test 12: Error case - No identification provided (should fail)${NC}"
curl -X POST "${BASE_URL}/v1/clients" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "business_name": "Test Business",
    "legal_business_name": "Test Business Legal Name",
    "giro": "Test giro",
    "tipo_contribuyente": "Test",
    "full_address": "Test Address",
    "country_code": "SV",
    "tipo_persona": "1",
    "department_code": "06",
    "municipality_code": "23"
  }' | jq '.'

echo -e "\n${BLUE}=== Tests Complete ===${NC}\n"
