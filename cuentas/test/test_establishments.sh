#!/bin/bash

# Test script for Establishment and Point of Sale endpoints
# Usage: ./test_establishments.sh <company_id>

BASE_URL="http://localhost:8080/v1"
CONTENT_TYPE="Content-Type: application/json"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo "jq is required but not installed. Install it with: brew install jq"
    exit 1
fi

# Check for company ID argument
if [ "$#" -ne 1 ]; then
    echo -e "${RED}Error: Company ID is required${NC}"
    echo "Usage: $0 <company_id>"
    echo "Example: $0 bda93a7d-45dd-4d62-823f-4213806ff68f"
    exit 1
fi

COMPANY_ID=$1

if [ -z "$COMPANY_ID" ]; then
    echo -e "${RED}Company ID cannot be empty${NC}"
    exit 1
fi

COMPANY_HEADER="X-Company-ID: $COMPANY_ID"

echo -e "${BLUE}=== Establishment and POS Testing ===${NC}"
echo -e "Company ID: ${YELLOW}$COMPANY_ID${NC}\n"

# Test 1: Create Establishment
echo -e "${BLUE}Test 1: Creating Establishment (Casa Matriz)${NC}"
CREATE_ESTABLISHMENT_RESPONSE=$(curl -s -X POST "$BASE_URL/establishments" \
  -H "$CONTENT_TYPE" \
  -H "$COMPANY_HEADER" \
  -d '{
    "tipo_establecimiento": "02",
    "nombre": "Casa Matriz - San Salvador",
    "cod_establecimiento": "0006",
    "departamento": "06",
    "municipio": "14",
    "complemento_direccion": "Colonia Escalón, Calle Principal #123, San Salvador",
    "telefono": "22501234"
  }')

echo "$CREATE_ESTABLISHMENT_RESPONSE" | jq '.'

ESTABLISHMENT_ID=$(echo "$CREATE_ESTABLISHMENT_RESPONSE" | jq -r '.id')

if [ "$ESTABLISHMENT_ID" != "null" ] && [ -n "$ESTABLISHMENT_ID" ]; then
    echo -e "${GREEN}✓ Establishment created successfully${NC}"
    echo -e "Establishment ID: $ESTABLISHMENT_ID\n"
else
    echo -e "${RED}✗ Failed to create establishment${NC}\n"
    exit 1
fi

# Test 2: Create another establishment (Sucursal)
echo -e "${BLUE}Test 2: Creating Establishment (Sucursal Santa Ana)${NC}"
CREATE_SUCURSAL_RESPONSE=$(curl -s -X POST "$BASE_URL/establishments" \
  -H "$CONTENT_TYPE" \
  -H "$COMPANY_HEADER" \
  -d '{
    "tipo_establecimiento": "01",
    "nombre": "Sucursal Santa Ana",
    "cod_establecimiento": "0002",
    "departamento": "13",
    "municipio": "01",
    "complemento_direccion": "Centro Comercial Metrocentro, Local 45, Santa Ana",
    "telefono": "24401234"
  }')

echo "$CREATE_SUCURSAL_RESPONSE" | jq '.'

SUCURSAL_ID=$(echo "$CREATE_SUCURSAL_RESPONSE" | jq -r '.id')

if [ "$SUCURSAL_ID" != "null" ] && [ -n "$SUCURSAL_ID" ]; then
    echo -e "${GREEN}✓ Sucursal created successfully${NC}"
    echo -e "Sucursal ID: $SUCURSAL_ID\n"
else
    echo -e "${RED}✗ Failed to create sucursal${NC}\n"
fi

# Test 3: List all establishments
echo -e "${BLUE}Test 3: Listing all Establishments${NC}"
LIST_ESTABLISHMENTS_RESPONSE=$(curl -s -X GET "$BASE_URL/establishments" \
  -H "$COMPANY_HEADER")

