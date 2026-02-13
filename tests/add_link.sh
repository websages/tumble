#!/bin/bash
# Script to add a Link via v1 API
# Usage: ./add_link.sh <user> <url>

USER=$1
URL=$2
BASE_URL="http://localhost:8080"

if [ -z "$USER" ] || [ -z "$URL" ]; then
    echo "Usage: $0 <user> <url>"
    exit 1
fi

curl -s -X POST "$BASE_URL/api/v1/links" \
    -H "Content-Type: application/json" \
    -d "{\"user\":\"$USER\",\"url\":\"$URL\"}"
