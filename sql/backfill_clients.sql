-- One-time backfill: set client metadata on all existing rows.
-- All existing data originates from IRC, #soggies channel on jameswhite.org.
-- Run this manually after deploying the client fields migration.
--
-- Usage (SQLite):  sqlite3 tumble.db < sql/backfill_clients.sql
-- Usage (MySQL):   mysql -u user -p tumble < sql/backfill_clients.sql

UPDATE ircLink
SET client_type = 'irc',
    client_network = 'jameswhite.org',
    client_channel = '#soggies'
WHERE client_type IS NULL;

UPDATE quote
SET client_type = 'irc',
    client_network = 'jameswhite.org',
    client_channel = '#soggies'
WHERE client_type IS NULL;

UPDATE image
SET client_type = 'irc',
    client_network = 'jameswhite.org',
    client_channel = '#soggies'
WHERE client_type IS NULL;
