#!/bin/bash

# Comprehensive test script for Nota de Crédito
# Usage: ./test_nota_credito_comprehensive.sh <company_id>
# Example: ./test_nota_credito_comprehensive.sh e65fb18b-1944-483c-b877-f11f8f4ad7c3

set -e

# Check arguments
if [ "$#" -lt 1 ]; then
    echo "Usage: $0 <company_id>"
    echo "Example: $0 e65fb18b-1944-483c-b877-f11f8f4ad7c3"
    exit 1
fi

# Configuration
API_URL="http://localhost:8080/v1"
COMPANY_ID=$1

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "=========================================================="
echo "Nota de Crédito - Comprehensive Test"
echo "=========================================================="
echo ""
echo "Company ID: $COMPANY_ID"
echo ""

# Step 1: List all invoices and find finalized CCFs
echo -e "${BLUE}Step 1: Searching for finalized CCF invoices...${NC}"
echo ""

INVOICES_RESPONSE=$(curl -s -X GET "$API_URL/invoices?status=finalized" \
    -H "X-Company-ID: $COMPANY_ID")

# Count total invoices
TOTAL_INVOICES=$(echo "$INVOICES_RESPONSE" | jq '.invoices | length')
echo "Found $TOTAL_INVOICES finalized invoice(s)"

if [ "$TOTAL_INVOICES" -eq 0 ]; then
    echo -e "${RED}❌ No finalized invoices found!${NC}"
    echo ""
    echo "Please create and finalize an invoice first:"
    echo "  1. Create an invoice"
    echo "  2. Finalize it"
    echo "  3. Run this script again"
    exit 1
fi

# Filter for CCF (type "03")
CCF_INVOICES=$(echo "$INVOICES_RESPONSE" | jq '[.invoices[] | select(.dte_type == "03")]')
CCF_COUNT=$(echo "$CCF_INVOICES" | jq 'length')

echo "Found $CCF_COUNT CCF invoice(s) (type 03)"
echo ""

if [ "$CCF_COUNT" -eq 0 ]; then
    echo -e "${RED}❌ No finalized CCF invoices found!${NC}"
    echo ""
    echo "Nota de Crédito can only reference CCF invoices (type 03)."
    echo "Found invoices of types:"
    echo "$INVOICES_RESPONSE" | jq -r '.invoices[] | "\(.invoice_number) - Type: \(.dte_type // "unknown")"'
    exit 1
fi

# Select first CCF
echo -e "${GREEN}✓ Found CCF invoices${NC}"
echo ""
echo "Available CCFs:"
echo "$CCF_INVOICES" | jq -r '.[] | "\(.invoice_number) | \(.client_name) | Total: $\(.total) | Created: \(.created_at | split("T")[0])"'
echo ""

# Use first CCF
CCF_ID=$(echo "$CCF_INVOICES" | jq -r '.[0].id')
CCF_NUMBER=$(echo "$CCF_INVOICES" | jq -r '.[0].invoice_number')

echo -e "${YELLOW}Selected CCF: $CCF_NUMBER (ID: $CCF_ID)${NC}"
echo ""

# Step 2: Get full invoice details with line items
echo -e "${BLUE}Step 2: Fetching CCF details with line items...${NC}"
echo ""

CCF_DETAILS=$(curl -s -X GET "$API_URL/invoices/$CCF_ID" \
    -H "X-Company-ID: $COMPANY_ID")

# Display CCF summary
echo "CCF Details:"
echo "  Number: $(echo "$CCF_DETAILS" | jq -r '.invoice_number')"
echo "  Client: $(echo "$CCF_DETAILS" | jq -r '.client_name')"
echo "  Status: $(echo "$CCF_DETAILS" | jq -r '.status')"
echo "  Subtotal: \$$(echo "$CCF_DETAILS" | jq -r '.subtotal')"
echo "  Total: \$$(echo "$CCF_DETAILS" | jq -r '.total')"
echo ""

# Check line items
LINE_ITEMS_COUNT=$(echo "$CCF_DETAILS" | jq '.line_items | length')

if [ "$LINE_ITEMS_COUNT" -eq 0 ]; then
    echo -e "${RED}❌ CCF has no line items!${NC}"
    exit 1
fi

echo "Line Items ($LINE_ITEMS_COUNT):"
echo "$CCF_DETAILS" | jq -r '.line_items[] | "  [\(.line_number)] \(.item_sku) - \(.item_name) | Qty: \(.quantity) @ $\(.unit_price) = $\(.line_total)"'
echo ""

