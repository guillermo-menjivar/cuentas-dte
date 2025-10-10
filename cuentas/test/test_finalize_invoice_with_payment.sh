#!/bin/bash

# Test script to finalize an invoice with payment information

if [ "$#" -lt 2 ]; then
    echo "Usage: $0 <company_id> <invoice_id> [payment_amount] [payment_method]"
    echo "Example: $0 abc-123 invoice-456"
    echo "Example: $0 abc-123 invoice-456 161.03 01"
    exit 1
fi

COMPANY_ID=$1
INVOICE_ID=$2
PAYMENT_AMOUNT=${3:-0}  # Default to 0 if not provided
PAYMENT_METHOD=${4:-"01"}  # Default to cash (01)

API_URL="http://localhost:8080/v1/invoices/$INVOICE_ID/finalize"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Finalizing Invoice with Payment ===${NC}"
echo "Company ID: $COMPANY_ID"
echo "Invoice ID: $INVOICE_ID"
echo "Payment Amount: \$$PAYMENT_AMOUNT"
echo "Payment Method: $PAYMENT_METHOD"
echo ""

# Build request body
REQUEST_BODY=$(cat <<EOF
{
  "payment": {
    "amount": $PAYMENT_AMOUNT,
    "payment_method": "$PAYMENT_METHOD",
    "reference": "TEST-PAYMENT-001"
  }
}
EOF
)

echo "Request Body:"
echo "$REQUEST_BODY" | jq '.'
echo ""

# Make the API call
RESPONSE=$(curl -s -X POST "$API_URL" \
    -H "Content-Type: application/json" \
    -H "X-Company-ID: $COMPANY_ID" \
    -d "$REQUEST_BODY")

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
    PAYMENT_STATUS=$(echo "$RESPONSE" | jq -r '.payment_status // empty')
    AMOUNT_PAID=$(echo "$RESPONSE" | jq -r '.amount_paid // empty')
    BALANCE_DUE=$(echo "$RESPONSE" | jq -r '.balance_due // empty')
    
    echo ""
    echo -e "${YELLOW}Invoice Information:${NC}"
    echo "Código Generación: $CODIGO"
    echo "Número Control: $NUMERO"
    echo "DTE Status: $DTE_STATUS"
    echo ""
    echo -e "${YELLOW}Payment Information:${NC}"
    echo "Payment Status: $PAYMENT_STATUS"
    echo "Amount Paid: \$$AMOUNT_PAID"
    echo "Balance Due: \$$BALANCE_DUE"
else
    echo ""
    echo -e "${RED}✗ Failed to finalize invoice${NC}"
    ERROR=$(echo "$RESPONSE" | jq -r '.error // empty')
    if [ -n "$ERROR" ]; then
        echo "Error: $ERROR"
    fi
fi
