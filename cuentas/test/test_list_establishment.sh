#!/bin/bash

# Script to list establishments and their points of sale

BASE_URL="http://localhost:8080/v1"

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

echo -e "${BLUE}=== List Establishments and Points of Sale ===${NC}\n"

# Get Company ID
read -p "Enter Company ID: " COMPANY_ID

if [ -z "$COMPANY_ID" ]; then
    echo -e "${RED}Company ID is required${NC}"
    exit 1
fi

COMPANY_HEADER="X-Company-ID: $COMPANY_ID"

# List all establishments
echo -e "${BLUE}Fetching Establishments...${NC}\n"
ESTABLISHMENTS_RESPONSE=$(curl -s -X GET "$BASE_URL/establishments?active_only=false" \
  -H "$COMPANY_HEADER")

ESTABLISHMENT_COUNT=$(echo "$ESTABLISHMENTS_RESPONSE" | jq '.count')

if [ "$ESTABLISHMENT_COUNT" = "null" ] || [ "$ESTABLISHMENT_COUNT" = "0" ]; then
    echo -e "${YELLOW}No establishments found for this company${NC}"
    exit 0
fi

echo -e "${GREEN}Found $ESTABLISHMENT_COUNT establishment(s)${NC}\n"

# Parse and display each establishment with its POS
echo "$ESTABLISHMENTS_RESPONSE" | jq -c '.establishments[]' | while read -r establishment; do
    EST_ID=$(echo "$establishment" | jq -r '.id')
    EST_NAME=$(echo "$establishment" | jq -r '.nombre')
    EST_TIPO=$(echo "$establishment" | jq -r '.tipo_establecimiento')
    EST_ACTIVE=$(echo "$establishment" | jq -r '.active')
    EST_DEPT=$(echo "$establishment" | jq -r '.departamento')
    EST_MUN=$(echo "$establishment" | jq -r '.municipio')
    EST_ADDRESS=$(echo "$establishment" | jq -r '.complemento_direccion')
    EST_PHONE=$(echo "$establishment" | jq -r '.telefono')
    COD_MH=$(echo "$establishment" | jq -r '.cod_establecimiento // "N/A"')
    COD_INTERNAL=$(echo "$establishment" | jq -r '.cod_establecimiento // "N/A"')
    
    # Determine establishment type name
    case $EST_TIPO in
        "01") TIPO_NAME="Sucursal" ;;
        "02") TIPO_NAME="Casa Matriz" ;;
        "04") TIPO_NAME="Bodega" ;;
        "07") TIPO_NAME="Patio" ;;
        "20") TIPO_NAME="Otro" ;;
        *) TIPO_NAME="Desconocido" ;;
    esac
    
    # Display establishment info
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}📍 $EST_NAME${NC}"
    echo -e "   ID: $EST_ID"
    echo -e "   Tipo: $TIPO_NAME ($EST_TIPO)"
    echo -e "   Estado: $([ "$EST_ACTIVE" = "true" ] && echo "✓ Activo" || echo "✗ Inactivo")"
    echo -e "   Código MH: $COD_MH"
    echo -e "   Código Interno: $COD_INTERNAL"
    echo -e "   Ubicación: Dept $EST_DEPT, Mun $EST_MUN"
    echo -e "   Dirección: $EST_ADDRESS"
    echo -e "   Teléfono: $EST_PHONE"
    
    # Fetch points of sale for this establishment
    POS_RESPONSE=$(curl -s -X GET "$BASE_URL/establishments/$EST_ID/pos?active_only=false" \
      -H "$COMPANY_HEADER")
    
    POS_COUNT=$(echo "$POS_RESPONSE" | jq '.count')
    
    if [ "$POS_COUNT" = "null" ] || [ "$POS_COUNT" = "0" ]; then
        echo -e "\n   ${YELLOW}No points of sale configured${NC}\n"
    else
        echo -e "\n   ${BLUE}Points of Sale ($POS_COUNT):${NC}"
        
        echo "$POS_RESPONSE" | jq -c '.points_of_sale[]' | while read -r pos; do
            POS_ID=$(echo "$pos" | jq -r '.id')
            POS_NAME=$(echo "$pos" | jq -r '.nombre')
            POS_ACTIVE=$(echo "$pos" | jq -r '.active')
            POS_PORTABLE=$(echo "$pos" | jq -r '.is_portable')
            POS_LAT=$(echo "$pos" | jq -r '.latitude // "N/A"')
            POS_LON=$(echo "$pos" | jq -r '.longitude // "N/A"')
            POS_COD_MH=$(echo "$pos" | jq -r '.cod_punto_venta // "N/A"')
            POS_COD_INTERNAL=$(echo "$pos" | jq -r '.cod_punto_venta // "N/A"')
            
            echo -e "   ├─ ${GREEN}🖥️  $POS_NAME${NC}"
            echo -e "   │  ID: $POS_ID"
            echo -e "   │  Estado: $([ "$POS_ACTIVE" = "true" ] && echo "✓ Activo" || echo "✗ Inactivo")"
            echo -e "   │  Tipo: $([ "$POS_PORTABLE" = "true" ] && echo "📱 Portátil" || echo "🏪 Fijo")"
            echo -e "   │  Código MH: $POS_COD_MH"
            echo -e "   │  Código Interno: $POS_COD_INTERNAL"
            
            if [ "$POS_PORTABLE" = "true" ] && [ "$POS_LAT" != "N/A" ]; then
                echo -e "   │  GPS: $POS_LAT, $POS_LON"
            fi
            echo ""
        done
    fi
done

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}Listing complete!${NC}\n"
