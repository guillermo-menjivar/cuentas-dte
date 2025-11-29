#!/bin/bash

BASE_URL="http://localhost:8080/v1"
CONTENT_TYPE="Content-Type: application/json"

GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

if [ "$#" -ne 1 ]; then
    echo -e "${RED}Error: Company ID is required${NC}"
    exit 1
fi

COMPANY_ID=$1
COMPANY_HEADER="X-Company-ID: $COMPANY_ID"

echo -e "${BLUE}=== Adding Establishments and POS ===${NC}"
echo -e "Company ID: ${YELLOW}$COMPANY_ID${NC}\n"

declare -a ESTABLISHMENT_IDS
declare -a POS_IDS

# ============================================
# ESTABLISHMENT 3: Sucursal San Miguel
# ============================================
echo -e "${BLUE}Creating Establishment: Sucursal San Miguel${NC}"
CREATE_EST3=$(curl -s -X POST "$BASE_URL/establishments" \
  -H "$CONTENT_TYPE" \
  -H "$COMPANY_HEADER" \
  -d '{
    "tipo_establecimiento": "01",
    "nombre": "Sucursal San Miguel",
    "cod_establecimiento": "0105",
    "departamento": "11",
    "municipio": "11",
    "complemento_direccion": "Avenida Roosevelt, Centro Comercial Plaza San Miguel, Local 23",
    "telefono": "26610123"
  }')

echo "$CREATE_EST3" | jq '.'
EST3_ID=$(echo "$CREATE_EST3" | jq -r '.id')
ESTABLISHMENT_IDS+=("$EST3_ID")

if [ "$EST3_ID" != "null" ] && [ -n "$EST3_ID" ]; then
    echo -e "${GREEN}✓ Sucursal San Miguel created: $EST3_ID${NC}"
    
    # Create POS 1 for San Miguel
    echo -e "${BLUE}  Creating POS 1 for San Miguel${NC}"
    POS1=$(curl -s -X POST "$BASE_URL/establishments/$EST3_ID/pos" \
      -H "$CONTENT_TYPE" \
      -H "$COMPANY_HEADER" \
      -d '{
        "nombre": "Caja Principal San Miguel",
        "cod_punto_venta": "0001",
        "is_portable": false
      }')
    echo "$POS1" | jq '.'
    POS1_ID=$(echo "$POS1" | jq -r '.id')
    POS_IDS+=("$POS1_ID")
    echo -e "${GREEN}  ✓ POS 1 created: $POS1_ID${NC}"
    
    # Create POS 2 for San Miguel
    echo -e "${BLUE}  Creating POS 2 for San Miguel${NC}"
    POS2=$(curl -s -X POST "$BASE_URL/establishments/$EST3_ID/pos" \
      -H "$CONTENT_TYPE" \
      -H "$COMPANY_HEADER" \
      -d '{
        "nombre": "Vendedor Ambulante",
        "cod_punto_venta": "0003",
        "latitude": 13.4833,
        "longitude": -88.1833,
        "is_portable": true
      }')
    echo "$POS2" | jq '.'
    POS2_ID=$(echo "$POS2" | jq -r '.id')
    POS_IDS+=("$POS2_ID")
    echo -e "${GREEN}  ✓ POS 2 created: $POS2_ID${NC}\n"
else
    echo -e "${RED}✗ Failed to create Sucursal San Miguel${NC}"
    echo "Response: $CREATE_EST3"
fi

# ============================================
# ESTABLISHMENT 4: Bodega Central
# ============================================
echo -e "${BLUE}Creating Establishment: Bodega Central${NC}"
CREATE_EST4=$(curl -s -X POST "$BASE_URL/establishments" \
  -H "$CONTENT_TYPE" \
  -H "$COMPANY_HEADER" \
  -d '{
    "tipo_establecimiento": "01",
    "nombre": "Bodega Central",
    "cod_establecimiento": "0201",
    "departamento": "06",
    "municipio": "01",
    "complemento_direccion": "Boulevard del Ejército, Zona Industrial, Bodega 12-A, Soyapango",
    "telefono": "22770456"
  }')

