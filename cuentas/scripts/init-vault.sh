#!/bin/bash

# Wait for Vault to be ready
echo "Waiting for Vault to be ready..."

# Try different ways to reach Vault
VAULT_URLS=("http://vault:8200" "http://cuentas-vault:8200" "http://localhost:8200")

VAULT_READY=false
for attempt in {1..30}; do
    for url in "${VAULT_URLS[@]}"; do
        echo "Attempt $attempt: Trying $url"
        if curl --connect-timeout 5 "$url/v1/sys/health" > /dev/null 2>&1; then
            echo "Vault is ready at $url!"
            export VAULT_ADDR="$url"
            VAULT_READY=true
            break
        fi
    done
    
    if [ "$VAULT_READY" = true ]; then
        break
    fi
    
    echo "We are here..."
    echo "Vault is not ready yet, waiting..."
    sleep 3
done

if [ "$VAULT_READY" = false ]; then
    echo "Failed to connect to Vault after 30 attempts"
    exit 1
fi

# Set root token
export VAULT_TOKEN="vault-root-token"

echo "Testing Vault authentication..."
if ! vault auth -method=token token="$VAULT_TOKEN" > /dev/null 2>&1; then
    echo "Failed to authenticate with Vault"
    exit 1
fi

echo "Vault authentication successful!"

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

if [ $? -ne 0 ]; then
    echo "Failed to create policy"
    exit 1
fi

# Create a token for the cuentas app
echo "Creating token for cuentas app..."
APP_TOKEN=$(vault token create \
  -policy=cuentas-policy \
  -ttl=24h \
  -renewable=true \
  -display-name="cuentas-app" \
  -format=json 2>/dev/null | jq -r '.auth.client_token' 2>/dev/null)

if [ -z "$APP_TOKEN" ] || [ "$APP_TOKEN" = "null" ]; then
    echo "Failed to create app token, trying alternative method..."
    # Fallback: try without jq
    TOKEN_OUTPUT=$(vault token create \
      -policy=cuentas-policy \
      -ttl=24h \
      -renewable=true \
      -display-name="cuentas-app" 2>/dev/null)
    
    APP_TOKEN=$(echo "$TOKEN_OUTPUT" | grep "token " | head -1 | awk '{print $2}')
    
    if [ -z "$APP_TOKEN" ]; then
        echo "Failed to create app token with fallback method"
        exit 1
    fi
fi

echo "Created token: ${APP_TOKEN:0:10}..."

# Write the token to a file that the app can read
echo "$APP_TOKEN" > /vault/data/app-token

echo "Vault initialization complete!"
echo "App token saved to /vault/data/app-token"
