CREATE TABLE IF NOT EXISTS activitypub_key (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  private_key TEXT NOT NULL DEFAULT '',
  public_key TEXT NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS activitypub_follower (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  actor_uri TEXT NOT NULL DEFAULT '',
  inbox_url TEXT NOT NULL DEFAULT '',
  shared_inbox TEXT,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_activitypub_follower_actor_uri ON activitypub_follower(actor_uri);

CREATE TABLE IF NOT EXISTS activitypub_delivery (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  inbox_url TEXT NOT NULL DEFAULT '',
  payload TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'pending',
  attempts INTEGER NOT NULL DEFAULT 0,
  next_attempt DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  last_error TEXT,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_activitypub_delivery_inbox_url ON activitypub_delivery(inbox_url);
CREATE INDEX IF NOT EXISTS idx_activitypub_delivery_status ON activitypub_delivery(status);
CREATE INDEX IF NOT EXISTS idx_activitypub_delivery_next_attempt ON activitypub_delivery(next_attempt);
