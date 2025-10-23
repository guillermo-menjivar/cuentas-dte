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
    \"2EADAAE2-40C3-4910-B0E5-A81AD4F117EF\"
  ],
  \"line_items\": [
    {
      \"related_ccf_id\": \"2EADAAE2-40C3-4910-B0E5-A81AD4F117EF\",
      \"is_new_item\": false,
      \"ccf_line_item_id\": \"25161abf-b542-435d-a4ef-5158536484d4\",
      \"inventory_item_id\": \"910a13c5-70ae-489b-ba0e-b72b31bfc1fd\",
      \"item_name\": \"Dell Latitude 5520 Laptop\",
      \"quantity\": 1,
      \"adjustment_amount\": 4876.00,
      \"discount_amount\": 0
    }
  ],
  \"payment_terms\": \"immediate\",
  \"notes\": \"Cargo por mora - pago tardío\"
}"
