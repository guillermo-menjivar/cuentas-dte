#!/bin/bash
# File: test_get_activity_details.sh
# Usage: ./test_get_activity_details.sh <activity_code>
# Example: ./test_get_activity_details.sh 56101

if [ "$#" -lt 1 ]; then
    echo "Usage: $0 <activity_code>"
    echo "Example: $0 56101"
    exit 1
fi

ACTIVITY_CODE=$1
API_URL="http://localhost:8080/v1/actividades-economicas/$ACTIVITY_CODE"

echo "Fetching activity details for code: $ACTIVITY_CODE"
echo ""

curl -s -X GET "$API_URL" | jq '.'
