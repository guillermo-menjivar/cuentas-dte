#!/bin/bash

################################################################################
# Bulk Client Upload Test Script
# 
# This script tests the bulk client upload endpoint by uploading a CSV file
# and displaying the results in a formatted way.
#
# Usage:
#   ./test_bulk_upload.sh [CSV_FILE] [API_URL] [TOKEN]
#
# Examples:
#   ./test_bulk_upload.sh sample_25_clients.csv http://localhost:8080/v1/clients/bulk-upload YOUR_TOKEN
#   ./test_bulk_upload.sh clients.csv https://api.example.com/v1/clients/bulk-upload YOUR_TOKEN
################################################################################

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Default values
CSV_FILE="${1:-sample_25_clients.csv}"
API_URL="${2:-http://localhost:8080/v1/clients/bulk-upload}"
TOKEN="${3:-YOUR_TOKEN_HERE}"

# Check if CSV file exists
if [ ! -f "$CSV_FILE" ]; then
    echo -e "${RED}Error: CSV file '$CSV_FILE' not found${NC}"
    echo "Usage: $0 [CSV_FILE] [API_URL] [TOKEN]"
    exit 1
fi

# Check if token is provided
if [ "$TOKEN" == "YOUR_TOKEN_HERE" ]; then
    echo -e "${YELLOW}Warning: Using default token. Please provide your actual token as the third argument.${NC}"
    echo "Usage: $0 [CSV_FILE] [API_URL] [TOKEN]"
    echo ""
fi

# Display test information
echo -e "${CYAN}========================================${NC}"
echo -e "${CYAN}  Bulk Client Upload Test${NC}"
echo -e "${CYAN}========================================${NC}"
echo -e "${BLUE}CSV File:${NC} $CSV_FILE"
echo -e "${BLUE}API URL:${NC}  $API_URL"
echo -e "${BLUE}Token:${NC}    ${TOKEN:0:20}..."
echo ""

# Count lines in CSV (excluding header)
LINE_COUNT=$(($(wc -l < "$CSV_FILE") - 1))
echo -e "${BLUE}Total records in CSV:${NC} $LINE_COUNT"
echo ""

# Create a temporary file for the response
RESPONSE_FILE=$(mktemp)
HTTP_CODE_FILE=$(mktemp)

# Make the API call
echo -e "${YELLOW}Uploading CSV file...${NC}"
echo ""

HTTP_CODE=$(curl -s -w "%{http_code}" -o "$RESPONSE_FILE" \
    -X POST "$API_URL" \
    -H "Authorization: Bearer $TOKEN" \
    -F "file=@$CSV_FILE")

echo "$HTTP_CODE" > "$HTTP_CODE_FILE"

# Display HTTP status code
echo -e "${CYAN}========================================${NC}"
echo -e "${CYAN}  Response${NC}"
echo -e "${CYAN}========================================${NC}"
echo -e "${BLUE}HTTP Status Code:${NC} $HTTP_CODE"
echo ""

# Check if jq is available for pretty JSON formatting
if command -v jq &> /dev/null; then
    HAS_JQ=true
else
    HAS_JQ=false
    echo -e "${YELLOW}Note: Install 'jq' for better JSON formatting${NC}"
    echo ""
fi

# Process response based on HTTP code
if [ "$HTTP_CODE" -eq 201 ]; then
    echo -e "${GREEN}✓ Upload successful!${NC}"
    echo ""
    
    if [ "$HAS_JQ" = true ]; then
        # Extract key information using jq
        TOTAL_ROWS=$(jq -r '.total_rows' "$RESPONSE_FILE")
        SUCCESS_COUNT=$(jq -r '.success_count' "$RESPONSE_FILE")
        FAILED_COUNT=$(jq -r '.failed_count' "$RESPONSE_FILE")
        
        echo -e "${CYAN}Summary:${NC}"
        echo -e "  ${BLUE}Total Rows:${NC}    $TOTAL_ROWS"
        echo -e "  ${GREEN}Success:${NC}       $SUCCESS_COUNT"
        echo -e "  ${RED}Failed:${NC}        $FAILED_COUNT"
        echo ""
        
        # Show failed clients if any
        if [ "$FAILED_COUNT" -gt 0 ]; then
            echo -e "${RED}Failed Clients:${NC}"
            jq -r '.failed_clients[] | "  Row \(.row): \(.business_name) - \(.error)"' "$RESPONSE_FILE"
            echo ""
        fi
        
        # Show first 5 successful clients
        if [ "$SUCCESS_COUNT" -gt 0 ]; then
            echo -e "${GREEN}Sample of Successful Clients (first 5):${NC}"
            jq -r '.success_clients[0:5] | .[] | "  ✓ \(.business_name) (\(.nit // .dui // "No ID"))"' "$RESPONSE_FILE"
            echo ""
        fi
        
        # Full response
        echo -e "${CYAN}Full Response:${NC}"
        jq '.' "$RESPONSE_FILE"
    else
        # Without jq, just show raw response
        cat "$RESPONSE_FILE"
    fi
    
elif [ "$HTTP_CODE" -eq 400 ]; then
    echo -e "${RED}✗ Bad Request - Validation Error${NC}"
    echo ""
    
    if [ "$HAS_JQ" = true ]; then
        ERROR_MSG=$(jq -r '.error' "$RESPONSE_FILE")
        ERROR_CODE=$(jq -r '.code' "$RESPONSE_FILE")
        
        echo -e "${RED}Error:${NC} $ERROR_MSG"
        echo -e "${RED}Code:${NC}  $ERROR_CODE"
        echo ""
        
        # Check if there are validation errors
        if jq -e '.errors' "$RESPONSE_FILE" > /dev/null 2>&1; then
            VALID_ROWS=$(jq -r '.valid_rows // 0' "$RESPONSE_FILE")
            FAILED_ROWS=$(jq -r '.failed_rows // 0' "$RESPONSE_FILE")
            
            echo -e "${BLUE}Valid Rows:${NC}   $VALID_ROWS"
            echo -e "${RED}Failed Rows:${NC}  $FAILED_ROWS"
            echo ""
            
            echo -e "${RED}Validation Errors:${NC}"
            jq -r '.errors[] | "  Row \(.row): \(.error)"' "$RESPONSE_FILE"
            echo ""
        fi
        
        echo -e "${CYAN}Full Response:${NC}"
        jq '.' "$RESPONSE_FILE"
    else
        cat "$RESPONSE_FILE"
    fi
    
elif [ "$HTTP_CODE" -eq 401 ]; then
    echo -e "${RED}✗ Unauthorized - Invalid or missing token${NC}"
    echo ""
    cat "$RESPONSE_FILE"
    
elif [ "$HTTP_CODE" -eq 500 ]; then
    echo -e "${RED}✗ Internal Server Error${NC}"
    echo ""
    cat "$RESPONSE_FILE"
    
else
    echo -e "${YELLOW}Unexpected HTTP Code: $HTTP_CODE${NC}"
    echo ""
    cat "$RESPONSE_FILE"
fi

echo ""
echo -e "${CYAN}========================================${NC}"

# Cleanup
rm -f "$RESPONSE_FILE" "$HTTP_CODE_FILE"

# Exit with appropriate code
if [ "$HTTP_CODE" -eq 201 ]; then
    exit 0
else
    exit 1
fi