echo "$LIST_ESTABLISHMENTS_RESPONSE" | jq '.'
ESTABLISHMENT_COUNT=$(echo "$LIST_ESTABLISHMENTS_RESPONSE" | jq '.count')
echo -e "${GREEN}✓ Found $ESTABLISHMENT_COUNT establishments${NC}\n"

# Test 4: Get specific establishment
echo -e "${BLUE}Test 4: Getting Establishment Details${NC}"
GET_ESTABLISHMENT_RESPONSE=$(curl -s -X GET "$BASE_URL/establishments/$ESTABLISHMENT_ID" \
  -H "$COMPANY_HEADER")

echo "$GET_ESTABLISHMENT_RESPONSE" | jq '.'
echo -e "${GREEN}✓ Retrieved establishment details${NC}\n"

# Test 5: Update establishment (WITHOUT changing cod_establecimiento)
echo -e "${BLUE}Test 5: Updating Establishment (phone and address only)${NC}"
UPDATE_ESTABLISHMENT_RESPONSE=$(curl -s -X PATCH "$BASE_URL/establishments/$ESTABLISHMENT_ID" \
  -H "$CONTENT_TYPE" \
  -H "$COMPANY_HEADER" \
  -d '{
    "telefono": "22509999",
    "complemento_direccion": "Colonia Escalón, Calle Principal #123, Edificio A, San Salvador"
  }')

echo "$UPDATE_ESTABLISHMENT_RESPONSE" | jq '.'
echo -e "${GREEN}✓ Establishment updated${NC}\n"

# Test 6: Create Point of Sale for establishment
echo -e "${BLUE}Test 6: Creating Point of Sale (Fixed POS)${NC}"
CREATE_POS_RESPONSE=$(curl -s -X POST "$BASE_URL/establishments/$ESTABLISHMENT_ID/pos" \
  -H "$CONTENT_TYPE" \
  -H "$COMPANY_HEADER" \
  -d '{
    "nombre": "Caja Principal 1",
    "cod_punto_venta": "0004",
    "is_portable": false
  }')

echo "$CREATE_POS_RESPONSE" | jq '.'

POS_ID=$(echo "$CREATE_POS_RESPONSE" | jq -r '.id')

if [ "$POS_ID" != "null" ] && [ -n "$POS_ID" ]; then
    echo -e "${GREEN}✓ Point of Sale created successfully${NC}"
    echo -e "POS ID: $POS_ID\n"
else
    echo -e "${RED}✗ Failed to create POS${NC}\n"
fi

# Test 7: Create Portable POS with GPS
echo -e "${BLUE}Test 7: Creating Portable Point of Sale (with GPS)${NC}"
CREATE_PORTABLE_POS_RESPONSE=$(curl -s -X POST "$BASE_URL/establishments/$ESTABLISHMENT_ID/pos" \
  -H "$CONTENT_TYPE" \
  -H "$COMPANY_HEADER" \
  -d '{
    "nombre": "Food Truck - Mobile POS",
    "cod_punto_venta": "0017",
    "latitude": 13.6989,
    "longitude": -89.1914,
    "is_portable": true
  }')

echo "$CREATE_PORTABLE_POS_RESPONSE" | jq '.'

PORTABLE_POS_ID=$(echo "$CREATE_PORTABLE_POS_RESPONSE" | jq -r '.id')

if [ "$PORTABLE_POS_ID" != "null" ] && [ -n "$PORTABLE_POS_ID" ]; then
    echo -e "${GREEN}✓ Portable POS created successfully${NC}"
    echo -e "Portable POS ID: $PORTABLE_POS_ID\n"
else
    echo -e "${RED}✗ Failed to create portable POS${NC}\n"
fi

# Test 8: List all POS for establishment
echo -e "${BLUE}Test 8: Listing all Points of Sale for Establishment${NC}"
LIST_POS_RESPONSE=$(curl -s -X GET "$BASE_URL/establishments/$ESTABLISHMENT_ID/pos" \
  -H "$COMPANY_HEADER")

