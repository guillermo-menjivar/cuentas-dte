#!/bin/bash

# Usage: ./create_nota_debito.sh <company_id> <ccf_id>
# Example: ./create_nota_debito.sh "a1b2c3d4-e5f6-4789-a012-bcdef1234567" "f9e8d7c6-b5a4-3210-9876-543210fedcba"

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <company_id> <ccf_id>"
    exit 1
fi

COMPANY_ID="$1"
CCF_ID="$2"

curl -X POST http://localhost:8080/v1/notas/debito \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: ${COMPANY_ID}" \
  -d "{
  \"ccf_ids\": [
    \"${CCF_ID}\"
  ],
  \"line_items\": [
    {
      \"related_ccf_id\": \"${CCF_ID}\",
      \"is_new_item\": true,
      \"inventory_item_id\": \"11223344-5566-7788-99aa-bbccddeeff00\",
      \"item_name\": \"Cargo por pago tardío\",
      \"quantity\": 1,
      \"unit_price\": 25.00,
      \"discount_amount\": 0
    }
  ],
  \"payment_terms\": \"immediate\",
  \"notes\": \"Cargo por mora - pago tardío\"
}"
