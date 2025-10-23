#!/bin/bash

# Check if company_id was provided
if [ -z "$1" ]; then
    echo "Usage: ./test_inventory_cost_setup.sh <company_id>"
    echo "Example: ./test_inventory_cost_setup.sh 550e8400-e29b-41d4-a716-446655440000"
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
NC='\033[0m' # No Color

echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     INVENTORY COST TRACKING - SETUP & TESTING             ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo -e "Company ID: ${COMPANY_ID}\n"

# Helper function to find item by SKU
find_item_by_sku() {
    local sku=$1
    curl -s -X GET "${BASE_URL}/v1/inventory/items?active=true" \
      -H "X-Company-ID: ${COMPANY_ID}" | jq -r --arg sku "$sku" '.items[] | select(.sku == $sku) | .id'
}

# Helper function to display moving average calculation
show_calculation() {
    local prev_qty=$1
    local prev_avg=$2
    local new_qty=$3
    local new_cost=$4
    local expected_avg=$5
    
    echo -e "   ${CYAN}Cálculo:${NC}"
    echo -e "   └─ ($prev_qty × \$$prev_avg + $new_qty × \$$new_cost) / $(($prev_qty + $new_qty))"
    echo -e "   └─ Promedio Esperado: \$$expected_avg"
}

echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}SECCIÓN A: CREAR ARTÍCULOS DE INVENTARIO${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}\n"

# Test 1: Create Taxable Item - Laptop
echo -e "${GREEN}🟢 Prueba 1: Creando Artículo Gravado - Laptop Dell${NC}"
LAPTOP_RESPONSE=$(curl -s -X POST "${BASE_URL}/v1/inventory/items" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "tipo_item": "1",
    "sku": "LAPTOP-DELL-001",
    "name": "Laptop Dell Latitude 5520",
    "description": "Laptop empresarial 15.6 pulgadas, Intel i7, 16GB RAM, 512GB SSD",
    "manufacturer": "Dell",
    "unit_price": 1200.00,
    "unit_of_measure": "unidad",
    "is_tax_exempt": false
  }')

echo "$LAPTOP_RESPONSE" | jq '.'
LAPTOP_ID=$(echo "$LAPTOP_RESPONSE" | jq -r '.id')
echo -e "   ${CYAN}└─ Laptop ID: $LAPTOP_ID${NC}\n"

# Test 2: Create Tax-Exempt Item - Educational Textbook
echo -e "${GREEN}🟢 Prueba 2: Creando Artículo Exento - Libro de Texto Educativo${NC}"
TEXTBOOK_RESPONSE=$(curl -s -X POST "${BASE_URL}/v1/inventory/items" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "tipo_item": "1",
    "sku": "LIBRO-MAT-SEC",
    "name": "Matemáticas Secundaria - Libro de Texto",
    "description": "Libro de texto educativo para matemáticas nivel secundaria, incluye ejercicios y soluciones",
    "unit_price": 25.00,
    "unit_of_measure": "unidad",
    "is_tax_exempt": true,
    "taxes": []
  }')

echo "$TEXTBOOK_RESPONSE" | jq '.'
TEXTBOOK_ID=$(echo "$TEXTBOOK_RESPONSE" | jq -r '.id')
echo -e "   ${CYAN}└─ Libro ID: $TEXTBOOK_ID${NC}\n"

# Test 3: Create Item - Mouse
echo -e "${GREEN}🟢 Prueba 3: Creando Artículo - Mouse Logitech${NC}"
MOUSE_RESPONSE=$(curl -s -X POST "${BASE_URL}/v1/inventory/items" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "tipo_item": "1",
    "sku": "MOUSE-LOG-001",
    "name": "Mouse Inalámbrico Logitech MX Master 3",
    "description": "Mouse ergonómico inalámbrico con desplazamiento de precisión",
    "manufacturer": "Logitech",
    "unit_price": 99.99,
    "unit_of_measure": "unidad"
  }')

echo "$MOUSE_RESPONSE" | jq '.'
MOUSE_ID=$(echo "$MOUSE_RESPONSE" | jq -r '.id')
echo -e "   ${CYAN}└─ Mouse ID: $MOUSE_ID${NC}\n"