# Select first line item for credit
FIRST_LINE_ITEM=$(echo "$CCF_DETAILS" | jq '.line_items[0]')
LINE_ITEM_ID=$(echo "$FIRST_LINE_ITEM" | jq -r '.id')
LINE_ITEM_NAME=$(echo "$FIRST_LINE_ITEM" | jq -r '.item_name')
LINE_ITEM_SKU=$(echo "$FIRST_LINE_ITEM" | jq -r '.item_sku')
LINE_ITEM_PRICE=$(echo "$FIRST_LINE_ITEM" | jq -r '.unit_price')
LINE_ITEM_QTY=$(echo "$FIRST_LINE_ITEM" | jq -r '.quantity')

echo -e "${YELLOW}Test Item: $LINE_ITEM_ID ($LINE_ITEM_NAME) as test item${NC}"
echo "  SKU: $LINE_ITEM_SKU"
echo "  Original Price: \$$LINE_ITEM_PRICE"
echo "  Original Quantity: $LINE_ITEM_QTY"
echo ""

# Step 3: Create Nota de Crédito
# Credit half the quantity at full price (partial credit)
QUANTITY_CREDITED=$(echo "$LINE_ITEM_QTY / 2" | bc)
CREDIT_AMOUNT="$LINE_ITEM_PRICE"

echo -e "${BLUE}Step 3: Creating Nota de Crédito...${NC}"
echo ""
echo "Credit Details:"
echo "  Quantity Credited: $QUANTITY_CREDITED (of $LINE_ITEM_QTY)"
echo "  Credit Amount: \$$CREDIT_AMOUNT per unit"
echo "  Total credit: \$$CREDIT_AMOUNT × $QUANTITY_CREDITED = \$$(echo "$CREDIT_AMOUNT * $QUANTITY_CREDITED" | bc)"
echo ""

NOTA_REQUEST=$(cat <<EOF
{
  "ccf_ids": ["$CCF_ID"],
  "credit_reason": "defect",
  "credit_description": "Cliente reportó defecto de fábrica en $QUANTITY_CREDITED unidades - automated test",
  "line_items": [
    {
      "related_ccf_id": "$CCF_ID",
      "ccf_line_item_id": "$LINE_ITEM_ID",
      "quantity_credited": $QUANTITY_CREDITED,
      "credit_amount": $CREDIT_AMOUNT,
      "credit_reason": "Material defect - automated test"
    }
  ],
  "payment_terms": "contado",
  "notes": "Automated test nota de crédito - $(date '+%Y-%m-%d %H:%M:%S')"
}
EOF
)

echo "Request payload:"
echo "$NOTA_REQUEST" | jq '.'
echo ""
echo "Sending request to: POST $API_URL/notas/credito"
echo ""

NOTA_RESPONSE=$(curl -s -X POST "$API_URL/notas/credito" \
    -H "X-Company-ID: $COMPANY_ID" \
    -H "Content-Type: application/json" \
    -d "$NOTA_REQUEST")

# Check for errors
if echo "$NOTA_RESPONSE" | jq -e '.error' > /dev/null 2>&1; then
    echo -e "${RED}❌ FAILED to create nota!${NC}"
    echo ""
    echo "Error response:"
    echo "$NOTA_RESPONSE" | jq '.'
    exit 1
fi

# Check if creation was successful
if echo "$NOTA_RESPONSE" | jq -e '.nota.id' > /dev/null 2>&1; then
    NOTA_ID=$(echo "$NOTA_RESPONSE" | jq -r '.nota.id')
    NOTA_NUMBER=$(echo "$NOTA_RESPONSE" | jq -r '.nota.nota_number')
    NOTA_STATUS=$(echo "$NOTA_RESPONSE" | jq -r '.nota.status')
    NOTA_SUBTOTAL=$(echo "$NOTA_RESPONSE" | jq -r '.nota.subtotal')
    NOTA_TAXES=$(echo "$NOTA_RESPONSE" | jq -r '.nota.total_taxes')
    NOTA_TOTAL=$(echo "$NOTA_RESPONSE" | jq -r '.nota.total')
    IS_FULL_ANNULMENT=$(echo "$NOTA_RESPONSE" | jq -r '.nota.is_full_annulment')
    
    echo -e "${GREEN}✅ SUCCESS! Nota de Crédito created${NC}"
    echo ""
    echo "=========================================================="
    echo "Nota Created"
    echo "=========================================================="
    echo "Nota ID: $NOTA_ID"
    echo "Nota Number: $NOTA_NUMBER"
    echo "Status: $NOTA_STATUS"
    echo "Full Annulment: $IS_FULL_ANNULMENT"
    echo "Credit Reason: $(echo "$NOTA_RESPONSE" | jq -r '.nota.credit_reason')"
    echo "Subtotal: \$$NOTA_SUBTOTAL"
    echo "Taxes (13%): \$$NOTA_TAXES"
    echo "Total: \$$NOTA_TOTAL"
    echo ""
