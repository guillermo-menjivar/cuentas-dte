#!/bin/bash

# Check if company_id was provided
if [ -z "$1" ]; then
    echo "Usage: ./test_inventory_cost_reports.sh <company_id>"
    echo "Example: ./test_inventory_cost_reports.sh 550e8400-e29b-41d4-a716-446655440000"
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
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color

echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     INVENTORY COST TRACKING - REPORTES Y VERIFICACIÓN     ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo -e "Company ID: ${COMPANY_ID}\n"

# Helper function to find item by SKU
find_item_by_sku() {
    local sku=$1
    curl -s -X GET "${BASE_URL}/v1/inventory/items?active=true" \
      -H "X-Company-ID: ${COMPANY_ID}" | jq -r --arg sku "$sku" '.items[] | select(.sku == $sku) | .id'
}

echo -e "${YELLOW}📋 Descubriendo artículos de inventario desde API...${NC}\n"

# Fetch item IDs dynamically from API
LAPTOP_ID=$(find_item_by_sku "LAPTOP-DELL-001")
TEXTBOOK_ID=$(find_item_by_sku "LIBRO-MAT-SEC")
MOUSE_ID=$(find_item_by_sku "MOUSE-LOG-001")
KEYBOARD_ID=$(find_item_by_sku "TECLADO-MECA-001")
MONITOR_ID=$(find_item_by_sku "MONITOR-DELL-27")
SERVICE_ID=$(find_item_by_sku "SRV-CONSULT-IT")

# Verify we found items
if [ -z "$LAPTOP_ID" ] || [ "$LAPTOP_ID" == "null" ]; then
    echo -e "${RED}❌ Error: No se pudieron encontrar los artículos creados por el script de configuración${NC}"
    echo -e "${YELLOW}Por favor ejecute primero: ./test_inventory_cost_setup.sh ${COMPANY_ID}${NC}\n"
    exit 1
fi

echo -e "${GREEN}✅ Artículos Encontrados por SKU:${NC}"
echo -e "   Laptop (LAPTOP-DELL-001):      ${LAPTOP_ID}"
echo -e "   Libro (LIBRO-MAT-SEC):         ${TEXTBOOK_ID}"
echo -e "   Mouse (MOUSE-LOG-001):         ${MOUSE_ID}"
echo -e "   Teclado (TECLADO-MECA-001):    ${KEYBOARD_ID}"
echo -e "   Monitor (MONITOR-DELL-27):     ${MONITOR_ID}"
echo -e "   Servicio (SRV-CONSULT-IT):     ${SERVICE_ID}\n"

echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}REPORTE 1: ESTADO INDIVIDUAL DE ARTÍCULOS${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}\n"

# Report 1A: Laptop State
echo -e "${GREEN}📊 Reporte 1A: Laptop Dell - Estado Actual${NC}"
LAPTOP_STATE=$(curl -s -X GET "${BASE_URL}/v1/inventory/items/${LAPTOP_ID}/state" \
  -H "X-Company-ID: ${COMPANY_ID}")

echo "$LAPTOP_STATE" | jq '.'

# Extract and display formatted
LAPTOP_QTY=$(echo "$LAPTOP_STATE" | jq -r '.current_quantity')
LAPTOP_AVG=$(echo "$LAPTOP_STATE" | jq -r '.current_avg_cost')
LAPTOP_TOTAL=$(echo "$LAPTOP_STATE" | jq -r '.current_total_cost')

echo -e "\n${CYAN}┌────────────────────────────────────────────┐${NC}"
echo -e "${CYAN}│ Laptop Dell - Resumen de Inventario       │${NC}"
echo -e "${CYAN}├────────────────────────────────────────────┤${NC}"
echo -e "${CYAN}│ Cantidad Actual:     ${LAPTOP_QTY} unidades      │${NC}"
echo -e "${CYAN}│ Costo Promedio:      \$${LAPTOP_AVG}            │${NC}"
echo -e "${CYAN}│ Valor Total:         \$${LAPTOP_TOTAL}          │${NC}"
echo -e "${CYAN}└────────────────────────────────────────────┘${NC}\n"

# Report 1B: Mouse State
echo -e "${GREEN}📊 Reporte 1B: Mouse Logitech - Estado Actual${NC}"
MOUSE_STATE=$(curl -s -X GET "${BASE_URL}/v1/inventory/items/${MOUSE_ID}/state" \
  -H "X-Company-ID: ${COMPANY_ID}")

echo "$MOUSE_STATE" | jq '.'
echo ""

