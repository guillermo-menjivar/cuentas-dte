#!/bin/bash
# File: test_get_categories.sh
# Usage: ./test_get_categories.sh
# Description: Lists all top-level economic activity categories

API_URL="http://localhost:8080/v1/actividades-economicas/categories"

echo "Fetching all economic activity categories..."
echo ""

curl -s -X GET "$API_URL"
