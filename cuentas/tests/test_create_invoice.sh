#!/bin/bash

# Usage: ./test_create_invoice.sh <company_id> <establishment_id> <pos_id> <client_id> <item_id> [item_id2 ...]
# Example: ./test_create_invoice.sh abc123 est456 pos789 client101 item201 item202

if [ "$#" -lt 5 ]; then
    echo "Usage: $0 <company_id> <establishment_id> <pos_id> <client_id> <item_id> [item_id2 ...]"
    echo "Example: $0 abc123 est456 pos789 client101 item201"
    echo ""
    echo "Tip: Use ./test_list_establishments.sh to find your establishment and POS IDs"
    exit 1
fi

COMPANY_ID=$1
ESTABLISHMENT_ID=$2
POS_ID=$3
CLIENT_ID=$4
shift 4  # Remove first four arguments
ITEMS=("$@")  # Remaining arguments are item IDs

API_URL="http://localhost:8080/v1/invoices"

# Build line items JSON array
LINE_ITEMS="["
for i in "${!ITEMS[@]}"; do
    if [ $i -gt 0 ]; then
        LINE_ITEMS+=","
    fi
    LINE_ITEMS+=$(cat <<EOF
{
    "item_id": "${ITEMS[$i]}",
    "quantity": 2,
    "discount_percentage": 5
}
EOF
)
done
LINE_ITEMS+="]"

# Create the invoice JSON
REQUEST_BODY=$(cat <<EOF
{
    "establishment_id": "$ESTABLISHMENT_ID",
    "point_of_sale_id": "$POS_ID",
    "client_id": "$CLIENT_ID",
    "payment_terms": "cash",
    "notes": "Test invoice created from script",
    "payment_method": "01",
    "contact_email": "test@example.com",
    "contact_whatsapp": "+50312345678",
    "line_items": $LINE_ITEMS
}
EOF
)

echo "Creating invoice..."
echo "Company ID: $COMPANY_ID"
echo "Establishment ID: $ESTABLISHMENT_ID"
echo "Point of Sale ID: $POS_ID"
echo "Client ID: $CLIENT_ID"
echo "Items: ${ITEMS[@]}"
echo ""

# Make the API call
RESPONSE=$(curl -s -X POST "$API_URL" \
    -H "Content-Type: application/json" \
    -H "X-Company-ID: $COMPANY_ID" \
    -d "$REQUEST_BODY")

echo "Response:"
echo "$RESPONSE" | jq '.'

# Extract invoice ID if successful
INVOICE_ID=$(echo "$RESPONSE" | jq -r '.id // empty')

if [ -n "$INVOICE_ID" ]; then
    echo ""
    echo "✓ Invoice created successfully!"
    echo "Invoice ID: $INVOICE_ID"
    echo "Establishment ID: $ESTABLISHMENT_ID"
    echo "Point of Sale ID: $POS_ID"
    
    echo ""
    echo "To view this invoice, run:"
    echo "curl -H 'X-Company-ID: $COMPANY_ID' http://localhost:8080/v1/invoices/$INVOICE_ID | jq '.'"
else
    echo ""
    echo "✗ Failed to create invoice"
    ERROR=$(echo "$RESPONSE" | jq -r '.error // empty')
    if [ -n "$ERROR" ]; then
        echo "Error: $ERROR"
    fi
fi
