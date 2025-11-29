#!/bin/bash

# Check if company_id was provided
if [ -z "$1" ]; then
    echo "Usage: ./test_inventory_cost_between_transactions.sh <company_id>"
    echo "Example: ./test_inventory_cost_between_transactions.sh 550e8400-e29b-41d4-a716-446655440000"
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
echo -e "${BLUE}║  IMPACTO DE COSTOS - Pruebas Entre Transacciones          ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo -e "Company ID: ${COMPANY_ID}\n"

echo -e "${YELLOW}Este script demuestra cómo los cambios de costo de inventario afectan transacciones${NC}"
echo -e "${YELLOW}Use esto ENTRE hacer facturas reales para ver el impacto del costo${NC}\n"

echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}PASO 1: DESCUBRIR ARTÍCULOS DE INVENTARIO DISPONIBLES${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}\n"

echo -e "${GREEN}📋 Obteniendo todos los artículos de inventario para la empresa...${NC}\n"

ITEMS_RESPONSE=$(curl -s -X GET "${BASE_URL}/v1/inventory/items?active=true" \
  -H "X-Company-ID: ${COMPANY_ID}")

ITEM_COUNT=$(echo "$ITEMS_RESPONSE" | jq '.count')

if [ "$ITEM_COUNT" -eq 0 ]; then
    echo -e "${RED}❌ ¡No se encontraron artículos de inventario!${NC}"
    echo -e "${YELLOW}Por favor cree algunos artículos primero usando test_inventory.sh${NC}\n"
    exit 1
fi

echo -e "${GREEN}✅ Encontrados ${ITEM_COUNT} artículos de inventario${NC}\n"

# Display items in a table
echo "$ITEMS_RESPONSE" | jq -r '.items[] | "\(.id)|\(.sku)|\(.name)|\(.tipo_item)"' | while IFS='|' read -r id sku name tipo; do
    TIPO_NAME="Producto"
    if [ "$tipo" == "2" ]; then
        TIPO_NAME="Servicio"
    fi
    echo -e "   ${CYAN}ID:${NC} $id"
    echo -e "   ${CYAN}SKU:${NC} $sku | ${CYAN}Nombre:${NC} $name | ${CYAN}Tipo:${NC} $TIPO_NAME"
    echo ""
done

echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}PASO 2: SELECCIONAR UN ARTÍCULO PARA DEMOSTRACIÓN${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}\n"

# Get first product (tipo_item = 1) from the list
SELECTED_ITEM=$(echo "$ITEMS_RESPONSE" | jq -r '.items[] | select(.tipo_item == "1") | .id' | head -n 1)

if [ -z "$SELECTED_ITEM" ] || [ "$SELECTED_ITEM" == "null" ]; then
    echo -e "${RED}❌ No se encontraron productos (solo servicios disponibles)${NC}"
    echo -e "${YELLOW}Esta demostración necesita al menos un producto (tipo_item=1)${NC}\n"
    exit 1
fi

SELECTED_ITEM_NAME=$(echo "$ITEMS_RESPONSE" | jq -r --arg id "$SELECTED_ITEM" '.items[] | select(.id == $id) | .name')
SELECTED_ITEM_SKU=$(echo "$ITEMS_RESPONSE" | jq -r --arg id "$SELECTED_ITEM" '.items[] | select(.id == $id) | .sku')

echo -e "${GREEN}📦 Artículo Seleccionado para Demo:${NC}"
echo -e "   ${CYAN}ID:${NC} $SELECTED_ITEM"
echo -e "   ${CYAN}SKU:${NC} $SELECTED_ITEM_SKU"
echo -e "   ${CYAN}Nombre:${NC} $SELECTED_ITEM_NAME\n"

echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}PASO 3: VERIFICAR ESTADO ACTUAL DE INVENTARIO${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}\n"

echo -e "${GREEN}📊 Estado Actual de Inventario (ANTES de agregar más inventario):${NC}\n"

STATE_BEFORE=$(curl -s -X GET "${BASE_URL}/v1/inventory/items/${SELECTED_ITEM}/state" \
  -H "X-Company-ID: ${COMPANY_ID}")

