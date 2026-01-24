#!/bin/bash
BASE_URL="http://localhost:8080"
FAIL=0

echo "Starting Twitter Behavior Tests against $BASE_URL..."

# Load Fixtures
echo "Loading fixtures..."
./tests/load_fixtures.sh > /dev/null

# 1. HAPPY PATH: Valid Twee
# We need a user/post that has a VALID twitter link.
# Based on fixtures/logs, we might look for a known valid one or insert one if we had an insert script.
# For now, let's assume one exists or we check the 'tester' user mentioned by the user.
# "See localhost:8080 for posts from 'tester'"

echo -n "Checking for Valid Tweet Render (tester)... "
CONTENT_VALID=$(curl -s "$BASE_URL/?poster=tester")

# Check for server-side widget code
if [[ "$CONTENT_VALID" == *"class=\"twitter-tweet\""* ]]; then
    echo "OK (Server-side widget found)"
else
    # It might be that 'tester' has no tweets, but user said "posts from tester... tweets no longer rendering".
    # So we assume they exist.
    echo "FAIL (Server-side widget MISSING for valid tweet)"
    FAIL=1
fi

# Check for og-preview hook (required for 404 check even on valid ones, though it does nothing if valid)
if [[ "$CONTENT_VALID" == *"class=\"og-preview\""* ]]; then
    echo "OK (og-preview hook found)"
else
     echo "FAIL (og-preview hook MISSING)"
     FAIL=1
fi


# 2. SAD PATH: Broken Tweet (404)
# We know 'kevin' has the broken one (ID 79480).
echo -n "Checking for Broken Tweet Handling (kevin)... "
CONTENT_BROKEN=$(curl -s "$BASE_URL/?poster=kevin")

# It SHOULD have the widget code initially (server-side doesn't know it's broken yet)
if [[ "$CONTENT_BROKEN" == *"class=\"twitter-tweet\""* ]]; then
    echo "OK (Server-side widget present initially)"
else
    echo "FAIL (Widget missing on broken link - strictly speaking it should be there, then removed by JS?)"
    # Actually, in our logic, we ALWAYS render the widget if it looks like a tweet.
    # The JS then runs ogpreview, gets 404, and REPLACES the content.
    # So purely from curl (server response), the widget MUST be there.
    FAIL=1
fi

# It MUST have the og-preview hook
if [[ "$CONTENT_BROKEN" == *"class=\"og-preview\""* ]]; then
     echo "OK (og-preview hook present)"
else
     echo "FAIL (og-preview hook MISSING on broken link)"
     FAIL=1
fi

# 3. VERIFY CLIENT SIDE 404 (Simulated)
# We can't easily run JS here, but we can verify the *endpoint* acting correctly.
# Request ogpreview for the broken URL.
BROKEN_URL="https://twitter.com/darkuncle/status/1483507577174441985"
echo -n "Checking ogpreview response for broken URL... "
PREVIEW_RESP=$(curl -s "$BASE_URL/ogpreview.cgi?url=$BROKEN_URL")

if [[ "$PREVIEW_RESP" == *"\"status\":404"* ]] || [[ "$PREVIEW_RESP" == *"\"status\": 404"* ]]; then
    echo "OK (API returns 404 status)"
else
    echo "FAIL (API did not return 404 status: $PREVIEW_RESP)"
    FAIL=1
fi

if [ $FAIL -eq 0 ]; then
    echo "All Twitter behavior tests passed!"
    exit 0
else
    echo "Tests failed!"
    exit 1
fi
