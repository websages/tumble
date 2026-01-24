#!/bin/bash
BASE_URL="http://localhost:8080"
FAIL=0

echo "Starting Poster Tests against $BASE_URL..."

# Fetch page conten
CONTENT=$(curl -s "$BASE_URL/?poster=kevin")

# Check we got conten
if [[ -z "$CONTENT" ]]; then
    echo "FAIL: No content returned"
    exit 1
fi

# Check for Item 79480 (The broken link item)
if [[ "$CONTENT" != *"data-irc-link-id=\"79480\""* ]]; then
     echo "FAIL: Item 79480 not found in output"
     FAIL=1
fi

# Check that 'twitter-tweet' class (Server Side Embed) IS present (We restored it)
if [[ "$CONTENT" == *"class=\"twitter-tweet\""* ]]; then
    echo "OK: Found hardcoded 'twitter-tweet' class."
else
    echo "FAIL: 'twitter-tweet' class NOT found (Server side embed missing)."
    FAIL=1
fi

# Check that 'og-preview' IS present (SuppressOG=false)
# The template renders: <div class="og-preview" id="og-preview-{{.ID}}"></div>
if [[ "$CONTENT" == *"id=\"og-preview-79480\""* ]]; then
    echo "OK: og-preview-79480 found."
else
    echo "FAIL: og-preview-79480 NOT found (SuppressOG might still be true)."
    FAIL=1
fi

if [ $FAIL -eq 0 ]; then
    echo "Poster tests passed!"
    exit 0
else
    echo "Poster tests failed!"
    exit 1
fi