echo "$STATE_BEFORE" | jq '.'

QTY_BEFORE=$(echo "$STATE_BEFORE" | jq -r '.current_quantity')
AVG_COST_BEFORE=$(echo "$STATE_BEFORE" | jq -r '.current_avg_cost')
TOTAL_COST_BEFORE=$(echo "$STATE_BEFORE" | jq -r '.current_total_cost')

echo -e "\n${MAGENTA}┌────────────────────────────────────────────────┐${NC}"
echo -e "${MAGENTA}│ ESTADO ACTUAL (Antes de Agregar Inventario)   │${NC}"
echo -e "${MAGENTA}├────────────────────────────────────────────────┤${NC}"
echo -e "${MAGENTA}│ Cantidad:           ${QTY_BEFORE} unidades            │${NC}"
echo -e "${MAGENTA}│ Costo Promedio:     \$${AVG_COST_BEFORE}                  │${NC}"
echo -e "${MAGENTA}│ Valor Total:        \$${TOTAL_COST_BEFORE}                │${NC}"
echo -e "${MAGENTA}└────────────────────────────────────────────────┘${NC}\n"

if [ "$QTY_BEFORE" == "0" ] || [ "$QTY_BEFORE" == "0.0" ]; then
    echo -e "${YELLOW}⚠️  ¡El artículo tiene inventario CERO!${NC}"
    echo -e "${YELLOW}Agreguemos inventario inicial primero...${NC}\n"
    
    echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}PASO 4A: AGREGAR INVENTARIO INICIAL${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}\n"
    
    INITIAL_QTY=100
    INITIAL_COST=50.00
    
    echo -e "${GREEN}📦 Agregando Inventario Inicial:${NC}"
    echo -e "   Cantidad: ${INITIAL_QTY} unidades"
    echo -e "   Costo Unitario: \$${INITIAL_COST}\n"
    
    PURCHASE_1=$(curl -s -X POST "${BASE_URL}/v1/inventory/items/${SELECTED_ITEM}/purchase" \
      -H "Content-Type: application/json" \
      -H "X-Company-ID: ${COMPANY_ID}" \
      -d "{
        \"quantity\": ${INITIAL_QTY},
        \"unit_cost\": ${INITIAL_COST},
        \"notes\": \"Inventario inicial - Demo de Costo Entre Transacciones\"
      }")
    
    echo "$PURCHASE_1" | jq '.'
    
    QTY_BEFORE=$INITIAL_QTY
    AVG_COST_BEFORE=$INITIAL_COST
    TOTAL_COST_BEFORE=$(echo "$INITIAL_QTY * $INITIAL_COST" | bc)
    
    echo -e "\n${GREEN}✅ ¡Inventario inicial agregado!${NC}"
    echo -e "   Promedio Móvil: \$0.00 → \$${INITIAL_COST}\n"
fi

echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}PASO 4B: SIMULAR PRIMERA TRANSACCIÓN (Antes de Agregar Más)${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}\n"

echo -e "${YELLOW}💰 Si crea una factura AHORA, los artículos se venderían a:${NC}"
echo -e "   ${CYAN}Costo Por Unidad: \$${AVG_COST_BEFORE}${NC}"
echo -e "   ${CYAN}Para 10 unidades: \$$(echo "$AVG_COST_BEFORE * 10" | bc) COGS (Costo de Ventas)${NC}\n"

echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}PASO 5: AGREGAR MÁS INVENTARIO A COSTO DIFERENTE${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}\n"

# Calculate a different cost (20% higher)
NEW_COST=$(echo "$AVG_COST_BEFORE * 1.2" | bc)
NEW_QTY=50

echo -e "${GREEN}📦 Agregando Más Inventario a Costo MÁS ALTO:${NC}"
echo -e "   Cantidad: ${NEW_QTY} unidades"
echo -e "   Costo Unitario: \$${NEW_COST} (aumento del 20%)"
echo -e "   Promedio Anterior: \$${AVG_COST_BEFORE}\n"

