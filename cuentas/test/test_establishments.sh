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

# Arrays to store created IDs
declare -a ESTABLISHMENT_IDS
declare -a POS_IDS

# ============================================
# ESTABLISHMENT 1: Casa Matriz - San Salvador
# ============================================
echo -e "${BLUE}Creating Establishment 1: Casa Matriz - San Salvador${NC}"
CREATE_EST1=$(curl -s -X POST "$BASE_URL/establishments" \
  -H "$CONTENT_TYPE" \
  -H "$COMPANY_HEADER" \
  -d '{
    "tipo_establecimiento": "02",
    "nombre": "Casa Matriz - San Salvador",
    "cod_establecimiento": "M038",
    "departamento": "06",
    "municipio": "14",
    "complemento_direccion": "Colonia Escalón, Calle Principal #123, Edificio A, San Salvador",
    "telefono": "22509999"
  }')

EST1_ID=$(echo "$CREATE_EST1" | jq -r '.id')
ESTABLISHMENT_IDS+=("$EST1_ID")

if [ "$EST1_ID" != "null" ] && [ -n "$EST1_ID" ]; then
    echo -e "${GREEN}✓ Casa Matriz created: $EST1_ID${NC}"
    
    # Create 2 POS for Casa Matriz
    echo -e "${BLUE}  Creating POS 1 for Casa Matriz (Fixed)${NC}"
    POS1=$(curl -s -X POST "$BASE_URL/establishments/$EST1_ID/pos" \
      -H "$CONTENT_TYPE" \
      -H "$COMPANY_HEADER" \
      -d '{
        "nombre": "Caja Principal 1",
        "cod_punto_venta": "P004",
        "is_portable": false
      }')
    POS1_ID=$(echo "$POS1" | jq -r '.id')
    POS_IDS+=("$POS1_ID")
    echo -e "${GREEN}  ✓ POS created: $POS1_ID${NC}"
    
    echo -e "${BLUE}  Creating POS 2 for Casa Matriz (Portable)${NC}"
    POS2=$(curl -s -X POST "$BASE_URL/establishments/$EST1_ID/pos" \
      -H "$CONTENT_TYPE" \
      -H "$COMPANY_HEADER" \
      -d '{
        "nombre": "Food Truck - Mobile POS",
        "cod_punto_venta": "P019",
        "latitude": 13.6989,
        "longitude": -89.1914,
        "is_portable": true
      }')
    POS2_ID=$(echo "$POS2" | jq -r '.id')
    POS_IDS+=("$POS2_ID")
    echo -e "${GREEN}  ✓ POS created: $POS2_ID${NC}\n"
else
    echo -e "${RED}✗ Failed to create Casa Matriz${NC}\n"
fi

# ============================================
# ESTABLISHMENT 2: Sucursal Santa Ana
# ============================================
echo -e "${BLUE}Creating Establishment 2: Sucursal Santa Ana${NC}"
CREATE_EST2=$(curl -s -X POST "$BASE_URL/establishments" \
  -H "$CONTENT_TYPE" \
  -H "$COMPANY_HEADER" \
  -d '{
    "tipo_establecimiento": "01",
    "nombre": "Sucursal Santa Ana",
    "cod_establecimiento": "S084",
    "departamento": "13",
    "municipio": "01",
    "complemento_direccion": "Centro Comercial Metrocentro, Local 45, Santa Ana",
    "telefono": "24401234"
  }')

EST2_ID=$(echo "$CREATE_EST2" | jq -r '.id')
ESTABLISHMENT_IDS+=("$EST2_ID")

if [ "$EST2_ID" != "null" ] && [ -n "$EST2_ID" ]; then
    echo -e "${GREEN}✓ Sucursal Santa Ana created: $EST2_ID${NC}"
    
    # Create 2 POS for Santa Ana
    echo -e "${BLUE}  Creating POS 1 for Santa Ana (Fixed)${NC}"
    POS3=$(curl -s -X POST "$BASE_URL/establishments/$EST2_ID/pos" \
      -H "$CONTENT_TYPE" \
      -H "$COMPANY_HEADER" \
      -d '{
        "nombre": "Caja Principal Santa Ana",
        "cod_punto_venta": "P001",
        "is_portable": false
      }')
    POS3_ID=$(echo "$POS3" | jq -r '.id')
    POS_IDS+=("$POS3_ID")
    echo -e "${GREEN}  ✓ POS created: $POS3_ID${NC}"
    
    echo -e "${BLUE}  Creating POS 2 for Santa Ana (Fixed)${NC}"
    POS4=$(curl -s -X POST "$BASE_URL/establishments/$EST2_ID/pos" \
      -H "$CONTENT_TYPE" \
      -H "$COMPANY_HEADER" \
      -d '{
        "nombre": "Caja Secundaria Santa Ana",
        "cod_punto_venta": "P002",
        "is_portable": false
      }')
    POS4_ID=$(echo "$POS4" | jq -r '.id')
    POS_IDS+=("$POS4_ID")
    echo -e "${GREEN}  ✓ POS created: $POS4_ID${NC}\n"
