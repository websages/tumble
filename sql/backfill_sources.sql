-- One-time backfill: set source metadata on all existing rows.
-- All existing data originates from IRC, #soggies channel on jameswhite.org.
-- Run this manually after deploying the source fields migration.
--
-- Usage (SQLite):  sqlite3 tumble.db < sql/backfill_sources.sql
-- Usage (MySQL):   mysql -u user -p tumble < sql/backfill_sources.sql

UPDATE ircLink
SET source_type = 'irc',
    source_network = 'jameswhite.org',
    source_channel = '#soggies'
WHERE source_type IS NULL;

UPDATE quote
SET source_type = 'irc',
    source_network = 'jameswhite.org',
    source_channel = '#soggies'
WHERE source_type IS NULL;

UPDATE image
SET source_type = 'irc',
    source_network = 'jameswhite.org',
    source_channel = '#soggies'
WHERE source_type IS NULL;