echo "$LIST_POS_RESPONSE" | jq '.'
POS_COUNT=$(echo "$LIST_POS_RESPONSE" | jq '.count')
echo -e "${GREEN}✓ Found $POS_COUNT points of sale${NC}\n"

# Test 9: Get specific POS
echo -e "${BLUE}Test 9: Getting Point of Sale Details${NC}"
GET_POS_RESPONSE=$(curl -s -X GET "$BASE_URL/pos/$POS_ID" \
  -H "$COMPANY_HEADER")

echo "$GET_POS_RESPONSE" | jq '.'
echo -e "${GREEN}✓ Retrieved POS details${NC}\n"

# Test 10: Update POS location (for portable POS)
if [ "$PORTABLE_POS_ID" != "null" ] && [ -n "$PORTABLE_POS_ID" ]; then
    echo -e "${BLUE}Test 10: Updating Portable POS Location${NC}"
    UPDATE_LOCATION_RESPONSE=$(curl -s -X PATCH "$BASE_URL/pos/$PORTABLE_POS_ID/location" \
      -H "$CONTENT_TYPE" \
      -H "$COMPANY_HEADER" \
      -d '{
        "latitude": 13.7001,
        "longitude": -89.1925
      }')

    echo "$UPDATE_LOCATION_RESPONSE" | jq '.'
    echo -e "${GREEN}✓ POS location updated${NC}\n"
fi

# Test 11: Update POS details (WITHOUT changing cod_punto_venta)
echo -e "${BLUE}Test 11: Updating Point of Sale (name only)${NC}"
UPDATE_POS_RESPONSE=$(curl -s -X PATCH "$BASE_URL/pos/$POS_ID" \
  -H "$CONTENT_TYPE" \
  -H "$COMPANY_HEADER" \
  -d '{
    "nombre": "Caja Principal 1 - Actualizada"
  }')

echo "$UPDATE_POS_RESPONSE" | jq '.'
echo -e "${GREEN}✓ POS updated${NC}\n"

# Test 12: List only active establishments
echo -e "${BLUE}Test 12: Listing Active Establishments Only${NC}"
LIST_ACTIVE_RESPONSE=$(curl -s -X GET "$BASE_URL/establishments?active_only=true" \
  -H "$COMPANY_HEADER")

echo "$LIST_ACTIVE_RESPONSE" | jq '.'
echo -e "${GREEN}✓ Retrieved active establishments${NC}\n"

# Summary
echo -e "${BLUE}=== Test Summary ===${NC}"
echo -e "Establishments created: 2"
echo -e "Points of Sale created: 2"
echo -e "Establishment ID: $ESTABLISHMENT_ID"
echo -e "Sucursal ID: $SUCURSAL_ID"
echo -e "Fixed POS ID: $POS_ID"
echo -e "Portable POS ID: $PORTABLE_POS_ID"
echo -e "\n${GREEN}All tests completed!${NC}"

# Optional: Cleanup prompt
echo -e "\n${BLUE}Do you want to deactivate the test data? (y/n)${NC}"
read -r CLEANUP

if [ "$CLEANUP" = "y" ]; then
    echo -e "${BLUE}Deactivating test POS...${NC}"

    curl -s -X DELETE "$BASE_URL/pos/$POS_ID" \
      -H "$COMPANY_HEADER" | jq '.'

    if [ "$PORTABLE_POS_ID" != "null" ]; then
        curl -s -X DELETE "$BASE_URL/pos/$PORTABLE_POS_ID" \
          -H "$COMPANY_HEADER" | jq '.'
    fi

    echo -e "${BLUE}Deactivating test establishments...${NC}"

    curl -s -X DELETE "$BASE_URL/establishments/$ESTABLISHMENT_ID" \
      -H "$COMPANY_HEADER" | jq '.'

    if [ "$SUCURSAL_ID" != "null" ]; then
        curl -s -X DELETE "$BASE_URL/establishments/$SUCURSAL_ID" \
          -H "$COMPANY_HEADER" | jq '.'
    fi

    echo -e "${GREEN}✓ Test data deactivated${NC}"
fi
