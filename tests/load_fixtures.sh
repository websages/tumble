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

# Reddit Post (Valheim)
$ADD_LINK_SCRIPT "gamer_girl" "https://www.reddit.com/r/valheim/comments/leqdj6/our_first_encounter_with_the_troll/"

# Imgur (Animated)
$ADD_LINK_SCRIPT "meme_lord" "https://imgur.com/only-one-jack-black-0qetp3u"

# Twitter Post
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

echo "Loading backdated 'Hot Links' directly into DB..."
sqlite3 tumble.sqlite < tests/fixtures_hot.sql

echo "Fixtures loaded."