else
    echo -e "${RED}❌ Unexpected response format${NC}"
    echo "$NOTA_RESPONSE" | jq '.'
    exit 1
fi

# Step 4: Get nota details
echo -e "${BLUE}Step 4: Fetching created nota details...${NC}"
echo ""

NOTA_DETAILS=$(curl -s -X GET "$API_URL/notas/credito/$NOTA_ID" \
    -H "X-Company-ID: $COMPANY_ID")

if echo "$NOTA_DETAILS" | jq -e '.id' > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Nota details retrieved${NC}"
    echo ""
    echo "Line Items:"
    echo "$NOTA_DETAILS" | jq -r '.line_items[] | "  • \(.original_item_name): \(.quantity_credited) units @ $\(.credit_amount) = $\(.line_total) total"'
    echo ""
    echo "Referenced CCFs:"
    echo "$NOTA_DETAILS" | jq -r '.ccf_references[] | "  • \(.ccf_number) (Date: \(.ccf_date))"'
    echo ""
else
    echo -e "${YELLOW}⚠️  Could not fetch nota details${NC}"
    echo "$NOTA_DETAILS" | jq '.'
fi

echo ""
echo "=========================================================="
echo "Finalizing Nota de Crédito..."
echo "=========================================================="
echo ""
echo "⏳ This may take a few seconds..."
echo ""

FINALIZE_RESPONSE=$(curl -s -X POST "$API_URL/notas/credito/$NOTA_ID/finalize" \
    -H "X-Company-ID: $COMPANY_ID" \
    -H "Content-Type: application/json")

if echo "$FINALIZE_RESPONSE" | jq -e '.nota.status' > /dev/null 2>&1; then
    FINAL_STATUS=$(echo "$FINALIZE_RESPONSE" | jq -r '.nota.status')
    DTE_STATUS=$(echo "$FINALIZE_RESPONSE" | jq -r '.nota.dte_status')
    DTE_NUMERO=$(echo "$FINALIZE_RESPONSE" | jq -r '.nota.dte_numero_control')
    
    if [ "$FINAL_STATUS" == "finalized" ]; then
        echo -e "${GREEN}✅ Nota finalized successfully!${NC}"
        echo ""
        echo "Status: $FINAL_STATUS"
        echo "DTE Status: $DTE_STATUS"
        echo "Numero Control: $DTE_NUMERO"
        
        # Check for sello recibido
        SELLO=$(echo "$FINALIZE_RESPONSE" | jq -r '.nota.dte_sello_recibido // empty')
        if [ -n "$SELLO" ] && [ "$SELLO" != "null" ]; then
            echo "Sello Recibido: ${SELLO:0:50}..."
        fi
        
        # Show codigo generacion
        CODIGO_GEN=$(echo "$FINALIZE_RESPONSE" | jq -r '.nota.dte_codigo_generacion // empty')
        if [ -n "$CODIGO_GEN" ] && [ "$CODIGO_GEN" != "null" ]; then
            echo "Código de Generación: $CODIGO_GEN"
        fi
    else
        echo -e "${YELLOW}⚠️  Status: $FINAL_STATUS${NC}"
        echo "$FINALIZE_RESPONSE" | jq '.'
    fi
else
    echo -e "${RED}❌ Finalization failed${NC}"
    echo "$FINALIZE_RESPONSE" | jq '.'
fi

echo ""
echo -e "${GREEN}=========================================================="
echo "Test Complete!"
echo "==========================================================${NC}"
echo ""
echo "Summary:"
echo "  • CCF Referenced: $CCF_NUMBER"
echo "  • Nota Created: $NOTA_NUMBER"
echo "  • Quantity Credited: $QUANTITY_CREDITED of $LINE_ITEM_QTY"
echo "  • Total Credit: \$$NOTA_TOTAL"
echo "  • Full Annulment: $IS_FULL_ANNULMENT"
echo ""
echo "Next Steps:"
echo "  1. Check commit log: GET /v1/dte/commit-log"
echo "  2. View nota: GET /v1/notas/credito/$NOTA_ID"
echo "  3. Verify in Hacienda portal"
echo ""
