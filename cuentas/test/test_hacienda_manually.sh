#!/bin/bash
set -e

# ... [previous usage and argument parsing code] ...

echo "=== Testing Full DTE Flow ==="
echo ""

# Step 0: Validate DTE against schema
echo "Step 0: Validating DTE structure..."
if ! ./validate_dte_schema.sh; then
    echo "❌ DTE validation failed. Fix errors before submitting."
    exit 1
fi
echo "✅ DTE structure is valid!"
echo ""

# Step 1: Sign the DTE
echo "Step 1: Signing DTE with Firmador..."

# Usage function
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -t, --token TOKEN    Use existing Hacienda auth token (optional)"
    echo "  -h, --help          Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                                    # Authenticate and submit"
    echo "  $0 --token 'Bearer eyJhbGci...'       # Use existing token"
    exit 1
}

# Parse command line arguments
HACIENDA_TOKEN=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -t|--token)
            HACIENDA_TOKEN="$2"
            shift 2
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo "Unknown option: $1"
            usage
            ;;
    esac
done

echo "=== Testing Full DTE Flow ==="
echo ""

# Step 1: Sign the DTE
echo "Step 1: Signing DTE with Firmador..."
SIGNED_JWT=$(curl -s -X POST http://167.172.230.154:8113/firmardocumento/ \
  -H "Content-Type: application/json" \
  -d @test_dte.json | jq -r '.body')

if [ "$SIGNED_JWT" == "null" ] || [ -z "$SIGNED_JWT" ]; then
    echo "❌ Failed to sign DTE"
    curl -s -X POST http://167.172.230.154:8113/firmardocumento/ \
      -H "Content-Type: application/json" \
      -d @test_dte.json | jq .
    exit 1
fi

echo "✅ DTE Signed successfully!"
echo "JWT length: ${#SIGNED_JWT} characters"
echo "First 100 chars: ${SIGNED_JWT:0:100}..."
echo ""

# Step 2: Get or use Hacienda token
if [ -z "$HACIENDA_TOKEN" ]; then
    echo "Step 2: Authenticating with Hacienda..."
    AUTH_RESPONSE=$(curl -s -X POST https://apitest.dtes.mh.gob.sv/seguridad/auth \
      -H "Content-Type: application/x-www-form-urlencoded" \
      -d "user=06143005061013&pwd=sdKC4uLduegSPT")
    
    HACIENDA_TOKEN=$(echo "$AUTH_RESPONSE" | jq -r '.body.token')
    
    if [ "$HACIENDA_TOKEN" == "null" ] || [ -z "$HACIENDA_TOKEN" ]; then
        echo "❌ Failed to authenticate with Hacienda"
        echo "$AUTH_RESPONSE" | jq .
        exit 1
    fi
    
    echo "✅ Authenticated with Hacienda!"
    echo "Token: ${HACIENDA_TOKEN:0:50}..."
    echo ""
    echo "💡 TIP: Reuse this token for 24 hours with:"
    echo "   $0 --token '$HACIENDA_TOKEN'"
    echo ""
else
    echo "Step 2: Using provided Hacienda token..."
    echo "Token: ${HACIENDA_TOKEN:0:50}..."
    echo ""
fi

# Step 3: Submit to Hacienda
echo "Step 3: Submitting DTE to Hacienda TEST environment..."

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

# Submit the DTE
RESPONSE=$(curl -s -X POST https://apitest.dtes.mh.gob.sv/fesv/recepciondte \
  -H "Authorization: $HACIENDA_TOKEN" \
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
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Estado: $(echo "$RESPONSE" | jq -r '.estado')"
    echo "Código de Generación: $(echo "$RESPONSE" | jq -r '.codigoGeneracion')"
    echo "Sello Recibido: $(echo "$RESPONSE" | jq -r '.selloRecibido')"
    echo "Fecha Procesamiento: $(echo "$RESPONSE" | jq -r '.fhProcesamiento')"
    echo "Código Mensaje: $(echo "$RESPONSE" | jq -r '.codigoMsg')"
    echo "Descripción: $(echo "$RESPONSE" | jq -r '.descripcionMsg')"
    
    # Check for observations
    OBSERVACIONES=$(echo "$RESPONSE" | jq -r '.observaciones[]' 2>/dev/null)
    if [ ! -z "$OBSERVACIONES" ]; then
        echo ""
        echo "⚠️  Observaciones:"
        echo "$OBSERVACIONES"
    fi
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
elif [ "$ESTADO" == "RECHAZADO" ]; then
    echo "❌ DTE REJECTED BY HACIENDA"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Código: $(echo "$RESPONSE" | jq -r '.codigoMsg')"
    echo "Reason: $(echo "$RESPONSE" | jq -r '.descripcionMsg')"
    
    # Show observations if any
    OBSERVACIONES=$(echo "$RESPONSE" | jq -r '.observaciones[]' 2>/dev/null)
    if [ ! -z "$OBSERVACIONES" ]; then
        echo ""
        echo "Observaciones:"
        echo "$OBSERVACIONES"
    fi
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    exit 1
else
    echo "⚠️  Unexpected response from Hacienda"
    echo "Estado: $ESTADO"
    exit 1
fi

# Cleanup
rm -f hacienda_payload.json

echo ""
echo "🎉 Test completed successfully!"
