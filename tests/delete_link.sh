#!/bin/bash
# Script to delete an IRC Link
# Usage: ./delete_link.sh <id>

ID=$1
BASE_URL="http://localhost:8080"

if [ -z "$ID" ]; then
    echo "Usage: $0 <id>"
    exit 1
fi

curl -v -X DELETE "$BASE_URL/irclink/?id=$ID"
echo ""
