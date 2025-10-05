#!/bin/bash

# Usage: ./test_get_company.sh <company_id>

if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <company_id>"
    exit 1
fi

COMPANY_ID=$1
API_URL="http://localhost:8080/v1/companies/$COMPANY_ID"

echo "Fetching company..."
echo "Company ID: $COMPANY_ID"
echo ""

curl -s -X GET "$API_URL" \
    -H "X-Company-ID: $COMPANY_ID" | jq '.'