# Test 4: Create Item - Keyboard
echo -e "${GREEN}🟢 Prueba 4: Creando Artículo - Teclado Mecánico${NC}"
KEYBOARD_RESPONSE=$(curl -s -X POST "${BASE_URL}/v1/inventory/items" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "tipo_item": "1",
    "sku": "TECLADO-MECA-001",
    "name": "Teclado Mecánico Inalámbrico RGB",
    "description": "Teclado mecánico con iluminación RGB, switches blue, conectividad Bluetooth",
    "unit_price": 75.00,
    "unit_of_measure": "unidad"
  }')

echo "$KEYBOARD_RESPONSE" | jq '.'
KEYBOARD_ID=$(echo "$KEYBOARD_RESPONSE" | jq -r '.id')
echo -e "   ${CYAN}└─ Teclado ID: $KEYBOARD_ID${NC}\n"

# Test 5: Create Item - Monitor
echo -e "${GREEN}🟢 Prueba 5: Creando Artículo - Monitor Dell${NC}"
MONITOR_RESPONSE=$(curl -s -X POST "${BASE_URL}/v1/inventory/items" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "tipo_item": "1",
    "sku": "MONITOR-DELL-27",
    "name": "Monitor Dell 27 Pulgadas 4K UHD",
    "description": "Monitor profesional 27 pulgadas, resolución 4K UHD, panel IPS",
    "manufacturer": "Dell",
    "unit_price": 450.00,
    "unit_of_measure": "unidad"
  }')

echo "$MONITOR_RESPONSE" | jq '.'
MONITOR_ID=$(echo "$MONITOR_RESPONSE" | jq -r '.id')
echo -e "   ${CYAN}└─ Monitor ID: $MONITOR_ID${NC}\n"

# Test 6: Create Service
echo -e "${GREEN}🟢 Prueba 6: Creando Servicio - Consultoría IT${NC}"
SERVICE_RESPONSE=$(curl -s -X POST "${BASE_URL}/v1/inventory/items" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "tipo_item": "2",
    "sku": "SRV-CONSULT-IT",
    "name": "Servicios de Consultoría en Tecnología",
    "description": "Consultoría profesional en tecnologías de información y arquitectura de sistemas",
    "unit_price": 150.00,
    "unit_of_measure": "hora"
  }')

echo "$SERVICE_RESPONSE" | jq '.'
SERVICE_ID=$(echo "$SERVICE_RESPONSE" | jq -r '.id')
echo -e "   ${CYAN}└─ Servicio ID: $SERVICE_ID${NC}\n"

echo -e "${YELLOW}📋 Resumen de Artículos Creados (buscar por SKU):${NC}"
echo -e "   LAPTOP-DELL-001      - Laptop Dell Latitude"
echo -e "   LIBRO-MAT-SEC        - Libro de Matemáticas (exento)"
echo -e "   MOUSE-LOG-001        - Mouse Logitech"
echo -e "   TECLADO-MECA-001     - Teclado Mecánico"
echo -e "   MONITOR-DELL-27      - Monitor Dell 27\""
echo -e "   SRV-CONSULT-IT       - Consultoría IT\n"

echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}SECCIÓN B: REGISTRAR COMPRAS (Seguimiento de Costos)${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}\n"

# Re-fetch IDs dynamically from API (in case items already existed)
LAPTOP_ID=$(find_item_by_sku "LAPTOP-DELL-001")
MOUSE_ID=$(find_item_by_sku "MOUSE-LOG-001")
KEYBOARD_ID=$(find_item_by_sku "TECLADO-MECA-001")
MONITOR_ID=$(find_item_by_sku "MONITOR-DELL-27")
TEXTBOOK_ID=$(find_item_by_sku "LIBRO-MAT-SEC")

# Test 7: First Purchase - Laptop
echo -e "${GREEN}🟢 Prueba 7: Registrar Primera Compra - Laptop${NC}"
echo -e "   ├─ Cantidad: 100 unidades"
echo -e "   ├─ Costo Unitario: \$400.00"
echo -e "   └─ Promedio Esperado: \$400.00\n"

curl -s -X POST "${BASE_URL}/v1/inventory/items/${LAPTOP_ID}/purchase" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "quantity": 100,
    "unit_cost": 400.00,
    "notes": "Compra inicial de laptops Dell desde proveedor"
  }' | jq '{
    event_type,
    quantity,
    unit_cost,
    moving_avg_cost_before,
    moving_avg_cost_after,
    balance_quantity_after,
    balance_total_cost_after
  }'

