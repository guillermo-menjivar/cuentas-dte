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

# Test 1: Create client with NIT and NCR (business entity)
echo -e "${GREEN}Test 1: Creating client with NIT and NCR - San Salvador Centro${NC}"
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

# Test 2: Create another client with NIT and NCR (different business)
echo -e "${GREEN}Test 2: Creating another client with NIT and NCR - San Salvador Este${NC}"
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
    "tipo_persona": "1",
    "department_code": "06",
    "municipality_code": "22"
  }' | jq '.'

echo -e "\n---\n"

# Test 3: Create client with DUI only (individual person)
echo -e "${GREEN}Test 3: Creating client with DUI only - San Salvador Norte${NC}"
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
    "tipo_persona": "2",
    "department_code": "06",
    "municipality_code": "20"
  }' | jq '.'

echo -e "\n---\n"

# Test 4: Create another client with DUI only
echo -e "${GREEN}Test 4: Creating another client with DUI only - San Salvador Sur${NC}"
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
    "department_code": "06",
    "tipo_persona": "1",
    "municipality_code": "24"
  }' | jq '.'

echo -e "\n---\n"

# Test 5: Create client with all three IDs (DUI, NIT, and NCR)
echo -e "${GREEN}Test 5: Creating client with DUI, NIT, and NCR - La Libertad Este${NC}"
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

# Test 6: Create client with DUI only (another individual)
echo -e "${GREEN}Test 6: Creating client with DUI only - Santa Ana Centro${NC}"
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
    "tipo_persona": "2",
    "department_code": "02",
    "municipality_code": "15"
  }' | jq '.'

echo -e "\n---\n"

# Test 7: Create large corporation with NIT and NCR - Usulután
echo -e "${GREEN}Test 7: Creating large corporation with NIT and NCR - Usulután Norte${NC}"
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

# Test 8: Error - Try to create with NCR only (should fail)
echo -e "${RED}Test 8: Error case - NCR without NIT (should fail)${NC}"
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
    "tipo_persona": "1",
    "department_code": "06",
    "municipality_code": "23"
  }' | jq '.'

echo -e "\n---\n"

# Test 9: Error - Try to create with NIT only (should fail)
echo -e "${RED}Test 9: Error case - NIT without NCR (should fail)${NC}"
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
    "tipo_persona": "1",
    "department_code": "06",
    "municipality_code": "23"
  }' | jq '.'

echo -e "\n---\n"

# Test 10: Error - Try to create without any ID (should fail)
echo -e "${RED}Test 10: Error case - No identification provided (should fail)${NC}"
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
    "department_code": "06",
    "municipality_code": "23"
  }' | jq '.'

echo -e "\n---\n"

# Test 11: Error - Invalid DUI format (should fail)
echo -e "${RED}Test 11: Error case - Invalid DUI format (should fail)${NC}"
curl -X POST "${BASE_URL}/v1/clients" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "dui": "12345678",
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

# Test 12: Error - Invalid NIT format (should fail)
echo -e "${RED}Test 12: Error case - Invalid NIT format (should fail)${NC}"
curl -X POST "${BASE_URL}/v1/clients" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "nit": "0614123456001-2",
    "ncr": "12345-6",
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

# Test 13: Error - Municipality doesn't belong to department (should fail)
echo -e "${RED}Test 13: Error case - San Salvador municipality (23) in Santa Ana department (02) (should fail)${NC}"
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
    "department_code": "02",
    "municipality_code": "23"
  }' | jq '.'

echo -e "\n---\n"

# Test 14: Error - La Libertad municipality in San Salvador department (should fail)
echo -e "${RED}Test 14: Error case - La Libertad Centro (24) in San Salvador department (06) (should fail)${NC}"
curl -X POST "${BASE_URL}/v1/clients" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "dui": "08901234-5",
    "business_name": "Test Business 2",
    "legal_business_name": "Test Business Legal Name 2",
    "giro": "Test giro",
    "tipo_contribuyente": "Test",
    "full_address": "Test Address",
    "country_code": "SV",
    "department_code": "06",
    "municipality_code": "25"
  }' | jq '.'

echo -e "\n---\n"

# Test 15: Error - Invalid municipality code (should fail)
echo -e "${RED}Test 15: Error case - Invalid municipality code (99) (should fail)${NC}"
curl -X POST "${BASE_URL}/v1/clients" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "dui": "09012345-6",
    "business_name": "Test Business 3",
    "legal_business_name": "Test Business Legal Name 3",
    "giro": "Test giro",
    "tipo_contribuyente": "Test",
    "full_address": "Test Address",
    "country_code": "SV",
    "department_code": "06",
    "municipality_code": "99"
  }' | jq '.'

echo -e "\n---\n"

# Test 16: Error - Invalid department code (should fail)
echo -e "${RED}Test 16: Error case - Invalid department code (99) (should fail)${NC}"
curl -X POST "${BASE_URL}/v1/clients" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "dui": "01234567-8",
    "business_name": "Test Business 4",
    "legal_business_name": "Test Business Legal Name 4",
    "giro": "Test giro",
    "tipo_contribuyente": "Test",
    "full_address": "Test Address",
    "country_code": "SV",
    "department_code": "99",
    "municipality_code": "23"
  }' | jq '.'

echo -e "\n${BLUE}=== Tests Complete ===${NC}\n"
