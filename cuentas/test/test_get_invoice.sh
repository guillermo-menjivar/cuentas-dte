#!/bin/bash

# Usage: ./test_get_invoice.sh <company_id> <invoice_id>

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <company_id> <invoice_id>"
    exit 1
fi

COMPANY_ID=$1
INVOICE_ID=$2
API_URL="http://localhost:8080/v1/invoices/$INVOICE_ID"

echo "Fetching invoice..."
echo "Company ID: $COMPANY_ID"
echo "Invoice ID: $INVOICE_ID"
echo ""

curl -s -X GET "$API_URL" \
    -H "X-Company-ID: $COMPANY_ID" | jq '.'
