#!/bin/bash
# Script to add an IRC Link
# Usage: ./add_link.sh <user> <url>

USER=$1
URL=$2
BASE_URL="http://localhost:8080"

if [ -z "$USER" ] || [ -z "$URL" ]; then
    echo "Usage: $0 <user> <url>"
    exit 1
fi


curl -s "$BASE_URL/irclink/?user=$USER&url=$URL&source=irc" >/dev/null

