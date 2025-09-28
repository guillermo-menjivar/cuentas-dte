#!/bin/sh

echo "=== Vault Init Debug ==="
echo "Container hostname: $(hostname)"

# Test network connectivity
echo "=== Testing network connectivity ==="
echo "Can we resolve 'vault'?"
nslookup vault || echo "DNS resolution failed"

echo "Testing Vault connectivity..."
VAULT_ADDR=""

# Try vault:8200
echo "Testing: http://vault:8200"
if curl -s --connect-timeout 5 --max-time 10 "http://vault:8200/v1/sys/health" > /dev/null 2>&1; then
    echo "SUCCESS: vault:8200 is reachable"
    VAULT_ADDR="http://vault:8200"
else
    echo "FAILED: vault:8200 is not reachable"
fi

# Try cuentas-vault:8200 if first failed
if [ -z "$VAULT_ADDR" ]; then
    echo "Testing: http://cuentas-vault:8200"
    if curl -s --connect-timeout 5 --max-time 10 "http://cuentas-vault:8200/v1/sys/health" > /dev/null 2>&1; then
        echo "SUCCESS: cuentas-vault:8200 is reachable"
        VAULT_ADDR="http://cuentas-vault:8200"
    else
        echo "FAILED: cuentas-vault:8200 is not reachable"
    fi
fi

if [ -z "$VAULT_ADDR" ]; then
    echo "ERROR: Could not reach Vault at any URL"
    echo "Available network interfaces:"
    ip addr show
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
    echo "Token created successfully: ${APP_TOKEN%${APP_TOKEN#??????????}}..."
else
    echo "Failed to create token"
    exit 1
fi

echo "Vault initialization complete!"