echo "$CREATE_EST4" | jq '.'
EST4_ID=$(echo "$CREATE_EST4" | jq -r '.id')
ESTABLISHMENT_IDS+=("$EST4_ID")

if [ "$EST4_ID" != "null" ] && [ -n "$EST4_ID" ]; then
    echo -e "${GREEN}✓ Bodega Central created: $EST4_ID${NC}"
    
    # Create POS 1 for Bodega
    echo -e "${BLUE}  Creating POS 1 for Bodega${NC}"
    POS3=$(curl -s -X POST "$BASE_URL/establishments/$EST4_ID/pos" \
      -H "$CONTENT_TYPE" \
      -H "$COMPANY_HEADER" \
      -d '{
        "nombre": "Control de Despacho",
        "cod_punto_venta": "0001",
        "is_portable": false
      }')
    echo "$POS3" | jq '.'
    POS3_ID=$(echo "$POS3" | jq -r '.id')
    POS_IDS+=("$POS3_ID")
    echo -e "${GREEN}  ✓ POS 1 created: $POS3_ID${NC}"
    
    # Create POS 2 for Bodega
    echo -e "${BLUE}  Creating POS 2 for Bodega${NC}"
    POS4=$(curl -s -X POST "$BASE_URL/establishments/$EST4_ID/pos" \
      -H "$CONTENT_TYPE" \
      -H "$COMPANY_HEADER" \
      -d '{
        "nombre": "Oficina Administrativa",
        "cod_punto_venta": "0002",
        "is_portable": false
      }')
    echo "$POS4" | jq '.'
    POS4_ID=$(echo "$POS4" | jq -r '.id')
    POS_IDS+=("$POS4_ID")
    echo -e "${GREEN}  ✓ POS 2 created: $POS4_ID${NC}\n"
else
    echo -e "${RED}✗ Failed to create Bodega Central${NC}"
fi

# ============================================
# ADD POS TO EXISTING SANTA ANA
# ============================================
echo -e "${BLUE}Adding POS to existing Sucursal Santa Ana${NC}"
SANTA_ANA_ID="19dafedc-2c0f-49e7-bf7a-b7a2b53ca013"

POS5=$(curl -s -X POST "$BASE_URL/establishments/$SANTA_ANA_ID/pos" \
  -H "$CONTENT_TYPE" \
  -H "$COMPANY_HEADER" \
  -d '{
    "nombre": "Caja Principal Santa Ana",
    "cod_punto_venta": "0001",
    "is_portable": false
  }')
echo "$POS5" | jq '.'
POS5_ID=$(echo "$POS5" | jq -r '.id')
POS_IDS+=("$POS5_ID")

if [ "$POS5_ID" != "null" ] && [ -n "$POS5_ID" ]; then
    echo -e "${GREEN}✓ POS added to Santa Ana: $POS5_ID${NC}"
    
    # Add POS 2 to Santa Ana
    POS6=$(curl -s -X POST "$BASE_URL/establishments/$SANTA_ANA_ID/pos" \
      -H "$CONTENT_TYPE" \
      -H "$COMPANY_HEADER" \
      -d '{
        "nombre": "Caja Secundaria Santa Ana",
        "cod_punto_venta": "0002",
        "is_portable": false
      }')
    echo "$POS6" | jq '.'
    POS6_ID=$(echo "$POS6" | jq -r '.id')
    POS_IDS+=("$POS6_ID")
    echo -e "${GREEN}✓ POS 2 added to Santa Ana: $POS6_ID${NC}\n"
else
    echo -e "${RED}✗ Failed to add POS to Santa Ana${NC}"
fi

echo -e "${BLUE}=== Summary ===${NC}"
echo -e "New Establishments: ${GREEN}$((${#ESTABLISHMENT_IDS[@]} - $(echo "${ESTABLISHMENT_IDS[@]}" | tr ' ' '\n' | grep -c "null")))${NC}"
echo -e "New POS: ${GREEN}$((${#POS_IDS[@]} - $(echo "${POS_IDS[@]}" | tr ' ' '\n' | grep -c "null")))${NC}"

echo -e "\n${GREEN}Setup complete! Now you have 4 establishments for testing.${NC}"
