#!/bin/bash
# Load fixtures into the database
# Usage: ./tests/load_fixtures.sh

BASE_URL="${API_BASE_URL:-http://localhost:8080}"
DB_PATH="${DB_PATH:-tumble.sqlite}"
ADD_LINK_SCRIPT="./tests/add_link.sh"

# Detect Driver from Config if not se
if [ -z "$DRIVER" ]; then
    # Try test config first, then fall back to main config
    CONFIG_FILE=""
    if [ -f "conf/config-test.yaml" ]; then
        CONFIG_FILE="conf/config-test.yaml"
    elif [ -f "conf/config.yaml" ]; then
        CONFIG_FILE="conf/config.yaml"
    fi

    if [ -n "$CONFIG_FILE" ]; then
        DETECTED_DRIVER=$(grep "driver:" "$CONFIG_FILE" | awk '{print $2}')
        if [ -n "$DETECTED_DRIVER" ]; then
            DRIVER=$DETECTED_DRIVER
            echo "Auto-detected driver: $DRIVER (from $CONFIG_FILE)"
        fi
    fi
fi

echo "Using BASE_URL: $BASE_URL"
echo "Using DB_PATH: $DB_PATH"

echo "Loading fixtures..."

# YouTube Video
$ADD_LINK_SCRIPT "video_fan" "https://youtu.be/rgDcbP4Hem4?si=YdwaAMNTiD9PXKzH"

# Image
$ADD_LINK_SCRIPT "pic_poster" "https://cdn.tinnies.club/accounts/avatars/109/626/500/076/902/223/original/2a4a0d1a4ce728c4.jpg"

# Mastodon Pos
$ADD_LINK_SCRIPT "social_butterfly" "https://fosstodon.org/@genebean/113945244453254504"

# Standard Link
$ADD_LINK_SCRIPT "web_surfer" "http://costs.wtf"

# Reddit Post (Valheim)
$ADD_LINK_SCRIPT "gamer_girl" "https://www.reddit.com/r/valheim/comments/leqdj6/our_first_encounter_with_the_troll/"

# Imgur (Animated)
$ADD_LINK_SCRIPT "meme_lord" "https://imgur.com/only-one-jack-black-0qetp3u"

# Twitter Pos
$ADD_LINK_SCRIPT "tweet_master" "https://x.com/jcockhren/status/1229101594505097216?s=20"

# Wikipedia (Go)
$ADD_LINK_SCRIPT "knowledge_seeker" "https://en.wikipedia.org/wiki/Go_(programming_language)"

# Broken Link (404)
$ADD_LINK_SCRIPT "404_finder" "http://google.com/this-page-does-not-exist-12345"

# Another Explicit 404 Link (Test Case)
$ADD_LINK_SCRIPT "broken_link_tester" "http://httpstat.us/404"


# Unavailable Video (Soft 404)
$ADD_LINK_SCRIPT "video_gone" "https://youtu.be/Ie_Wl9eNffE"

# Valid Video (Rick Roll)
$ADD_LINK_SCRIPT "astley_fan" "https://www.youtube.com/watch?v=dQw4w9WgXcQ"

ADD_QUOTE_SCRIPT="./tests/add_quote.sh"

# Quotes
$ADD_QUOTE_SCRIPT "Linus Torvalds" "Talk is cheap. Show me the code."
$ADD_QUOTE_SCRIPT "Brian Kernighan" "Debugging is twice as hard as writing the code in the first place."
$ADD_QUOTE_SCRIPT "Simba" "Everything the light touches is our kingdom."

# Spotify
$ADD_LINK_SCRIPT "music_lover" "https://open.spotify.com/episode/7makk4oTQel546B0PZlDM5"

# TikTok
$ADD_LINK_SCRIPT "tiktok_star" "https://www.tiktok.com/@tiagogreis/video/6830059644233223429"

# Flickr
$ADD_LINK_SCRIPT "photog" "http://flickr.com/photos/bees/2362225867/"

# Instagram
$ADD_LINK_SCRIPT "insta_fan" "https://www.instagram.com/p/fA9uwTtkSN/"

# Dailymotion
$ADD_LINK_SCRIPT "video_daily" "https://www.dailymotion.com/video/x7tgad0"

# Kickstarter
$ADD_LINK_SCRIPT "backer" "https://www.kickstarter.com/projects/ouya/ouya-a-new-kind-of-video-game-console"

# SlideShare
$ADD_LINK_SCRIPT "presenter" "http://www.slideshare.net/lyndadotcom/code-drivesworld12"

# Speaker Deck
$ADD_LINK_SCRIPT "speaker" "https://speakerdeck.com/mislav/git"

# Giphy
$ADD_LINK_SCRIPT "gif_master" "https://giphy.com/gifs/cant-hardly-wait-kW8mnYSNkUYKc"

# Kevin's Broken Twitter Link (Sad Path)
# Use the specific broken ID if possible, but add_link auto-increments.
# We will just assert on content behavior for "kevin".
$ADD_LINK_SCRIPT "kevin" "https://twitter.com/darkuncle/status/1483507577174441985"

# Tester's Valid Twitter Link (Happy Path)
$ADD_LINK_SCRIPT "tester" "https://twitter.com/jcockhren/status/1229101594505097216?s=20"

echo "Loading backdated 'Hot Links' directly into DB..."
if [ "$DRIVER" == "mysql" ]; then
    MYSQL_HOST="${MYSQL_HOST:-localhost}"
    MYSQL_USER="${MYSQL_USER:-tumble}"
    # Defaulting to no password for local dev if not se

    # Try to detect port from config if not se
    if [ -z "$MYSQL_PORT" ] && [ -n "$CONFIG_FILE" ]; then
        # simple grep for host: ip:por
        HOST_LINE=$(grep "host:" "$CONFIG_FILE" | awk '{print $2}')
        if [[ "$HOST_LINE" == *":"* ]]; then
            MYSQL_PORT=${HOST_LINE#*:}
            echo "Auto-detected MySQL Port: $MYSQL_PORT"
        fi
    fi
    MYSQL_PORT="${MYSQL_PORT:-3306}"

    CMD="mysql -h $MYSQL_HOST -P $MYSQL_PORT -u $MYSQL_USER"
    if [ -n "$MYSQL_PASSWORD" ]; then
        CMD="$CMD -p$MYSQL_PASSWORD"
    fi
    # Use database from config or env?
    DB_NAME="${MYSQL_DATABASE:-tumble}"
    # Check config for database name too if possible
    if [ -z "$MYSQL_DATABASE" ] && [ -n "$CONFIG_FILE" ]; then
         DETECTED_DB=$(grep "database:" "$CONFIG_FILE" | awk '{print $2}')
         if [ -n "$DETECTED_DB" ]; then
            DB_NAME=$DETECTED_DB
         fi
    fi

    # Suppress password warning
    $CMD "$DB_NAME" < tests/fixtures_hot_mysql.sql 2>/dev/null
else
    sqlite3 "$DB_PATH" < tests/fixtures_hot.sql
fi

echo "Fixtures loaded."
