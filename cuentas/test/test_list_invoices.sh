#!/bin/bash

# Usage: ./test_list_invoices.sh <company_id> [status] [client_id] [payment_status]
# Example: ./test_list_invoices.sh abc123
# Example: ./test_list_invoices.sh abc123 draft
# Example: ./test_list_invoices.sh abc123 finalized client456

if [ "$#" -lt 1 ]; then
    echo "Usage: $0 <company_id> [status] [client_id] [payment_status]"
    echo "Example: $0 abc123"
    echo "Example: $0 abc123 draft"
    echo "Example: $0 abc123 finalized client456 unpaid"
    exit 1
fi

COMPANY_ID=$1
STATUS=${2:-}
CLIENT_ID=${3:-}
PAYMENT_STATUS=${4:-}

API_URL="http://localhost:8080/v1/invoices"

# Build query parameters
PARAMS="?"
[ -n "$STATUS" ] && PARAMS+="status=$STATUS&"
[ -n "$CLIENT_ID" ] && PARAMS+="client_id=$CLIENT_ID&"
[ -n "$PAYMENT_STATUS" ] && PARAMS+="payment_status=$PAYMENT_STATUS&"

# Remove trailing & or ?
PARAMS=$(echo "$PARAMS" | sed 's/[?&]$//')

echo "Listing invoices..."
echo "Company ID: $COMPANY_ID"
[ -n "$STATUS" ] && echo "Status filter: $STATUS"
[ -n "$CLIENT_ID" ] && echo "Client ID filter: $CLIENT_ID"
[ -n "$PAYMENT_STATUS" ] && echo "Payment Status filter: $PAYMENT_STATUS"
echo ""

curl -s -X GET "$API_URL$PARAMS" \
    -H "X-Company-ID: $COMPANY_ID" | jq '.'
