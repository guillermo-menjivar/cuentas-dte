#!/bin/bash

# Wait for Vault to be ready
echo "Waiting for Vault to be ready..."
until curl -s http://vault:8200/v1/sys/health > /dev/null 2>&1; do
    echo "Vault is not ready yet, waiting..."
    sleep 2
done

echo "Vault is ready!"

# Set Vault address and root token
export VAULT_ADDR="http://vault:8200"
export VAULT_TOKEN="vault-root-token"

# Create a policy for the cuentas app
echo "Creating cuentas policy..."
vault policy write cuentas-policy - <<EOF
# Allow read/write access to company secrets
path "secret/data/companies/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

path "secret/metadata/companies/*" {
  capabilities = ["list"]
}

# Allow listing companies
path "secret/metadata/companies" {
  capabilities = ["list"]
}
EOF

# Create a token for the cuentas app
echo "Creating token for cuentas app..."
APP_TOKEN=$(vault token create \
  -policy=cuentas-policy \
  -ttl=24h \
  -renewable=true \
  -display-name="cuentas-app" \
  -format=json | jq -r '.auth.client_token')

if [ -z "$APP_TOKEN" ] || [ "$APP_TOKEN" = "null" ]; then
    echo "Failed to create app token"
    exit 1
fi

echo "Created token: $APP_TOKEN"

# Write the token to a file that the app can read
echo "$APP_TOKEN" > /vault/data/app-token

echo "Vault initialization complete!"
echo "App token saved to /vault/data/app-token"
