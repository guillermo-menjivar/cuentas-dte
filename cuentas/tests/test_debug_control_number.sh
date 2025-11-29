#!/bin/bash

NUMERO="DTE-01-00010002-000000000000001"

echo "Testing numeroControl: $NUMERO"
echo "Length: ${#NUMERO}"
echo ""

# Extract parts
PREFIX=$(echo "$NUMERO" | cut -d'-' -f1,2)
ESTAB=$(echo "$NUMERO" | cut -d'-' -f3)
SEQ=$(echo "$NUMERO" | cut -d'-' -f4)

echo "Prefix: $PREFIX (should be 'DTE-01')"
echo "Establishment: $ESTAB (length: ${#ESTAB}, should be 8)"
echo "Sequence: $SEQ (length: ${#SEQ}, should be 15)"
echo ""

# Test regex
if [[ "$NUMERO" =~ ^DTE-01-[A-Z0-9]{8}-[0-9]{15}$ ]]; then
    echo "✅ Matches pattern!"
else
    echo "❌ Does NOT match pattern!"
    echo ""
    echo "Pattern: ^DTE-01-[A-Z0-9]{8}-[0-9]{15}$"
    echo "Your value: $NUMERO"
fi