else
    echo -e "${RED}✗ Failed to create Sucursal Santa Ana${NC}\n"
fi

# ============================================
# ESTABLISHMENT 3: Sucursal San Miguel
# ============================================
echo -e "${BLUE}Creating Establishment 3: Sucursal San Miguel${NC}"
CREATE_EST3=$(curl -s -X POST "$BASE_URL/establishments" \
  -H "$CONTENT_TYPE" \
  -H "$COMPANY_HEADER" \
  -d '{
    "tipo_establecimiento": "01",
    "nombre": "Sucursal San Miguel",
    "cod_establecimiento": "S105",
    "departamento": "11",
    "municipio": "11",
    "complemento_direccion": "Avenida Roosevelt, Centro Comercial Plaza San Miguel, Local 23",
    "telefono": "26610123"
  }')

EST3_ID=$(echo "$CREATE_EST3" | jq -r '.id')
ESTABLISHMENT_IDS+=("$EST3_ID")

if [ "$EST3_ID" != "null" ] && [ -n "$EST3_ID" ]; then
    echo -e "${GREEN}✓ Sucursal San Miguel created: $EST3_ID${NC}"
    
    # Create 2 POS for San Miguel
    echo -e "${BLUE}  Creating POS 1 for San Miguel (Fixed)${NC}"
    POS5=$(curl -s -X POST "$BASE_URL/establishments/$EST3_ID/pos" \
      -H "$CONTENT_TYPE" \
      -H "$COMPANY_HEADER" \
      -d '{
        "nombre": "Caja Principal San Miguel",
        "cod_punto_venta": "P001",
        "is_portable": false
      }')
    POS5_ID=$(echo "$POS5" | jq -r '.id')
    POS_IDS+=("$POS5_ID")
    echo -e "${GREEN}  ✓ POS created: $POS5_ID${NC}"
    
    echo -e "${BLUE}  Creating POS 2 for San Miguel (Portable)${NC}"
    POS6=$(curl -s -X POST "$BASE_URL/establishments/$EST3_ID/pos" \
      -H "$CONTENT_TYPE" \
      -H "$COMPANY_HEADER" \
      -d '{
        "nombre": "Vendedor Ambulante San Miguel",
        "cod_punto_venta": "P003",
        "latitude": 13.4833,
        "longitude": -88.1833,
        "is_portable": true
      }')
    POS6_ID=$(echo "$POS6" | jq -r '.id')
    POS_IDS+=("$POS6_ID")
    echo -e "${GREEN}  ✓ POS created: $POS6_ID${NC}\n"
else
    echo -e "${RED}✗ Failed to create Sucursal San Miguel${NC}\n"
fi

# ============================================
# ESTABLISHMENT 4: Bodega Central
# ============================================
echo -e "${BLUE}Creating Establishment 4: Bodega Central (Warehouse)${NC}"
CREATE_EST4=$(curl -s -X POST "$BASE_URL/establishments" \
  -H "$CONTENT_TYPE" \
  -H "$COMPANY_HEADER" \
  -d '{
    "tipo_establecimiento": "01",
    "nombre": "Bodega Central",
    "cod_establecimiento": "B001",
    "departamento": "06",
    "municipio": "01",
    "complemento_direccion": "Boulevard del Ejército, Zona Industrial, Bodega 12-A, Soyapango",
    "telefono": "22770456"
  }')

EST4_ID=$(echo "$CREATE_EST4" | jq -r '.id')
ESTABLISHMENT_IDS+=("$EST4_ID")

