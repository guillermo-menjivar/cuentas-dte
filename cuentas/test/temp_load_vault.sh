#!/bin/bash
# temp_load_vault.sh

COMPANY_ID="12b51f1b-b7de-4844-9e09-2515035f6900"
FIRMADOR_PASSWORD="sdKC4uLduegSPT"
HC_PASSWORD="MF7HwttFuZ.*3RY"

echo "🔐 Restoring passwords to Vault..."

# Store firmador password
docker exec cuentas-vault vault kv put \
  secret/companies/${COMPANY_ID}_firmador/password \
  password="${FIRMADOR_PASSWORD}"
echo "✅ Firmador password stored"

# Store Hacienda password
docker exec cuentas-vault vault kv put \
  secret/companies/${COMPANY_ID}_hacienda/password \
  password="${HC_PASSWORD}"
echo "✅ Hacienda password stored"

echo "🎉 Done! Passwords restored to Vault"

# Verify
echo ""
echo "🔍 Verifying stored passwords..."
docker exec cuentas-vault vault kv get secret/companies/${COMPANY_ID}_firmador/password
docker exec cuentas-vault vault kv get secret/companies/${COMPANY_ID}_hacienda/password
