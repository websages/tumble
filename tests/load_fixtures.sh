#!/bin/bash
# Load fixtures into the database
# Usage: ./tests/load_fixtures.sh

BASE_URL="http://localhost:8080"
ADD_LINK_SCRIPT="./tests/add_link.sh"

echo "Loading fixtures..."

# YouTube Video
$ADD_LINK_SCRIPT "video_fan" "https://youtu.be/rgDcbP4Hem4?si=YdwaAMNTiD9PXKzH"

# Image
$ADD_LINK_SCRIPT "pic_poster" "https://cdn.tinnies.club/accounts/avatars/109/626/500/076/902/223/original/2a4a0d1a4ce728c4.jpg"

# Mastodon Post
$ADD_LINK_SCRIPT "social_butterfly" "https://fosstodon.org/@genebean/113945244453254504"

# Standard Link
$ADD_LINK_SCRIPT "web_surfer" "http://costs.wtf"

# Twitter Post
$ADD_LINK_SCRIPT "tweet_master" "https://x.com/jcockhren/status/1229101594505097216?s=20"

ADD_QUOTE_SCRIPT="./tests/add_quote.sh"

# Quotes
$ADD_QUOTE_SCRIPT "Linus Torvalds" "Talk is cheap. Show me the code."
$ADD_QUOTE_SCRIPT "Brian Kernighan" "Debugging is twice as hard as writing the code in the first place."
$ADD_QUOTE_SCRIPT "Simba" "Everything the light touches is our kingdom."

echo "Loading backdated 'Hot Links' directly into DB..."
sqlite3 tumble.sqlite < tests/fixtures_hot.sql

echo "Fixtures loaded."
