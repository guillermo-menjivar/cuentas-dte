#!/bin/bash

# Test script to finalize a draft invoice

if [ "$#" -lt 2 ]; then
    echo "Usage: $0 <company_id> <invoice_id>"
    echo "Example: $0 abc-123 invoice-456"
    exit 1
fi

COMPANY_ID=$1
INVOICE_ID=$2

API_URL="http://localhost:8080/v1/invoices/$INVOICE_ID/finalize"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Finalizing Invoice ===${NC}"
echo "Company ID: $COMPANY_ID"
echo "Invoice ID: $INVOICE_ID"
echo ""

# Make the API call
RESPONSE=$(curl -s -X POST "$API_URL" \
    -H "Content-Type: application/json" \
    -H "X-Company-ID: $COMPANY_ID")

echo "Response:"
echo "$RESPONSE" | jq '.'

# Check if successful
STATUS=$(echo "$RESPONSE" | jq -r '.status // empty')

if [ "$STATUS" = "finalized" ]; then
    echo ""
    echo -e "${GREEN}✓ Invoice finalized successfully!${NC}"
    
    CODIGO=$(echo "$RESPONSE" | jq -r '.dte_codigo_generacion // empty')
    NUMERO=$(echo "$RESPONSE" | jq -r '.dte_numero_control // empty')
    DTE_STATUS=$(echo "$RESPONSE" | jq -r '.dte_status // empty')
    FINALIZED_AT=$(echo "$RESPONSE" | jq -r '.finalized_at // empty')
    
    echo ""
    echo -e "${YELLOW}DTE Information:${NC}"
    echo "Código Generación: $CODIGO"
    echo "Número Control: $NUMERO"
    echo "DTE Status: $DTE_STATUS"
    echo "Finalized At: $FINALIZED_AT"
else
    echo ""
    echo -e "${RED}✗ Failed to finalize invoice${NC}"
    ERROR=$(echo "$RESPONSE" | jq -r '.error // empty')
    if [ -n "$ERROR" ]; then
        echo "Error: $ERROR"
    fi
fi
