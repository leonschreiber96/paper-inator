-- Initial schema for paper-inator.
-- Each migration file is applied once, in filename order, inside a transaction.

CREATE TABLE feeds (
    id                 INTEGER PRIMARY KEY AUTOINCREMENT,
    name               TEXT    NOT NULL,
    url                TEXT    NOT NULL UNIQUE,
    enabled            INTEGER NOT NULL DEFAULT 1,
    fetch_interval_sec INTEGER NOT NULL DEFAULT 0, -- 0 means "use the global default"
    last_fetched_at    TIMESTAMP,
    created_at         TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Per-feed mapping of a source field onto an internal publication field.
CREATE TABLE field_mappings (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    feed_id      INTEGER NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,
    source_field TEXT    NOT NULL,
    target_field TEXT    NOT NULL,
    UNIQUE(feed_id, target_field)
);

CREATE TABLE publications (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    feed_id      INTEGER NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,
    title        TEXT    NOT NULL,
    authors      TEXT    NOT NULL DEFAULT '',
    abstract     TEXT    NOT NULL DEFAULT '',
    link         TEXT    NOT NULL DEFAULT '',
    published_at TIMESTAMP,
    fetched_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    dedup_key    TEXT    NOT NULL UNIQUE, -- deterministic hash of normalized title + authors
    raw          TEXT    NOT NULL DEFAULT ''
);

CREATE INDEX idx_publications_feed       ON publications(feed_id);
CREATE INDEX idx_publications_published  ON publications(published_at);

-- User-configured email digests. Logic is implemented when the feature lands;
-- the table exists now so the schema is stable.
CREATE TABLE summaries (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT    NOT NULL,
    recipient  TEXT    NOT NULL,
    feed_ids   TEXT    NOT NULL DEFAULT '[]', -- JSON array of feed ids
    max_items  INTEGER NOT NULL DEFAULT 10,
    schedule   TEXT    NOT NULL DEFAULT '',
    enabled    INTEGER NOT NULL DEFAULT 1
);

-- Generic key/value store for frontend and global settings.
CREATE TABLE settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT ''
);
