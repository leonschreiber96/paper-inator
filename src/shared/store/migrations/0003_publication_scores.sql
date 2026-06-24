-- Stores the relevance score assigned to each publication by the enrichment
-- worker. One row per publication; absence means the publication has not been
-- scored yet (e.g. no interest profile is configured, or the worker hasn't
-- reached it). Deleted automatically when the parent publication is deleted.
CREATE TABLE publication_scores (
    publication_id  INTEGER NOT NULL PRIMARY KEY REFERENCES publications(id) ON DELETE CASCADE,
    score           REAL    NOT NULL,           -- 0.0 to 1.0
    notes           TEXT    NOT NULL DEFAULT '', -- human-readable explanation of the score
    scorer_type     TEXT    NOT NULL,            -- 'keyword' or 'llm'
    scored_at       TEXT    NOT NULL             -- UTC timestamp, same format as other time columns
);
