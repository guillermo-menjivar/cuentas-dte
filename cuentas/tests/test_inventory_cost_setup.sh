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

# Helper function to get or create item
get_or_create_item() {
    local sku=$1
    local item_name=$2
    local create_json=$3
    
    # Check if item already exists
    local existing_id=$(find_item_by_sku "$sku")
    
    if [ ! -z "$existing_id" ] && [ "$existing_id" != "null" ]; then
        echo -e "   ${YELLOW}⚠️  Item ya existe, usando existente${NC}"
        echo -e "   ${CYAN}└─ ID: $existing_id${NC}\n"
        echo "$existing_id"
        return 0
    fi
    
    # Create new item
    local response=$(curl -s -X POST "${BASE_URL}/v1/inventory/items" \
      -H "Content-Type: application/json" \
      -H "X-Company-ID: ${COMPANY_ID}" \
      -d "$create_json")
    
    local new_id=$(echo "$response" | jq -r '.id')
    
    if [ "$new_id" == "null" ] || [ -z "$new_id" ]; then
        echo -e "   ${RED}❌ Error al crear item${NC}"
        echo "$response" | jq '.'
        return 1
    fi
    
    echo "$response" | jq '.'
    echo -e "   ${CYAN}└─ Nuevo item creado: $new_id${NC}\n"
    echo "$new_id"
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
echo -e "${GREEN}🟢 Prueba 1: Obtener o Crear Artículo Gravado - Laptop Dell${NC}"
LAPTOP_ID=$(get_or_create_item "LAPTOP-DELL-001" "Laptop Dell" '{
  "tipo_item": "1",
  "sku": "LAPTOP-DELL-001",
  "name": "Laptop Dell Latitude 5520",
  "description": "Laptop empresarial 15.6 pulgadas, Intel i7, 16GB RAM, 512GB SSD",
  "manufacturer": "Dell",
  "unit_price": 1200.00,
  "unit_of_measure": "unidad",
  "is_tax_exempt": false
}')

# Test 2: Create Tax-Exempt Item - Educational Textbook
echo -e "${GREEN}🟢 Prueba 2: Obtener o Crear Artículo Exento - Libro de Texto Educativo${NC}"
TEXTBOOK_ID=$(get_or_create_item "LIBRO-MAT-SEC" "Libro de Matemáticas" '{
  "tipo_item": "1",
  "sku": "LIBRO-MAT-SEC",
  "name": "Matemáticas Secundaria - Libro de Texto",
  "description": "Libro de texto educativo para matemáticas nivel secundaria, incluye ejercicios y soluciones",
  "unit_price": 25.00,
  "unit_of_measure": "unidad",
  "is_tax_exempt": true,
  "taxes": []
}')

# Test 3: Create Item - Mouse
echo -e "${GREEN}🟢 Prueba 3: Obtener o Crear Artículo - Mouse Logitech${NC}"
MOUSE_ID=$(get_or_create_item "MOUSE-LOG-001" "Mouse Logitech" '{
  "tipo_item": "1",
  "sku": "MOUSE-LOG-001",
  "name": "Mouse Inalámbrico Logitech MX Master 3",
  "description": "Mouse ergonómico inalámbrico con desplazamiento de precisión",
  "manufacturer": "Logitech",
  "unit_price": 99.99,
  "unit_of_measure": "unidad"
}')

# Test 4: Create Item - Keyboard
echo -e "${GREEN}🟢 Prueba 4: Obtener o Crear Artículo - Teclado Mecánico${NC}"
KEYBOARD_ID=$(get_or_create_item "TECLADO-MECA-001" "Teclado Mecánico" '{
  "tipo_item": "1",
  "sku": "TECLADO-MECA-001",
  "name": "Teclado Mecánico Inalámbrico RGB",
  "description": "Teclado mecánico con iluminación RGB, switches blue, conectividad Bluetooth",
  "unit_price": 75.00,
  "unit_of_measure": "unidad"
}')

# Test 5: Create Item - Monitor
echo -e "${GREEN}🟢 Prueba 5: Obtener o Crear Artículo - Monitor Dell${NC}"
MONITOR_ID=$(get_or_create_item "MONITOR-DELL-27" "Monitor Dell" '{
  "tipo_item": "1",
  "sku": "MONITOR-DELL-27",
  "name": "Monitor Dell 27 Pulgadas 4K UHD",
  "description": "Monitor profesional 27 pulgadas, resolución 4K UHD, panel IPS",
  "manufacturer": "Dell",
  "unit_price": 450.00,
  "unit_of_measure": "unidad"
}')

# Test 6: Create Service
echo -e "${GREEN}🟢 Prueba 6: Obtener o Crear Servicio - Consultoría IT${NC}"
SERVICE_ID=$(get_or_create_item "SRV-CONSULT-IT" "Consultoría IT" '{
  "tipo_item": "2",
  "sku": "SRV-CONSULT-IT",
  "name": "Servicios de Consultoría en Tecnología",
  "description": "Consultoría profesional en tecnologías de información y arquitectura de sistemas",
  "unit_price": 150.00,
  "unit_of_measure": "hora"
}')

echo -e "${YELLOW}📋 Resumen de Artículos Creados (buscar por SKU):${NC}"
echo -e "   LAPTOP-DELL-001      - Laptop Dell Latitude (ID: ${LAPTOP_ID})"
echo -e "   LIBRO-MAT-SEC        - Libro de Matemáticas (exento) (ID: ${TEXTBOOK_ID})"
echo -e "   MOUSE-LOG-001        - Mouse Logitech (ID: ${MOUSE_ID})"
echo -e "   TECLADO-MECA-001     - Teclado Mecánico (ID: ${KEYBOARD_ID})"
echo -e "   MONITOR-DELL-27      - Monitor Dell 27\" (ID: ${MONITOR_ID})"
echo -e "   SRV-CONSULT-IT       - Consultoría IT (ID: ${SERVICE_ID})\n"

echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}SECCIÓN B: REGISTRAR COMPRAS (Seguimiento de Costos)${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}\n"

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

# Test 16: Negative Adjustment - Expired Textbooks
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
    "quantity": -10000,
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