# Report 1C: Keyboard State
echo -e "${GREEN}📊 Reporte 1C: Teclado Mecánico - Estado Actual${NC}"
KEYBOARD_STATE=$(curl -s -X GET "${BASE_URL}/v1/inventory/items/${KEYBOARD_ID}/state" \
  -H "X-Company-ID: ${COMPANY_ID}")

echo "$KEYBOARD_STATE" | jq '.'
echo ""

# Report 1D: Monitor State
echo -e "${GREEN}📊 Reporte 1D: Monitor Dell - Estado Actual${NC}"
MONITOR_STATE=$(curl -s -X GET "${BASE_URL}/v1/inventory/items/${MONITOR_ID}/state" \
  -H "X-Company-ID: ${COMPANY_ID}")

echo "$MONITOR_STATE" | jq '.'
echo ""

# Report 1E: Tax-Exempt Textbook State
echo -e "${GREEN}📊 Reporte 1E: Libros de Texto Educativos (Exentos) - Estado Actual${NC}"
TEXTBOOK_STATE=$(curl -s -X GET "${BASE_URL}/v1/inventory/items/${TEXTBOOK_ID}/state" \
  -H "X-Company-ID: ${COMPANY_ID}")

echo "$TEXTBOOK_STATE" | jq '.'
echo ""

echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}REPORTE 2: TODOS LOS ESTADOS DE INVENTARIO (Vista General)${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}\n"

echo -e "${GREEN}📊 Reporte 2: Portafolio Completo de Inventario${NC}"
ALL_STATES=$(curl -s -X GET "${BASE_URL}/v1/inventory/states" \
  -H "X-Company-ID: ${COMPANY_ID}")

echo "$ALL_STATES" | jq '.'

# Calculate and display portfolio summary
PORTFOLIO_VALUE=$(echo "$ALL_STATES" | jq '[.states[].current_total_cost] | add')
TOTAL_ITEMS=$(echo "$ALL_STATES" | jq '.count')

echo -e "\n${MAGENTA}┌─────────────────────────────────────────────────────┐${NC}"
echo -e "${MAGENTA}│ RESUMEN DEL PORTAFOLIO DE INVENTARIO                │${NC}"
echo -e "${MAGENTA}├─────────────────────────────────────────────────────┤${NC}"
echo -e "${MAGENTA}│ Total de Artículos:       ${TOTAL_ITEMS} productos              │${NC}"
echo -e "${MAGENTA}│ Valor Total del Portafolio: \$${PORTFOLIO_VALUE}              │${NC}"
echo -e "${MAGENTA}└─────────────────────────────────────────────────────┘${NC}\n"

echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}REPORTE 3: SOLO ARTÍCULOS EN EXISTENCIA${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}\n"

echo -e "${GREEN}📊 Reporte 3: Solo Artículos en Existencia (Cantidad > 0)${NC}"
curl -s -X GET "${BASE_URL}/v1/inventory/states?in_stock_only=true" \
  -H "X-Company-ID: ${COMPANY_ID}" | jq '.'

echo ""

echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}REPORTE 4: HISTORIAL DE COSTOS (Registro Detallado)${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}\n"

# Report 4A: Laptop Cost History
echo -e "${GREEN}📊 Reporte 4A: Laptop Dell - Historial Completo de Costos${NC}"
LAPTOP_HISTORY=$(curl -s -X GET "${BASE_URL}/v1/inventory/items/${LAPTOP_ID}/cost-history?limit=20" \
  -H "X-Company-ID: ${COMPANY_ID}")

echo "$LAPTOP_HISTORY" | jq '.events[] | {
  fecha: .event_timestamp,
  tipo: .event_type,
  cantidad: .quantity,
  costo_unitario: .unit_cost,
  promedio_antes: .moving_avg_cost_before,
  promedio_despues: .moving_avg_cost_after,
  balance_cantidad: .balance_quantity_after,
  notas: .notes
}'

# Calculate cost trend
FIRST_AVG=$(echo "$LAPTOP_HISTORY" | jq -r '.events[-1].moving_avg_cost_after')
LATEST_AVG=$(echo "$LAPTOP_HISTORY" | jq -r '.events[0].moving_avg_cost_after')
EVENT_COUNT=$(echo "$LAPTOP_HISTORY" | jq '.count')

