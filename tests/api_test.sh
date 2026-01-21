#!/bin/bash
# API Integration Tests for Tumble Go rewrite

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

# 3. Search (HTML)
check_200 "/search.cgi?search=test"

# 4. IRCLink Redirect (Setup needed for real test, checking 404/400 for bad ID)
echo -n "Checking /irclink/?id=999999 (Expect 404/Redirect)... "
status=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/irclink/?id=999999")
if [ "$status" == "404" ] || [ "$status" == "302" ]; then
    echo "OK (Status: $status)"
else
    echo "FAIL (Status: $status)"
    FAIL=1
fi

# 5. v0 Endpoints
check_200 "/v0/"
check_200 "/v0/search.cgi?search=test"

# 6. YouTube 404 Check
echo -n "Checking YouTube value for missing video (Expect 200 OK + status: 404)... "
resp=$(curl -s "$BASE_URL/ogpreview.cgi?url=https://www.youtube.com/watch?v=video_gone")
# Check if response contains '"status": 404' (or 'status":404' depending on spacing)
if [[ "$resp" == *'"status":404'* ]] || [[ "$resp" == *'"status": 404'* ]]; then
   echo "OK"
else
   echo "FAIL (Got: $resp)"
   FAIL=1
fi

if [ $FAIL -eq 0 ]; then
    echo "All tests passed!"
    exit 0
else
    echo "Tests failed!"
    exit 1
fi
