-- Stores the set of source field names discovered in each feed during ingestion.
-- sample_value holds the first non-empty value seen for that field, surfaced in
-- the mapping UI so users can identify fields without knowing the RSS spec.
CREATE TABLE feed_fields (
    feed_id      INTEGER NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,
    field_name   TEXT    NOT NULL,
    sample_value TEXT    NOT NULL DEFAULT '',
    PRIMARY KEY (feed_id, field_name)
);
