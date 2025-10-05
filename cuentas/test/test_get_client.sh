#!/bin/bash

# Usage: ./test_get_client.sh <company_id> <client_id>

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <company_id> <client_id>"
    exit 1
fi

COMPANY_ID=$1
CLIENT_ID=$2
API_URL="http://localhost:8080/v1/clients/$CLIENT_ID"

echo "Fetching client..."
echo "Company ID: $COMPANY_ID"
echo "Client ID: $CLIENT_ID"
echo ""

curl -s -X GET "$API_URL" \
    -H "X-Company-ID: $COMPANY_ID" | jq '.'
