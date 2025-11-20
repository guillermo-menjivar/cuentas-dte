#!/bin/bash

################################################################################
# Simple S3 Directory Structure Setup
# 
# Creates the initial directory structure for DTE storage in an existing S3 bucket
#
# Usage:
#   ./create_s3_structure.sh <bucket-name>
#
# Example:
#   ./create_s3_structure.sh cuentas-dtes-prod
################################################################################

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

BUCKET_NAME="${1}"

if [ -z "$BUCKET_NAME" ]; then
    echo "Usage: $0 <bucket-name>"
    echo "Example: $0 cuentas-dtes-prod"
    exit 1
fi

echo -e "${CYAN}Creating directory structure in: ${BUCKET_NAME}${NC}"
echo ""

# Create directory structure by uploading README files
echo -e "${BLUE}Creating dtes/ directory...${NC}"
echo "DTE Downloads - Individual DTE retrieval" | aws s3 cp - "s3://${BUCKET_NAME}/dtes/README.txt"

echo -e "${BLUE}Creating analytics/ directory...${NC}"
echo "DTE Analytics - Partitioned for Athena queries" | aws s3 cp - "s3://${BUCKET_NAME}/analytics/README.txt"

# Create structure documentation
cat > /tmp/structure.txt <<EOF
S3 Directory Structure for DTE Storage
=======================================

dtes/
  - Fast downloads path
  - Structure: dtes/{company_id}/{codigo_generacion}/document.json

analytics/
  - Athena analytics path  
  - Structure: analytics/company_id={uuid}/dte_type={type}/year={yyyy}/month={mm}/day={dd}/{codigo_generacion}.json

Created: $(date)
EOF

echo -e "${BLUE}Creating structure documentation...${NC}"
aws s3 cp /tmp/structure.txt "s3://${BUCKET_NAME}/STRUCTURE.txt"
rm /tmp/structure.txt

echo ""
echo -e "${GREEN}✓ Directory structure created successfully!${NC}"
echo ""
echo "Verify with:"
echo "  aws s3 ls s3://${BUCKET_NAME}/"
