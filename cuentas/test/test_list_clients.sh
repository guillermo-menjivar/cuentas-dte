#!/bin/bash

# Usage: ./test_list_clients.sh <company_id>

if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <company_id>"
    exit 1
fi

COMPANY_ID=$1
API_URL="http://localhost:8080/v1/clients"

echo "Listing clients..."
echo "Company ID: $COMPANY_ID"
echo ""

curl -s -X GET "$API_URL" \
    -H "X-Company-ID: $COMPANY_ID" | jq '.'
