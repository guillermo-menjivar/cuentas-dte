#!/bin/bash

# Usage: ./test_find_consumidor_final.sh <company_id>
# Finds the Consumidor Final client for a company

if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <company_id>"
    exit 1
fi

COMPANY_ID=$1
API_URL="http://localhost:8080/v1/clients"

echo "Finding Consumidor Final client..."
echo "Company ID: $COMPANY_ID"
echo ""

RESPONSE=$(curl -s -X GET "$API_URL" -H "X-Company-ID: $COMPANY_ID")

CONSUMIDOR_FINAL=$(echo "$RESPONSE" | jq '.clients[] | select(.business_name == "Consumidor Final")')

if [ -n "$CONSUMIDOR_FINAL" ]; then
    echo "$CONSUMIDOR_FINAL" | jq '.'
    CONSUMIDOR_ID=$(echo "$CONSUMIDOR_FINAL" | jq -r '.id')
    echo ""
    echo "Consumidor Final ID: $CONSUMIDOR_ID"
else
    echo "Consumidor Final client not found"
fi
