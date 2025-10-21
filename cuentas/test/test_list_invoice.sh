#!/bin/bash

# Test script to list invoices with various filters

if [ "$#" -lt 1 ]; then
    echo "Usage: $0 <company_id> [status] [dte_status] [client_id]"
    echo "Example: $0 abc-123"
    echo "Example: $0 abc-123 finalized"
    echo "Example: $0 abc-123 finalized not_submitted"
    exit 1
fi

COMPANY_ID=$1
STATUS=$2
DTE_STATUS=$3
CLIENT_ID=$4

API_URL="http://localhost:8080/v1/invoices"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Build query parameters
PARAMS=""
if [ -n "$STATUS" ]; then
    PARAMS="status=$STATUS"
fi
if [ -n "$DTE_STATUS" ]; then
    [ -n "$PARAMS" ] && PARAMS="${PARAMS}&"
    PARAMS="${PARAMS}dte_status=$DTE_STATUS"
fi
if [ -n "$CLIENT_ID" ]; then
    [ -n "$PARAMS" ] && PARAMS="${PARAMS}&"
    PARAMS="${PARAMS}client_id=$CLIENT_ID"
fi

if [ -n "$PARAMS" ]; then
    API_URL="${API_URL}?${PARAMS}"
fi

echo -e "${BLUE}=== Listing Invoices ===${NC}"
echo "Company ID: $COMPANY_ID"
[ -n "$STATUS" ] && echo "Status Filter: $STATUS"
[ -n "$DTE_STATUS" ] && echo "DTE Status Filter: $DTE_STATUS"
[ -n "$CLIENT_ID" ] && echo "Client ID Filter: $CLIENT_ID"
echo ""

# Make the API call
RESPONSE=$(curl -s -X GET "$API_URL" \
    -H "X-Company-ID: $COMPANY_ID")

echo "Response:"
echo "$RESPONSE"
echo "$RESPONSE" | jq '.'

# Display summary
COUNT=$(echo "$RESPONSE" | jq '.count // 0')

if [ "$COUNT" -gt 0 ]; then
    echo ""
    echo -e "${GREEN}Found $COUNT invoice(s)${NC}"
    echo ""
    echo -e "${YELLOW}Summary:${NC}"
    
    # Display each invoice in a table format
    echo "$RESPONSE" | jq -r '.invoices[] | 
        "ID: \(.id)\n" +
        "Number: \(.invoice_number)\n" +
        "Client: \(.client_name)\n" +
        "Total: $\(.total)\n" +
        "Status: \(.status)\n" +
        "DTE Status: \(.dte_status // "N/A")\n" +
        "Código Gen: \(.dte_codigo_generacion // "N/A")\n" +
        "Número Control: \(.dte_numero_control // "N/A")\n" +
        "Created: \(.created_at)\n" +
        "DTE Sello \(.dte_sello)\n" +
        "DTE Tipo \(.dte_type)\n" +
        "Finalized: \(.finalized_at // "N/A")\n" +
        "---"'
else
    echo ""
    echo -e "${YELLOW}No invoices found${NC}"
fi
