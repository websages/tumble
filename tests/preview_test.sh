#!/bin/bash
# Tests for /ogpreview.cgi
# Usage: ./tests/preview_test.sh [BASE_URL]

BASE_URL="${1:-http://localhost:8080}"
ENDPOINT="$BASE_URL/ogpreview"

echo "Running Preview Tests against $ENDPOINT"

FAILURES=0

test_preview() {
    local NAME="$1"
    local URL="$2"
    local JQ_FILTER="$3"

    # URL Encode
    ENCODED_URL=$(jq -nr --arg v "$URL" '$v|@uri')

    echo -n "  Test: $NAME... "

    RESPONSE=$(curl -s "$ENDPOINT?url=$ENCODED_URL")

    # Check if curl failed
    if [ $? -ne 0 ]; then
        echo "FAIL (curl error)"
        FAILURES=$((FAILURES + 1))
        return
    fi

    # Check assertion
    MATCH=$(echo "$RESPONSE" | jq -e "$JQ_FILTER" 2>/dev/null)

    if [ "$MATCH" = "true" ]; then
        echo "PASS"
    else
        echo "FAIL"
        echo "    URL: $URL"
        echo "    Filter: $JQ_FILTER"
        echo "    Response: $RESPONSE"
        FAILURES=$((FAILURES + 1))
    fi
}

# --- REDDIT ---
# --- REDDIT ---
# Valid: Should have rich metadata (provider_name or type)
test_preview "Reddit Valid" \
    "https://www.reddit.com/r/valheim/comments/leqdj6/our_first_encounter_with_the_troll/" \
    '.provider_name == "Reddit" or .title != null'

# Invalid: specific non-existent post.
test_preview "Reddit Invalid" \
    "https://www.reddit.com/r/valheim/comments/INVALID_ID_12345/" \
    '.error != null or (.title | contains("Page not found") or contains("Reddit"))'

# --- SPOTIFY ---
# Valid
test_preview "Spotify Valid" \
    "https://open.spotify.com/episode/7makk4oTQel546B0PZlDM5" \
    '.provider_name == "Spotify" or .type == "rich"'


# Invalid - Spotify returns generic fallback page for invalid IDs
test_preview "Spotify Invalid" \
    "https://open.spotify.com/track/INVALID_TRACK_ID" \
    '.provider_name == "Spotify" and (.title | contains("Web Player"))'

# --- IMGUR ---
# Valid
test_preview "Imgur Valid" \
    "https://imgur.com/only-one-jack-black-0qetp3u" \
    '.title != null'

# Invalid
test_preview "Imgur Invalid" \
    "https://imgur.com/gallery/INVALID_GALLERY_ID" \
    '.error != null or (.title | contains("Imgur"))'

# --- YOUTUBE ---
# Valid
test_preview "YouTube Valid" \
    "https://www.youtube.com/watch?v=dQw4w9WgXcQ" \
    '.provider_name == "YouTube"'

# Invalid (Video Unavailable) - Special handling in preview.go returning status 404
test_preview "YouTube Invalid" \
    "https://youtu.be/Ie_Wl9eNffE" \
    '.status == 404 and .error == "Video Unavailable"'

# --- TWITTER ---
# Valid - Returns empty object {} on success per preview_twitter.go
test_preview "Twitter Valid" \
    "https://x.com/jcockhren/status/1229101594505097216" \
    '. == {}'

# Invalid - Falls back to scrape. Twitter returns a generic page title / icon.
test_preview "Twitter Invalid" \
    "https://x.com/jcockhren/status/0000000000000000000" \
    '.error == "Tweet Unavailable" and .status == 404'

# --- TIKTOK ---
# Valid
test_preview "TikTok Valid" \
    "https://www.tiktok.com/@tiagogreis/video/6830059644233223429" \
    '.provider_name == "TikTok" or .type == "video"'


# Invalid - Falls back to scrape.
test_preview "TikTok Invalid" \
    "https://www.tiktok.com/@user/video/1234567890123456789" \
    '.title | contains("TikTok")'


if [ $FAILURES -eq 0 ]; then
    echo "All preview tests passed!"
    exit 0
else
    echo "$FAILURES preview tests failed."
    exit 1
fi
