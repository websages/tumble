#!/bin/bash
# Script to add a Quote
# Usage: ./add_quote.sh <author> <quote>

AUTHOR=$1
QUOTE=$2
BASE_URL="http://localhost:8080"

if [ -z "$AUTHOR" ] || [ -z "$QUOTE" ]; then
    echo "Usage: $0 <author> <quote>"
    exit 1
fi


# Encode Content for safe passing? specific for shell escaping?
# Assuming simple strings for now or rely on curl --data-urlencode

curl -s --data-urlencode "quote=$QUOTE" --data-urlencode "author=$AUTHOR" "$BASE_URL/quote/" >/dev/null

