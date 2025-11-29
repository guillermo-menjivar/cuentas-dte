#!/bin/bash
# temp_load_vault.sh

COMPANY_ID="12b51f1b-b7de-4844-9e09-2515035f6900"
FIRMADOR_PASSWORD="sdKC4uLduegSPT"
HC_PASSWORD="MF7HwttFuZ.*3RY"

echo "🔐 Restoring passwords to Vault..."

# Store firmador password (with VAULT_ADDR and VAULT_TOKEN env vars)
docker exec -e VAULT_ADDR=http://127.0.0.1:8200 -e VAULT_TOKEN=vault-root-token cuentas-vault vault kv put \
  secret/companies/${COMPANY_ID}_firmador/password \
  password="${FIRMADOR_PASSWORD}"
echo "✅ Firmador password stored"

# Store Hacienda password
docker exec -e VAULT_ADDR=http://127.0.0.1:8200 -e VAULT_TOKEN=vault-root-token cuentas-vault vault kv put \
  secret/companies/${COMPANY_ID}_hacienda/password \
  password="${HC_PASSWORD}"
echo "✅ Hacienda password stored"

echo "🎉 Done! Passwords restored to Vault"

# Verify
echo ""
echo "🔍 Verifying stored passwords..."
docker exec -e VAULT_ADDR=http://127.0.0.1:8200 -e VAULT_TOKEN=vault-root-token cuentas-vault vault kv get secret/companies/${COMPANY_ID}_firmador/password
docker exec -e VAULT_ADDR=http://127.0.0.1:8200 -e VAULT_TOKEN=vault-root-token cuentas-vault vault kv get secret/companies/${COMPANY_ID}_hacienda/password
