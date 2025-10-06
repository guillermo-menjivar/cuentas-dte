#!/bin/bash
# File: test_get_activities_by_category.sh
# Usage: ./test_get_activities_by_category.sh <category_code>
# Example: ./test_get_activities_by_category.sh 46

if [ "$#" -lt 1 ]; then
    echo "Usage: $0 <category_code>"
    echo "Example: $0 46"
    exit 1
fi

CATEGORY_CODE=$1
API_URL="http://localhost:8080/v1/actividades-economicas/categories/$CATEGORY_CODE/activities"

echo "Fetching activities for category: $CATEGORY_CODE"
echo ""

curl -s -X GET "$API_URL" | jq '.'
