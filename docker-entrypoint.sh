#!/bin/sh
# Container entrypoint: run tumble under Litestream when object storage is
# configured, otherwise run tumble directly.
set -e

DB="${TUMBLE_DATABASE:-/data/tumble.sqlite}"
# Export so litestream.yml's ${TUMBLE_DATABASE} resolves to the same path.
export TUMBLE_DATABASE="$DB"

if [ -n "${BUCKET_NAME:-}${LITESTREAM_BUCKET:-}" ]; then
    # Restore from the backup only if the database is missing on the volume
    # (e.g. a fresh machine after volume loss). A no-op when a backup or DB
    # already exists.
    litestream restore -if-db-not-exists -if-replica-exists "$DB"

    # Run tumble as a subprocess of Litestream so replication continues for
    # the life of the process and flushes on shutdown.
    exec litestream replicate -exec "tumble"
else
    echo "litestream: no bucket configured (BUCKET_NAME unset), starting without replication" >&2
    exec tumble
fi
