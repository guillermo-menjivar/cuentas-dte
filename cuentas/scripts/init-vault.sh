#!/bin/sh

echo "=== Vault Init Debug ==="
echo "Container hostname: $(hostname)"

# Test network connectivity
echo "=== Testing network connectivity ==="
echo "Can we resolve 'vault'?"
VAULT_IP=$(nslookup vault | grep 'Address:' | tail -1 | awk '{print $2}')
echo "Vault IP resolved to: $VAULT_IP"

echo "Testing Vault connectivity using vault CLI..."
VAULT_ADDR=""

# Try different vault addresses
for addr in "http://vault:8200" "http://$VAULT_IP:8200" "http://cuentas-vault:8200"; do
    echo "Testing: $addr"
    export VAULT_ADDR="$addr"
    
    if vault status > /dev/null 2>&1; then
        echo "SUCCESS: $addr is reachable"
        break
    else
        echo "FAILED: $addr is not reachable"
        VAULT_ADDR=""
    fi
done

if [ -z "$VAULT_ADDR" ]; then
    echo "ERROR: Could not reach Vault at any URL"
    echo "Last vault status output:"
    vault status
    exit 1
fi

echo "Using Vault at: $VAULT_ADDR"
export VAULT_TOKEN="vault-root-token"

# Test authentication
echo "Testing Vault authentication..."
vault status

# Create simple app token
echo "Creating simple app token..."
APP_TOKEN=$(vault token create -ttl=24h -format=json | jq -r '.auth.client_token' 2>/dev/null)

# Fallback if jq is not available
if [ -z "$APP_TOKEN" ] || [ "$APP_TOKEN" = "null" ]; then
    echo "jq not available, using vault token create without json format..."
    TOKEN_OUTPUT=$(vault token create -ttl=24h 2>/dev/null)
    APP_TOKEN=$(echo "$TOKEN_OUTPUT" | grep '^token ' | awk '{print $2}')
fi

if [ -n "$APP_TOKEN" ] && [ "$APP_TOKEN" != "null" ]; then
    echo "$APP_TOKEN" > /vault/data/app-token
    echo "Token created successfully"
else
    echo "Failed to create token"
    exit 1
fi

echo "Vault initialization complete!"
