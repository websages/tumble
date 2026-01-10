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

echo "Fixtures loaded."
