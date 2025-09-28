#!/bin/sh

echo "=== Cuentas App Startup ==="

echo "Waiting for Vault token to be created..."

# Wait for the app token to be available
MAX_ATTEMPTS=30
ATTEMPT=0

while [ ! -f /vault/data/app-token ] && [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    ATTEMPT=$((ATTEMPT + 1))
    echo "Attempt $ATTEMPT/$MAX_ATTEMPTS: Waiting for Vault token..."
    sleep 2
done

if [ ! -f /vault/data/app-token ]; then
    echo "ERROR: Vault token not found after $MAX_ATTEMPTS attempts"
    exit 1
fi

# Read the app token
APP_TOKEN=$(cat /vault/data/app-token)

if [ -z "$APP_TOKEN" ]; then
    echo "ERROR: Failed to read app token from file"
    exit 1
fi

echo "Successfully read Vault token"

# Set the token as environment variable
export CUENTAS_VAULT_TOKEN="$APP_TOKEN"

# Start the app
echo "Starting Cuentas application..."
exec ./cuentas serve
