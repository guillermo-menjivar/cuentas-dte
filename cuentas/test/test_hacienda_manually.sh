#!/bin/bash
set -e

echo "=== Testing Full DTE Flow ==="
echo ""

# Step 1: Sign the DTE
echo "Step 1: Signing DTE with Firmador..."
SIGNED_JWT=$(curl -s -X POST http://167.172.230.154:8113/firmardocumento/ \
  -H "Content-Type: application/json" \
  -d @test_dte.json | jq -r '.body')

if [ "$SIGNED_JWT" == "null" ] || [ -z "$SIGNED_JWT" ]; then
    echo "❌ Failed to sign DTE"
    exit 1
fi

echo "✅ DTE Signed successfully!"
echo "JWT length: ${#SIGNED_JWT} characters"
echo "First 100 chars: ${SIGNED_JWT:0:100}..."
echo ""

# Step 2: Submit to Hacienda
echo "Step 2: Submitting to Hacienda TEST environment..."

# Create the submission payload
cat > hacienda_payload.json <<EOF
{
  "ambiente": "00",
  "idEnvio": 1,
  "version": 1,
  "tipoDte": "01",
  "documento": "$SIGNED_JWT",
  "codigoGeneracion": "B007E30D-339F-49F6-A889-B32F1A13CD1D"
}
EOF

# First, authenticate
echo "Authenticating with Hacienda..."
TOKEN=$(curl -s -X POST https://apitest.dtes.mh.gob.sv/seguridad/auth \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "user=06143005061013&pwd=sdKC4uLduegSPT" | jq -r '.body.token')

if [ "$TOKEN" == "null" ] || [ -z "$TOKEN" ]; then
    echo "❌ Failed to authenticate with Hacienda"
    exit 1
fi

echo "✅ Authenticated with Hacienda!"
echo ""

# Submit the DTE
echo "Submitting DTE to Hacienda..."
RESPONSE=$(curl -s -X POST https://apitest.dtes.mh.gob.sv/fesv/recepciondte \
  -H "Authorization: $TOKEN" \
  -H "Content-Type: application/json" \
  -H "User-Agent: CuentasApp/1.0" \
  -d @hacienda_payload.json)

echo ""
echo "=== HACIENDA RESPONSE ==="
echo "$RESPONSE" | jq .
echo ""

# Check the response
ESTADO=$(echo "$RESPONSE" | jq -r '.estado')

if [ "$ESTADO" == "PROCESADO" ]; then
    echo "✅ SUCCESS! DTE ACCEPTED BY HACIENDA!"
    echo "Estado: $(echo "$RESPONSE" | jq -r '.estado')"
    echo "Código de Generación: $(echo "$RESPONSE" | jq -r '.codigoGeneracion')"
    echo "Sello Recibido: $(echo "$RESPONSE" | jq -r '.selloRecibido')"
    echo "Fecha Procesamiento: $(echo "$RESPONSE" | jq -r '.fhProcesamiento')"
elif [ "$ESTADO" == "RECHAZADO" ]; then
    echo "❌ DTE REJECTED BY HACIENDA"
    echo "Reason: $(echo "$RESPONSE" | jq -r '.descripcionMsg')"
    exit 1
else
    echo "⚠️  Unexpected response from Hacienda"
    exit 1
fi

# Cleanup
rm -f hacienda_payload.json
