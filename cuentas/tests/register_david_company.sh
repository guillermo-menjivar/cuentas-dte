#!/bin/bash

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🚀 Registering company...${NC}"

RESPONSE=$(curl -s -X POST http://localhost:8080/v1/companies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Paredes & Paredes, S.A. de C.V",
    "nit": "0614-300506-101-3",
    "dte_ambiente": "00",
    "nombre_comercial": "Paredes & Paredes, S.A. de C.V",
    "ncr": "172631-3",
    "firmador_username": "06143005061013",
    "firmador_password": "sdKC4uLduegSPT",
    "hc_username": "06143005061013",
    "hc_password": "MF7HwttFuZ.*3RY",
    "cod_actividad": "69200",
    "email": "contact@paredes.com",
    "departamento": "06",
    "municipio": "21",
    "complemento_direccion": "Col Escalón 75 Av Nte No 5245 San Salvador, San Salvador",
    "telefono": "23232323"
  }')

# Check if response contains error
ERROR=$(echo "$RESPONSE" | jq -r '.error // empty')

if [ -n "$ERROR" ]; then
  # Check if it's a duplicate error
  if [[ "$ERROR" == *"duplicate key"* ]] || [[ "$ERROR" == *"already exists"* ]]; then
    echo -e "${YELLOW}⚠️  Company already registered (duplicate found)${NC}"
    echo -e "${GREEN}✓ Continuing with existing company...${NC}"
    exit 0
  else
    # Other error
    echo -e "${RED}❌ Failed to register company:${NC}"
    echo "$RESPONSE" | jq .
    exit 1
  fi
else
  # Success
  echo -e "${GREEN}✓ Company registered successfully!${NC}"
  echo "$RESPONSE" | jq .
  exit 0
fi