if [ "$EST4_ID" != "null" ] && [ -n "$EST4_ID" ]; then
    echo -e "${GREEN}✓ Bodega Central created: $EST4_ID${NC}"
    
    # Create 2 POS for Bodega
    echo -e "${BLUE}  Creating POS 1 for Bodega (Fixed)${NC}"
    POS7=$(curl -s -X POST "$BASE_URL/establishments/$EST4_ID/pos" \
      -H "$CONTENT_TYPE" \
      -H "$COMPANY_HEADER" \
      -d '{
        "nombre": "Control de Despacho",
        "cod_punto_venta": "P001",
        "is_portable": false
      }')
    POS7_ID=$(echo "$POS7" | jq -r '.id')
    POS_IDS+=("$POS7_ID")
    echo -e "${GREEN}  ✓ POS created: $POS7_ID${NC}"
    
    echo -e "${BLUE}  Creating POS 2 for Bodega (Fixed)${NC}"
    POS8=$(curl -s -X POST "$BASE_URL/establishments/$EST4_ID/pos" \
      -H "$CONTENT_TYPE" \
      -H "$COMPANY_HEADER" \
      -d '{
        "nombre": "Oficina Administrativa",
        "cod_punto_venta": "P002",
        "is_portable": false
      }')
    POS8_ID=$(echo "$POS8" | jq -r '.id')
    POS_IDS+=("$POS8_ID")
    echo -e "${GREEN}  ✓ POS created: $POS8_ID${NC}\n"
else
    echo -e "${RED}✗ Failed to create Bodega Central${NC}\n"
fi

# ============================================
# VERIFICATION
# ============================================
echo -e "${BLUE}=== Verification ===${NC}"
echo -e "${BLUE}Listing all establishments...${NC}"
LIST_ALL=$(curl -s -X GET "$BASE_URL/establishments" \
  -H "$COMPANY_HEADER")

TOTAL_EST=$(echo "$LIST_ALL" | jq '.count')
echo -e "${GREEN}✓ Total establishments: $TOTAL_EST${NC}"

echo "$LIST_ALL" | jq '.establishments[] | {id, nombre, cod_establecimiento, departamento, municipio}'

echo -e "\n${BLUE}Listing all points of sale...${NC}"
for EST_ID in "${ESTABLISHMENT_IDS[@]}"; do
    if [ "$EST_ID" != "null" ] && [ -n "$EST_ID" ]; then
        EST_NAME=$(echo "$LIST_ALL" | jq -r ".establishments[] | select(.id==\"$EST_ID\") | .nombre")
        echo -e "${YELLOW}POS for: $EST_NAME${NC}"
        LIST_POS=$(curl -s -X GET "$BASE_URL/establishments/$EST_ID/pos" \
          -H "$COMPANY_HEADER")
        echo "$LIST_POS" | jq '.points_of_sale[] | {id, nombre, cod_punto_venta, is_portable}'
        echo ""
    fi
done

# ============================================
# SUMMARY
# ============================================
echo -e "${BLUE}=== Summary ===${NC}"
echo -e "Establishments created: ${GREEN}${#ESTABLISHMENT_IDS[@]}${NC}"
echo -e "Points of Sale created: ${GREEN}${#POS_IDS[@]}${NC}"
echo -e "\nEstablishment IDs:"
for i in "${!ESTABLISHMENT_IDS[@]}"; do
    echo -e "  $((i+1)). ${ESTABLISHMENT_IDS[$i]}"
done
echo -e "\nPOS IDs:"
for i in "${!POS_IDS[@]}"; do
    echo -e "  $((i+1)). ${POS_IDS[$i]}"
done

echo -e "\n${GREEN}All tests completed!${NC}"

# ============================================
# CLEANUP PROMPT
# ============================================
echo -e "\n${BLUE}Do you want to deactivate all test data? (y/n)${NC}"
read -r CLEANUP

if [ "$CLEANUP" = "y" ]; then
    echo -e "${BLUE}Deactivating all test POS...${NC}"
    for POS_ID in "${POS_IDS[@]}"; do
        if [ "$POS_ID" != "null" ] && [ -n "$POS_ID" ]; then
            curl -s -X DELETE "$BASE_URL/pos/$POS_ID" -H "$COMPANY_HEADER" > /dev/null
            echo -e "${GREEN}  ✓ Deactivated POS: $POS_ID${NC}"
        fi
    done

    echo -e "${BLUE}Deactivating all test establishments...${NC}"
    for EST_ID in "${ESTABLISHMENT_IDS[@]}"; do
        if [ "$EST_ID" != "null" ] && [ -n "$EST_ID" ]; then
            curl -s -X DELETE "$BASE_URL/establishments/$EST_ID" -H "$COMPANY_HEADER" > /dev/null
            echo -e "${GREEN}  ✓ Deactivated establishment: $EST_ID${NC}"
        fi
    done

    echo -e "${GREEN}✓ All test data deactivated${NC}"
fi
