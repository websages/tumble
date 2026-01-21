-- SQLite Schema for Tumble
-- Translated from MySQL schema in migrations file

-- Migration 001: Initial schema
CREATE TABLE IF NOT EXISTS image (
  imageID INTEGER PRIMARY KEY AUTOINCREMENT,
  timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  title TEXT NOT NULL DEFAULT '',
  link TEXT NOT NULL DEFAULT '',
  url TEXT NOT NULL,
  md5sum TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_image_id ON image(imageID);

CREATE TABLE IF NOT EXISTS ircLink (
  ircLinkID INTEGER PRIMARY KEY AUTOINCREMENT,
  timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  user TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL DEFAULT '',
  url TEXT NOT NULL,
  clicks INTEGER NOT NULL DEFAULT 0,
  content_type TEXT
);

CREATE INDEX IF NOT EXISTS idx_irclink_id ON ircLink(ircLinkID);

-- Note: SQLite doesn't have native FULLTEXT like MySQL's MyISAM
-- The tumble::DB::SQLite driver uses LIKE-based search instead
-- For better performance, consider using FTS5 virtual tables in the future

CREATE TABLE IF NOT EXISTS quote (
  quoteID INTEGER PRIMARY KEY AUTOINCREMENT,
  timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  quote TEXT NOT NULL DEFAULT '',
  author TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_quote_id ON quote(quoteID);

-- Schema version tracking table
CREATE TABLE IF NOT EXISTS schema_version (
  version INTEGER PRIMARY KEY,
  applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  description TEXT
);
