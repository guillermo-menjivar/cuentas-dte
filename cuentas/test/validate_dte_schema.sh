#!/bin/bash
set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "=== DTE JSON Schema Validator ==="
echo ""

# Check if ajv-cli is installed
if ! command -v ajv &> /dev/null; then
    echo -e "${YELLOW}Installing ajv-cli for JSON schema validation...${NC}"
    npm install -g ajv-cli ajv-formats
fi

# Extract just the dteJson from test_dte.json
echo "Extracting DTE from payload..."
jq '.dteJson' test_dte.json > temp_dte.json

echo "Validating DTE against JSON Schema..."
echo ""

# Validate
if ajv validate -s fe-fc-v1.json -d temp_dte.json --spec=draft7 2>&1 | tee validation_output.txt; then
    echo ""
    echo -e "${GREEN}✅ DTE is VALID according to schema!${NC}"
    rm -f temp_dte.json validation_output.txt
    exit 0
else
    echo ""
    echo -e "${RED}❌ DTE validation FAILED!${NC}"
    echo ""
    echo "Errors found:"
    cat validation_output.txt
    rm -f temp_dte.json validation_output.txt
    exit 1
fi
