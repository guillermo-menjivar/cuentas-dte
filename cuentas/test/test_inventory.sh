#!/bin/bash

# Check if company_id was provided
if [ -z "$1" ]; then
    echo "Usage: ./test_inventory.sh <company_id>"
    echo "Example: ./test_inventory.sh 550e8400-e29b-41d4-a716-446655440000"
    exit 1
fi

# Set variables
BASE_URL="http://localhost:8080"
COMPANY_ID="$1"

# Color codes for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Inventory Creation Tests ===${NC}"
echo -e "Company ID: ${COMPANY_ID}\n"

# Test 1: Create Product 1 - Laptop (using default taxes, auto-generate SKU/barcode)
echo -e "${GREEN}Test 1: Creating Product - Dell Laptop (auto-generated SKU/barcode)${NC}"
curl -X POST "${BASE_URL}/v1/inventory/items" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "tipo_item": "1",
    "name": "Dell Latitude 5520 Laptop",
    "description": "15.6 inch business laptop, Intel i7, 16GB RAM, 512GB SSD",
    "manufacturer": "Dell",
    "unit_price": 1200.00,
    "cost_price": 950.00,
    "unit_of_measure": "unidad",
    "color": "Black"
  }' | jq '.'

echo -e "\n---\n"

# Test 2: Create Product 2 - Mouse (with custom SKU)
echo -e "${GREEN}Test 2: Creating Product - Logitech Mouse (custom SKU)${NC}"
curl -X POST "${BASE_URL}/v1/inventory/items" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "tipo_item": "1",
    "sku": "MOUSE-LOG-001",
    "name": "Logitech MX Master 3",
    "description": "Wireless ergonomic mouse with precision scrolling",
    "manufacturer": "Logitech",
    "unit_price": 99.99,
    "cost_price": 65.00,
    "unit_of_measure": "unidad",
    "color": "Graphite"
  }' | jq '.'

echo -e "\n---\n"

# Test 3: Create Service 1 - IT Consulting
echo -e "${GREEN}Test 3: Creating Service - IT Consulting (default taxes)${NC}"
curl -X POST "${BASE_URL}/v1/inventory/items" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "tipo_item": "2",
    "name": "IT Consulting Services",
    "description": "Professional IT consulting and system architecture design",
    "unit_price": 150.00,
    "unit_of_measure": "hora"
  }' | jq '.'

echo -e "\n---\n"

# Test 4: Create Service 2 - Software Installation
echo -e "${GREEN}Test 4: Creating Service - Software Installation${NC}"
curl -X POST "${BASE_URL}/v1/inventory/items" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "tipo_item": "2",
    "sku": "SRV-INSTALL-001",
    "name": "Software Installation and Configuration",
    "description": "Professional software installation, setup, and training",
    "unit_price": 75.00,
    "unit_of_measure": "servicio"
  }' | jq '.'

echo -e "\n---\n"

# Test 5: Create Beer with additional tax (manually add alcohol tax after creation)
echo -e "${GREEN}Test 5: Creating Product - Pilsener Beer (will add alcohol tax)${NC}"
BEER_RESPONSE=$(curl -s -X POST "${BASE_URL}/v1/inventory/items" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "tipo_item": "1",
    "name": "Pilsener Beer 12oz",
    "description": "Premium lager beer, 12 oz bottle",
    "manufacturer": "Cervecería La Constancia",
    "unit_price": 1.50,
    "cost_price": 0.85,
    "unit_of_measure": "unidad"
  }')

echo "$BEER_RESPONSE" | jq '.'

# Extract beer item ID
BEER_ID=$(echo "$BEER_RESPONSE" | jq -r '.id')

