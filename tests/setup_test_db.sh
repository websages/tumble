#!/bin/bash
set -e

# Configuration
BINARY="./bin/tumble"
CONFIG="conf/config-test.yaml"
DB_PATH="tumble-test.sqlite"
PORT="8080"
BASE_URL="http://localhost:$PORT"

echo "Setting up $DB_PATH..."
rm -f "$DB_PATH"

# Start server
echo "Starting server..."
$BINARY "$CONFIG" &
PID=$!
echo "Server PID: $PID"

# Ensure cleanup
trap "echo 'Stopping server...'; kill $PID || true" EXIT

# Wait for server to be ready
echo "Waiting for server to be ready on port $PORT..."
for i in {1..30}; do
    if curl -s "http://localhost:$PORT" >/dev/null; then
        echo "Server is up!"
        break
    fi
    sleep 1
done

# Run fixtures
echo "Running fixtures..."
export DB_PATH="$DB_PATH"
export API_BASE_URL="$BASE_URL"
./tests/load_fixtures.sh

echo "Test database created successfully."