echo ""

# Test 8: Second Purchase - Laptop (Higher Cost)
echo -e "${GREEN}🟢 Prueba 8: Registrar Segunda Compra - Laptop (Aumento de Precio)${NC}"
echo -e "   ├─ Cantidad: 50 unidades"
echo -e "   ├─ Costo Unitario: \$450.00"
echo -e "   └─ Promedio Esperado: \$416.67\n"
show_calculation 100 400.00 50 450.00 416.67

curl -s -X POST "${BASE_URL}/v1/inventory/items/${LAPTOP_ID}/purchase" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "quantity": 50,
    "unit_cost": 450.00,
    "notes": "Segunda compra - precio incrementado por proveedor"
  }' | jq '{
    event_type,
    quantity,
    unit_cost,
    moving_avg_cost_before,
    moving_avg_cost_after,
    balance_quantity_after,
    balance_total_cost_after
  }'

echo ""

# Test 9: Third Purchase - Laptop (Lower Cost)
echo -e "${GREEN}🟢 Prueba 9: Registrar Tercera Compra - Laptop (Descuento)${NC}"
echo -e "   ├─ Cantidad: 25 unidades"
echo -e "   ├─ Costo Unitario: \$380.00"
echo -e "   └─ Promedio Esperado: \$410.00\n"

curl -s -X POST "${BASE_URL}/v1/inventory/items/${LAPTOP_ID}/purchase" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "quantity": 25,
    "unit_cost": 380.00,
    "notes": "Compra con descuento por volumen"
  }' | jq '{
    event_type,
    quantity,
    unit_cost,
    moving_avg_cost_before,
    moving_avg_cost_after,
    balance_quantity_after,
    balance_total_cost_after
  }'

echo ""

# Test 10: Purchase - Mouse (Steady Cost)
echo -e "${GREEN}🟢 Prueba 10: Registrar Compras - Mouse (Costo Estable)${NC}"
echo -e "   Compra 1: 100 unidades @ \$8.50"

curl -s -X POST "${BASE_URL}/v1/inventory/items/${MOUSE_ID}/purchase" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "quantity": 100,
    "unit_cost": 8.50,
    "notes": "Inventario inicial de mouse"
  }' | jq '{quantity, unit_cost, moving_avg_cost_after}'

echo -e "   Compra 2: 100 unidades @ \$8.50"

curl -s -X POST "${BASE_URL}/v1/inventory/items/${MOUSE_ID}/purchase" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "quantity": 100,
    "unit_cost": 8.50,
    "notes": "Segunda compra - mismo costo"
  }' | jq '{quantity, unit_cost, moving_avg_cost_after}'

echo -e "   Compra 3: 50 unidades @ \$8.50"

curl -s -X POST "${BASE_URL}/v1/inventory/items/${MOUSE_ID}/purchase" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "quantity": 50,
    "unit_cost": 8.50,
    "notes": "Tercera compra - mismo costo"
  }' | jq '{quantity, unit_cost, moving_avg_cost_after}'

echo -e "   ${CYAN}└─ El promedio debe mantenerse en \$8.50${NC}\n"

# Test 11: Purchase - Keyboard (Rising Cost)
echo -e "${GREEN}🟢 Prueba 11: Registrar Compras - Teclado (Costo Creciente)${NC}"
echo -e "   Compra 1: 50 unidades @ \$20.00"

curl -s -X POST "${BASE_URL}/v1/inventory/items/${KEYBOARD_ID}/purchase" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "quantity": 50,
    "unit_cost": 20.00,
    "notes": "Compra inicial de teclados"
  }' | jq '{quantity, unit_cost, moving_avg_cost_after}'

echo -e "   Compra 2: 50 unidades @ \$25.00"

curl -s -X POST "${BASE_URL}/v1/inventory/items/${KEYBOARD_ID}/purchase" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "quantity": 50,
    "unit_cost": 25.00,
    "notes": "Segunda compra - aumento de precio"
  }' | jq '{quantity, unit_cost, moving_avg_cost_after}'

echo -e "   Compra 3: 50 unidades @ \$30.00"