if [ "$BEER_ID" != "null" ] && [ -n "$BEER_ID" ]; then
    echo -e "\n${YELLOW}Adding alcohol tax to beer (item ID: $BEER_ID)${NC}"
    
    # Add alcohol tax (you'll need to use a valid tributo code for alcohol tax from your codigos)
    # Using S3.A7 as example - replace with actual alcohol tax code if different
    curl -X POST "${BASE_URL}/v1/inventory/items/${BEER_ID}/taxes" \
      -H "Content-Type: application/json" \
      -H "X-Company-ID: ${COMPANY_ID}" \
      -d '{
        "tributo_code": "S3.A7"
      }' | jq '.'
else
    echo -e "${RED}Failed to create beer item, skipping tax addition${NC}"
fi

echo -e "\n---\n"

# Test 6: List all items
echo -e "${GREEN}Test 6: Listing all inventory items${NC}"
curl -X GET "${BASE_URL}/v1/inventory/items" \
  -H "X-Company-ID: ${COMPANY_ID}" | jq '.'

echo -e "\n---\n"

# Test 7: Filter by tipo_item (only products)
echo -e "${GREEN}Test 7: Listing only products (tipo_item=1)${NC}"
curl -X GET "${BASE_URL}/v1/inventory/items?tipo_item=1" \
  -H "X-Company-ID: ${COMPANY_ID}" | jq '.'

echo -e "\n---\n"

# Test 8: Filter by tipo_item (only services)
echo -e "${GREEN}Test 8: Listing only services (tipo_item=2)${NC}"
curl -X GET "${BASE_URL}/v1/inventory/items?tipo_item=2" \
  -H "X-Company-ID: ${COMPANY_ID}" | jq '.'

echo -e "\n---\n"

# Test 9: Get taxes for beer item
if [ "$BEER_ID" != "null" ] && [ -n "$BEER_ID" ]; then
    echo -e "${GREEN}Test 9: Getting all taxes for beer item${NC}"
    curl -X GET "${BASE_URL}/v1/inventory/items/${BEER_ID}/taxes" \
      -H "X-Company-ID: ${COMPANY_ID}" | jq '.'
fi

echo -e "\n${BLUE}=== Error Case Tests ===${NC}\n"

# Test 10: Try to create with invalid tipo_item
echo -e "${RED}Test 10: Error case - Invalid tipo_item (should fail)${NC}"
curl -X POST "${BASE_URL}/v1/inventory/items" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "tipo_item": "5",
    "name": "Invalid Item",
    "unit_price": 10.00,
    "unit_of_measure": "unidad"
  }' | jq '.'

echo -e "\n---\n"

# Test 11: Try to create with invalid unit_of_measure
echo -e "${RED}Test 11: Error case - Invalid unit_of_measure (should fail)${NC}"
curl -X POST "${BASE_URL}/v1/inventory/items" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "tipo_item": "1",
    "name": "Test Product",
    "unit_price": 10.00,
    "unit_of_measure": "invalid_unit"
  }' | jq '.'

echo -e "\n---\n"

# Test 12: Try to add invalid tax code
if [ "$BEER_ID" != "null" ] && [ -n "$BEER_ID" ]; then
    echo -e "${RED}Test 12: Error case - Invalid tributo_code (should fail)${NC}"
    curl -X POST "${BASE_URL}/v1/inventory/items/${BEER_ID}/taxes" \
      -H "Content-Type: application/json" \
      -H "X-Company-ID: ${COMPANY_ID}" \
      -d '{
        "tributo_code": "INVALID-TAX"
      }' | jq '.'
fi

echo -e "\n---\n"

# Test 13: Try to create duplicate SKU
echo -e "${RED}Test 13: Error case - Duplicate SKU (should fail)${NC}"
curl -X POST "${BASE_URL}/v1/inventory/items" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "tipo_item": "1",
    "sku": "MOUSE-LOG-001",
    "name": "Duplicate Mouse",
    "unit_price": 50.00,
    "unit_of_measure": "unidad"
  }' | jq '.'

echo -e "\n${BLUE}=== Tests Complete ===${NC}\n"
