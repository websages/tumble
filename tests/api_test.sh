#!/bin/bash
# API Integration Tests for Tumble

BASE_URL="http://localhost:8080"
FAIL=0

echo "Starting Tests against $BASE_URL..."

# Helper to check if a page returns 200
check_200() {
    url=$1
    echo -n "Checking $url... "
    status=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL$url")
    if [ "$status" == "200" ]; then
        echo "OK"
    else
        echo "FAIL (Status: $status)"
        FAIL=1
    fi
}

# Helper to check content type
check_content_type() {
    url=$1
    expected=$2
    echo -n "Checking Content-Type for $url (expecting $expected)... "
    ct=$(curl -s -I "$BASE_URL$url" | grep -i "Content-Type" | awk '{print $2}' | tr -d '\r')
    if [[ "$ct" == *"$expected"* ]]; then
        echo "OK"
    else
        echo "FAIL (Got: $ct)"
        FAIL=1
    fi
}

# 1. Main Index (HTML)
check_200 "/"
check_content_type "/" "text/html"

# 2. Main Index (RSS/XML)
check_200 "/index.xml?dtype=rss"
check_content_type "/index.xml?dtype=rss" "text/xml"

# Optional: XML Validation if xmllint is presen
if command -v xmllint &> /dev/null; then
    echo -n "Validating RSS XML structure... "
    curl -s "$BASE_URL/index.xml?dtype=rss" > rss_temp.xml
    if xmllint --noout rss_temp.xml 2>/dev/null; then
        echo "OK"
        rm rss_temp.xml
    else
        echo "FAIL (XML Validation errors)"
        xmllint --noout rss_temp.xml
        rm rss_temp.xml
        FAIL=1
    fi
else
    echo "Skipping XML validation (xmllint not found)"
fi

# 3. Search (HTML)
check_200 "/search?search=test"

# 4. API v1 Preview
echo -n "Checking YouTube value for missing video (Expect 200 OK + status: 404) on /api/v1/preview... "
resp=$(curl -s "$BASE_URL/api/v1/preview?url=https://www.youtube.com/watch?v=video_gone")
if [[ "$resp" == *'"status":404'* ]] || [[ "$resp" == *'"status": 404'* ]]; then
   echo "OK"
else
   echo "FAIL (Got: $resp)"
   FAIL=1
fi

# 5. API v1 Link Create -> Delete -> Verify
echo -n "Testing API v1 Link Create/Delete flow... "
# Create a link
CREATE_OUT=$(curl -s -X POST "$BASE_URL/api/v1/links" \
    -H "Content-Type: application/json" \
    -H "Accept: text/plain" \
    -d '{"user":"testdel","url":"http://delete-test.com"}')
# Parse ID from response (format: "Created link N: URL")
if [[ "$CREATE_OUT" =~ Created\ link\ ([0-9]+) ]]; then
    DEL_ID=${BASH_REMATCH[1]}
    # Delete it
    DEL_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE -H "X-API-Key: test-admin-secret" "$BASE_URL/api/v1/links/$DEL_ID")
    if [ "$DEL_STATUS" == "204" ]; then
        # Verify it's gone
        GONE_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/v1/links/$DEL_ID")
        if [ "$GONE_STATUS" == "404" ]; then
            echo "OK"
        else
            echo "FAIL (Expected 404 after delete, got $GONE_STATUS)"
            FAIL=1
        fi
    else
        echo "FAIL (Delete request failed with $DEL_STATUS)"
        FAIL=1
    fi
else
    echo "FAIL (Could not create test link: $CREATE_OUT)"
    FAIL=1
fi

if [ $FAIL -eq 0 ]; then
    echo "All tests passed!"
    exit 0
else
    echo "Tests failed!"
    exit 1
fi
