#!/bin/bash

# Usage: ./test_list_companies.sh

API_URL="http://localhost:8080/v1/companies"

# Note: This endpoint might not exist in your current API
# If it doesn't, you'll need to add it to your handlers

curl -s -X GET "$API_URL" | jq '.'