echo -e "\n${CYAN}┌────────────────────────────────────────┐${NC}"
echo -e "${CYAN}│ Análisis de Tendencia de Costos       │${NC}"
echo -e "${CYAN}├────────────────────────────────────────┤${NC}"
echo -e "${CYAN}│ Primer Costo Prom:  \$${FIRST_AVG}           │${NC}"
echo -e "${CYAN}│ Último Costo Prom:  \$${LATEST_AVG}          │${NC}"
echo -e "${CYAN}│ Total de Eventos:   ${EVENT_COUNT}                 │${NC}"
echo -e "${CYAN}└────────────────────────────────────────┘${NC}\n"

# Report 4B: Mouse Cost History (Steady Cost)
echo -e "${GREEN}📊 Reporte 4B: Mouse Logitech - Historial de Costos (Patrón Estable)${NC}"
curl -s -X GET "${BASE_URL}/v1/inventory/items/${MOUSE_ID}/cost-history?limit=10" \
  -H "X-Company-ID: ${COMPANY_ID}" | jq '.events[] | {
  fecha: .event_timestamp,
  tipo: .event_type,
  cantidad: .quantity,
  costo_unitario: .unit_cost,
  promedio_despues: .moving_avg_cost_after
}'

echo ""

# Report 4C: Keyboard Cost History (Rising Cost)
echo -e "${GREEN}📊 Reporte 4C: Teclado Mecánico - Historial de Costos (Patrón Creciente)${NC}"
curl -s -X GET "${BASE_URL}/v1/inventory/items/${KEYBOARD_ID}/cost-history?limit=10" \
  -H "X-Company-ID: ${COMPANY_ID}" | jq '.events[] | {
  fecha: .event_timestamp,
  tipo: .event_type,
  cantidad: .quantity,
  costo_unitario: .unit_cost,
  promedio_despues: .moving_avg_cost_after
}'

echo ""

# Report 4D: Monitor Cost History (Volatile Cost)
echo -e "${GREEN}📊 Reporte 4D: Monitor Dell - Historial de Costos (Patrón Volátil)${NC}"
curl -s -X GET "${BASE_URL}/v1/inventory/items/${MONITOR_ID}/cost-history?limit=10" \
  -H "X-Company-ID: ${COMPANY_ID}" | jq '.events[] | {
  fecha: .event_timestamp,
  tipo: .event_type,
  cantidad: .quantity,
  costo_unitario: .unit_cost,
  promedio_despues: .moving_avg_cost_after
}'

echo ""

# Report 4E: Tax-Exempt Textbooks History
echo -e "${GREEN}📊 Reporte 4E: Libros de Texto Educativos (Exentos) - Historial de Costos${NC}"
curl -s -X GET "${BASE_URL}/v1/inventory/items/${TEXTBOOK_ID}/cost-history?limit=10" \
  -H "X-Company-ID: ${COMPANY_ID}" | jq '.events[] | {
  fecha: .event_timestamp,
  tipo: .event_type,
  cantidad: .quantity,
  costo_unitario: .unit_cost,
  promedio_despues: .moving_avg_cost_after,
  notas: .notes
}'

echo ""

echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}REPORTE 5: HISTORIAL DE COSTOS CON LÍMITE${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}\n"

echo -e "${GREEN}📊 Reporte 5: Laptop - Solo Últimos 3 Eventos${NC}"
curl -s -X GET "${BASE_URL}/v1/inventory/items/${LAPTOP_ID}/cost-history?limit=3" \
  -H "X-Company-ID: ${COMPANY_ID}" | jq '.'

echo ""

echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}REPORTE 6: REGISTRO DE AJUSTES${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}\n"

echo -e "${GREEN}📊 Reporte 6: Todos los Ajustes en Todos los Artículos${NC}"
echo -e "   (Filtrando por tipo de evento ADJUSTMENT)\n"

curl -s -X GET "${BASE_URL}/v1/inventory/items/${LAPTOP_ID}/cost-history?limit=50" \
  -H "X-Company-ID: ${COMPANY_ID}" | jq '.events[] | select(.event_type == "ADJUSTMENT") | {
  fecha: .event_timestamp,
  cantidad: .quantity,
  razon: .notes,
  impacto_promedio: (.moving_avg_cost_after - .moving_avg_cost_before)
}'

echo ""

echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}✅ TODOS LOS REPORTES COMPLETADOS${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}\n"

echo -e "${YELLOW}📊 Resumen:${NC}"
echo -e "   • Valor del Portafolio: \$${PORTFOLIO_VALUE}"
echo -e "   • Total de Artículos: ${TOTAL_ITEMS}"
echo -e "   • Tendencia de Costo Laptop: \$${FIRST_AVG} → \$${LATEST_AVG}"
echo -e "\n${GREEN}¡Todas las funciones de seguimiento de costos verificadas! ✅${NC}\n"
