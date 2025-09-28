#!/bin/bash

echo "Waiting for Vault token to be created..."

# Wait for the app token to be available
while [ ! -f /vault/data/app-token ]; do
    echo "Waiting for Vault initialization to complete..."
    sleep 2
done

# Read the app token
APP_TOKEN=$(cat /vault/data/app-token)

if [ -z "$APP_TOKEN" ]; then
    echo "Failed to read app token"
    exit 1
fi

echo "Using Vault token: ${APP_TOKEN:0:10}..."

# Set the token as environment variable and start the app
export CUENTAS_VAULT_TOKEN="$APP_TOKEN"
exec ./cuentas serve
