#!/bin/bash
# Script to delete an IRC Link
# Usage: ./delete_link.sh <id> [admin_secret]
# If admin_secret is not provided, uses TUMBLE_ADMIN_SECRET env var

ID=$1
SECRET=${2:-$TUMBLE_ADMIN_SECRET}
BASE_URL="http://localhost:8080"

if [ -z "$ID" ]; then
    echo "Usage: $0 <id> [admin_secret]"
    echo "       Or set TUMBLE_ADMIN_SECRET environment variable"
    exit 1
fi

if [ -z "$SECRET" ]; then
    echo "Warning: No admin secret provided. Request may fail."
    curl -v -X DELETE "$BASE_URL/irclink/?id=$ID"
else
    curl -v -X DELETE -H "X-Admin-Secret: $SECRET" "$BASE_URL/irclink/?id=$ID"
fi
echo ""
