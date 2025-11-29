#!/bin/bash
# finalize_all_invoices.sh

# First, get the invoices JSON (adjust the endpoint as needed)
INVOICES_JSON=$(curl -s -H "X-Company-ID: ddc26f8f-3dec-46fc-92bf-cfd0a3b2a78d" \
  "http://localhost:8080/v1/invoices?status=draft")

# Parse and iterate through each invoice
echo "$INVOICES_JSON" | jq -r '.invoices[] | "\(.company_id) \(.id) \(.balance_due)"' | while read -r company_id invoice_id balance_due; do
  echo "================================================"
  echo "Finalizing Invoice: $invoice_id"
  echo "Company: $company_id"
  echo "Balance Due: $balance_due"
  echo "================================================"
  
  bash test_finalize_invoice_with_payment.sh "$company_id" "$invoice_id" "$balance_due" "01"
  
  echo ""
  echo "Waiting 2 seconds before next invoice..."
  sleep 2
done

echo "✅ All invoices finalized!"
