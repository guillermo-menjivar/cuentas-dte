#!/bin/sh

echo "=== Vault Init Debug ==="
echo "Container hostname: $(hostname)"

# Test network connectivity
echo "=== Testing network connectivity ==="
echo "Can we resolve 'vault'?"
VAULT_IP=$(nslookup vault | grep 'Address:' | tail -1 | awk '{print $2}')
echo "Vault IP resolved to: $VAULT_IP"

echo "Testing Vault connectivity..."
VAULT_ADDR=""

# Try different combinations
VAULT_URLS="http://vault:8200 http://$VAULT_IP:8200 http://cuentas-vault:8200"

for url in $VAULT_URLS; do
    echo "Testing: $url"
    if curl -s --connect-timeout 5 --max-time 10 "$url/v1/sys/health" > /dev/null 2>&1; then
        echo "SUCCESS: $url is reachable"
        VAULT_ADDR="$url"
        break
    else
        echo "FAILED: $url is not reachable"
    fi
done

if [ -z "$VAULT_ADDR" ]; then
    echo "ERROR: Could not reach Vault at any URL"
    echo "Let's check what's listening on port 8200..."
    echo "Testing direct curl with verbose output:"
    curl -v --connect-timeout 5 "http://$VAULT_IP:8200/v1/sys/health" || echo "Direct IP curl failed"
    exit 1
fi

echo "Using Vault at: $VAULT_ADDR"
export VAULT_ADDR
export VAULT_TOKEN="vault-root-token"

# Test authentication
echo "Testing Vault authentication..."
vault status

# Create simple app token
echo "Creating simple app token..."
APP_TOKEN=$(vault token create -ttl=24h -format=json | jq -r '.auth.client_token' 2>/dev/null)

if [ -n "$APP_TOKEN" ] && [ "$APP_TOKEN" != "null" ]; then
    echo "$APP_TOKEN" > /vault/data/app-token
    echo "Token created successfully"
else
    echo "Failed to create token"
    exit 1
fi

echo "Vault initialization complete!"