# Calculate expected new average
EXPECTED_NEW_AVG=$(echo "scale=2; ($TOTAL_COST_BEFORE + ($NEW_QTY * $NEW_COST)) / ($QTY_BEFORE + $NEW_QTY)" | bc)

echo -e "${CYAN}📊 Cálculo Esperado:${NC}"
echo -e "   Total Actual:    \$${TOTAL_COST_BEFORE} (${QTY_BEFORE} unidades × \$${AVG_COST_BEFORE})"
echo -e "   Nueva Compra:    \$$(echo "$NEW_QTY * $NEW_COST" | bc) (${NEW_QTY} unidades × \$${NEW_COST})"
echo -e "   Nuevo Total:     \$$(echo "$TOTAL_COST_BEFORE + ($NEW_QTY * $NEW_COST)" | bc)"
echo -e "   Nueva Cantidad:  $(echo "$QTY_BEFORE + $NEW_QTY" | bc) unidades"
echo -e "   ${YELLOW}Nuevo Promedio Esperado: \$${EXPECTED_NEW_AVG}${NC}\n"

PURCHASE_2=$(curl -s -X POST "${BASE_URL}/v1/inventory/items/${SELECTED_ITEM}/purchase" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d "{
    \"quantity\": ${NEW_QTY},
    \"unit_cost\": ${NEW_COST},
    \"notes\": \"Segundo lote - Costo aumentado - Demo Entre Transacciones\"
  }")

echo -e "${GREEN}✅ Compra Registrada:${NC}\n"
echo "$PURCHASE_2" | jq '.'

AVG_COST_AFTER=$(echo "$PURCHASE_2" | jq -r '.moving_avg_cost_after')
QTY_AFTER=$(echo "$PURCHASE_2" | jq -r '.balance_quantity_after')
TOTAL_COST_AFTER=$(echo "$PURCHASE_2" | jq -r '.balance_total_cost_after')

echo -e "\n${MAGENTA}┌────────────────────────────────────────────────┐${NC}"
echo -e "${MAGENTA}│ ANÁLISIS DE CAMBIO DE COSTO                   │${NC}"
echo -e "${MAGENTA}├────────────────────────────────────────────────┤${NC}"
echo -e "${MAGENTA}│ Promedio Anterior:  \$${AVG_COST_BEFORE}                 │${NC}"
echo -e "${MAGENTA}│ Nuevo Promedio:     \$${AVG_COST_AFTER}                  │${NC}"
echo -e "${MAGENTA}│ Cambio:             \$$(echo "$AVG_COST_AFTER - $AVG_COST_BEFORE" | bc)                   │${NC}"
echo -e "${MAGENTA}│ % Cambio:           $(echo "scale=2; (($AVG_COST_AFTER - $AVG_COST_BEFORE) / $AVG_COST_BEFORE) * 100" | bc)%                  │${NC}"
echo -e "${MAGENTA}└────────────────────────────────────────────────┘${NC}\n"

echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}PASO 6: SIMULAR SEGUNDA TRANSACCIÓN (Después de Agregar)${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}\n"

echo -e "${YELLOW}💰 Si crea una factura AHORA, los artículos se venderían a:${NC}"
echo -e "   ${CYAN}Costo Por Unidad: \$${AVG_COST_AFTER}${NC}"
echo -e "   ${CYAN}Para 10 unidades: \$$(echo "$AVG_COST_AFTER * 10" | bc) COGS${NC}\n"

echo -e "${RED}📊 COMPARACIÓN - Misma Venta de 10 Unidades:${NC}"
echo -e "   ${CYAN}ANTES de agregar inventario: COGS = \$$(echo "$AVG_COST_BEFORE * 10" | bc)${NC}"
echo -e "   ${CYAN}DESPUÉS de agregar:          COGS = \$$(echo "$AVG_COST_AFTER * 10" | bc)${NC}"
echo -e "   ${YELLOW}Diferencia:                  \$$(echo "($AVG_COST_AFTER - $AVG_COST_BEFORE) * 10" | bc)${NC}\n"

echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}PASO 7: VER HISTORIAL COMPLETO DE COSTOS${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}\n"