curl -s -X POST "${BASE_URL}/v1/inventory/items/${KEYBOARD_ID}/purchase" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "quantity": 50,
    "unit_cost": 30.00,
    "notes": "Tercera compra - continúa incremento"
  }' | jq '{quantity, unit_cost, moving_avg_cost_after}'

echo -e "   ${CYAN}└─ El promedio debe aumentar: \$20 → \$22.50 → \$25.00${NC}\n"

# Test 12: Purchase - Monitor (Volatile Cost)
echo -e "${GREEN}🟢 Prueba 12: Registrar Compras - Monitor (Costo Volátil)${NC}"
echo -e "   Compra 1: 30 unidades @ \$200.00"

curl -s -X POST "${BASE_URL}/v1/inventory/items/${MONITOR_ID}/purchase" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "quantity": 30,
    "unit_cost": 200.00,
    "notes": "Compra inicial de monitores"
  }' | jq '{quantity, unit_cost, moving_avg_cost_after}'

echo -e "   Compra 2: 20 unidades @ \$250.00"

curl -s -X POST "${BASE_URL}/v1/inventory/items/${MONITOR_ID}/purchase" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "quantity": 20,
    "unit_cost": 250.00,
    "notes": "Segunda compra - modelos premium"
  }' | jq '{quantity, unit_cost, moving_avg_cost_after}'

echo -e "   Compra 3: 25 unidades @ \$180.00"

curl -s -X POST "${BASE_URL}/v1/inventory/items/${MONITOR_ID}/purchase" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "quantity": 25,
    "unit_cost": 180.00,
    "notes": "Tercera compra - liquidación"
  }' | jq '{quantity, unit_cost, moving_avg_cost_after}'

echo -e "   ${CYAN}└─ El promedio debe fluctuar: \$200 → \$220 → \$208${NC}\n"

# Test 13: Purchase - Tax-Exempt Educational Textbooks
echo -e "${GREEN}🟢 Prueba 13: Registrar Compra - Libros de Texto Educativos (Exentos)${NC}"
echo -e "   ├─ Cantidad: 500 unidades"
echo -e "   ├─ Costo Unitario: \$12.00"
echo -e "   └─ Verificar: Artículos exentos rastrean costo igual que gravados\n"

curl -s -X POST "${BASE_URL}/v1/inventory/items/${TEXTBOOK_ID}/purchase" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "quantity": 500,
    "unit_cost": 12.00,
    "notes": "Compra inicial de libros de texto para distribución escolar"
  }' | jq '{
    event_type,
    quantity,
    unit_cost,
    moving_avg_cost_after,
    balance_quantity_after
  }'

echo ""

echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}SECCIÓN C: REGISTRAR AJUSTES${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}\n"

# Test 14: Positive Adjustment - Found Inventory
echo -e "${GREEN}🟢 Prueba 14: Ajuste Positivo - Laptops Encontradas${NC}"
echo -e "   ├─ Cantidad: +10 unidades"
echo -e "   ├─ Costo Unitario: \$420.00"
echo -e "   └─ Razón: Encontradas durante conteo físico\n"

curl -s -X POST "${BASE_URL}/v1/inventory/items/${LAPTOP_ID}/adjustment" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "quantity": 10,
    "unit_cost": 420.00,
    "reason": "Encontradas 10 unidades en bodega durante conteo físico de inventario"
  }' | jq '{
    event_type,
    quantity,
    unit_cost,
    moving_avg_cost_before,
    moving_avg_cost_after,
    balance_quantity_after,
    notes
  }'

echo ""

# Test 15: Negative Adjustment - Damaged Inventory
echo -e "${GREEN}🟢 Prueba 15: Ajuste Negativo - Laptops Dañadas${NC}"
echo -e "   ├─ Cantidad: -5 unidades"
echo -e "   ├─ Costo Unitario: (promedio actual - calculado automáticamente)"
echo -e "   └─ Razón: Daño por agua\n"

curl -s -X POST "${BASE_URL}/v1/inventory/items/${LAPTOP_ID}/adjustment" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "quantity": -5,
    "reason": "Daño por agua durante almacenamiento - unidades destruidas y removidas del inventario"
  }' | jq '{
    event_type,
    quantity,
    unit_cost,
    moving_avg_cost_before,
    moving_avg_cost_after,
    balance_quantity_after,
    notes
  }'

