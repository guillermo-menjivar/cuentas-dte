#!/bin/bash
# File: test_search_activities.sh
# Usage: ./test_search_activities.sh <query> [limit]
# Example: ./test_search_activities.sh restaurante
# Example: ./test_search_activities.sh "venta de ropa" 20

if [ "$#" -lt 1 ]; then
    echo "Usage: $0 <query> [limit]"
    echo "Example: $0 restaurante"
    echo "Example: $0 'venta de ropa' 20"
    exit 1
fi

QUERY=$1
LIMIT=${2:-10}

API_URL="http://localhost:8080/v1/actividades-economicas/search"

# Build query parameters
PARAMS="?"
PARAMS+="q=$QUERY&"
PARAMS+="limit=$LIMIT&"

# Remove trailing &
PARAMS=$(echo "$PARAMS" | sed 's/&$//')

echo "Searching economic activities..."
echo "Query: $QUERY"
echo "Limit: $LIMIT"
echo ""

# URL encode the query properly
ENCODED_QUERY=$(printf %s "$QUERY" | jq -sRr @uri)

curl -s -X GET "${API_URL}?q=${ENCODED_QUERY}&limit=${LIMIT}" | jq '.'