echo -e "${GREEN}📊 Historial Completo de Costos para ${SELECTED_ITEM_NAME}:${NC}\n"

HISTORY=$(curl -s -X GET "${BASE_URL}/v1/inventory/items/${SELECTED_ITEM}/cost-history?limit=10" \
  -H "X-Company-ID: ${COMPANY_ID}")

echo "$HISTORY" | jq '.events[] | {
  fecha: .event_timestamp,
  tipo: .event_type,
  cantidad: .quantity,
  costo_unitario: .unit_cost,
  promedio_antes: .moving_avg_cost_before,
  promedio_despues: .moving_avg_cost_after,
  balance_cantidad: .balance_quantity_after,
  notas: .notes
}'

echo ""

echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}PASO 8: VERIFICAR ESTADO ACTUAL${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}\n"

echo -e "${GREEN}📊 Estado Final de Inventario:${NC}\n"

STATE_AFTER=$(curl -s -X GET "${BASE_URL}/v1/inventory/items/${SELECTED_ITEM}/state" \
  -H "X-Company-ID: ${COMPANY_ID}")

echo "$STATE_AFTER" | jq '.'

echo ""

echo -e "${MAGENTA}┌─────────────────────────────────────────────────────────┐${NC}"
echo -e "${MAGENTA}│ RESUMEN FINAL - ${SELECTED_ITEM_NAME}${NC}"
echo -e "${MAGENTA}├─────────────────────────────────────────────────────────┤${NC}"
echo -e "${MAGENTA}│ Cantidad Inicial:      ${QTY_BEFORE} unidades                       │${NC}"
echo -e "${MAGENTA}│ Costo Promedio Inicial:\$${AVG_COST_BEFORE}                         │${NC}"
echo -e "${MAGENTA}│                                                         │${NC}"
echo -e "${MAGENTA}│ Cantidad Agregada:     ${NEW_QTY} unidades                        │${NC}"
echo -e "${MAGENTA}│ Costo Unitario Agregado:\$${NEW_COST}                          │${NC}"
echo -e "${MAGENTA}│                                                         │${NC}"
echo -e "${MAGENTA}│ Cantidad Final:        ${QTY_AFTER} unidades                       │${NC}"
echo -e "${MAGENTA}│ Costo Promedio Final:  \$${AVG_COST_AFTER}                         │${NC}"
echo -e "${MAGENTA}│ Valor Total Final:     \$${TOTAL_COST_AFTER}                       │${NC}"
echo -e "${MAGENTA}│                                                         │${NC}"
echo -e "${MAGENTA}│ ${YELLOW}Impacto de Costo en Próxima Venta:${NC}                     │${NC}"
echo -e "${MAGENTA}│ • Vender 10 uds habría costado \$$(echo "$AVG_COST_BEFORE * 10" | bc) ANTES    │${NC}"
echo -e "${MAGENTA}│ • Vender 10 uds ahora costará \$$(echo "$AVG_COST_AFTER * 10" | bc) DESPUÉS  │${NC}"
echo -e "${MAGENTA}│ • Diferencia: \$$(echo "($AVG_COST_AFTER - $AVG_COST_BEFORE) * 10" | bc) más alto en COGS            │${NC}"
echo -e "${MAGENTA}└─────────────────────────────────────────────────────────┘${NC}\n"

echo -e "${GREEN}✅ DEMOSTRACIÓN COMPLETADA${NC}\n"

echo -e "${YELLOW}💡 Conclusiones Clave:${NC}"
echo -e "   1. El costo promedio móvil cambia al agregar inventario a precios diferentes"
echo -e "   2. Su próxima factura usará el NUEVO costo promedio (\$${AVG_COST_AFTER})"
echo -e "   3. Esto afecta el COGS y los márgenes de ganancia en las ventas"
echo -e "   4. Todos los cambios quedan registrados en el log inmutable de eventos\n"

echo -e "${CYAN}🔄 ¿Quiere ver esto de nuevo?${NC}"
echo -e "   Ejecute: ./test_inventory_cost_between_transactions.sh ${COMPANY_ID}\n"