echo -e "   ${CYAN}└─ Nota: El promedio debe mantenerse igual, solo disminuye cantidad/costo total${NC}\n"

# Test 16: Negative Adjustment - Damaged Textbooks
echo -e "${GREEN}🟢 Prueba 16: Ajuste Negativo - Libros Dañados${NC}"
echo -e "   ├─ Cantidad: -50 unidades"
echo -e "   └─ Razón: Libros dañados por humedad\n"

curl -s -X POST "${BASE_URL}/v1/inventory/items/${TEXTBOOK_ID}/adjustment" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "quantity": -50,
    "reason": "Libros de texto dañados por humedad - removidos del inventario"
  }' | jq '{
    event_type,
    quantity,
    moving_avg_cost_after,
    balance_quantity_after,
    notes
  }'

echo ""

# Test 17: Positive Adjustment - Mouse Found
echo -e "${GREEN}🟢 Prueba 17: Ajuste Positivo - Mouse Encontrados${NC}"

curl -s -X POST "${BASE_URL}/v1/inventory/items/${MOUSE_ID}/adjustment" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "quantity": 25,
    "unit_cost": 9.00,
    "reason": "Encontrada caja sin abrir en cuarto de almacenamiento"
  }' | jq '{
    event_type,
    quantity,
    unit_cost,
    moving_avg_cost_before,
    moving_avg_cost_after,
    balance_quantity_after
  }'

echo ""

echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}SECCIÓN D: CASOS DE ERROR${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}\n"

# Test 18: Error - Adjustment causes negative inventory
echo -e "${RED}🔴 Prueba 18: ERROR - Ajuste Causaría Inventario Negativo${NC}"
echo -e "   └─ Esperado: 400 Bad Request\n"

curl -s -X POST "${BASE_URL}/v1/inventory/items/${LAPTOP_ID}/adjustment" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "quantity": -1000,
    "reason": "Prueba inventario negativo"
  }' | jq '.'

echo ""

# Test 19: Error - Positive adjustment without unit cost
echo -e "${RED}🔴 Prueba 19: ERROR - Ajuste Positivo Sin Costo Unitario${NC}"
echo -e "   └─ Esperado: 400 Bad Request\n"

curl -s -X POST "${BASE_URL}/v1/inventory/items/${LAPTOP_ID}/adjustment" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "quantity": 10,
    "reason": "Encontré artículos pero olvidé el costo"
  }' | jq '.'

echo ""

# Test 20: Error - Zero quantity adjustment
echo -e "${RED}🔴 Prueba 20: ERROR - Ajuste de Cantidad Cero${NC}"
echo -e "   └─ Esperado: 400 Bad Request\n"

curl -s -X POST "${BASE_URL}/v1/inventory/items/${LAPTOP_ID}/adjustment" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "quantity": 0,
    "unit_cost": 100.00,
    "reason": "Sin cambio"
  }' | jq '.'

echo ""

# Test 21: Error - Negative unit cost
echo -e "${RED}🔴 Prueba 21: ERROR - Costo Unitario Negativo${NC}"
echo -e "   └─ Esperado: 400 Bad Request\n"

curl -s -X POST "${BASE_URL}/v1/inventory/items/${LAPTOP_ID}/purchase" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d '{
    "quantity": 10,
    "unit_cost": -50.00,
    "notes": "Prueba costo negativo"
  }' | jq '.'

echo ""

echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}✅ CONFIGURACIÓN COMPLETADA${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}\n"

echo -e "${YELLOW}📋 Artículos Creados (buscar por SKU):${NC}"
echo -e "   LAPTOP-DELL-001      - Laptop Dell Latitude"
echo -e "   LIBRO-MAT-SEC        - Libro de Matemáticas (exento de impuestos)"
echo -e "   MOUSE-LOG-001        - Mouse Logitech"
echo -e "   TECLADO-MECA-001     - Teclado Mecánico"
echo -e "   MONITOR-DELL-27      - Monitor Dell 27\""
echo -e "   SRV-CONSULT-IT       - Consultoría IT"

echo -e "\n${YELLOW}🔍 Ejecutar script de reportes: ./test_inventory_cost_reports.sh ${COMPANY_ID}${NC}\n"
