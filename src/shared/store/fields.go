package store

import (
	"fmt"

	"paperinator/src/shared/models"
)

// UpsertFeedFields persists the set of source fields discovered during one
// ingest run. Existing rows are updated only if a non-empty sample_value is
// being recorded for the first time; subsequent ingests never overwrite a
// sample that is already set. New fields are inserted.
func (s *Store) UpsertFeedFields(feedID int64, fields map[string]string) error {
	if len(fields) == 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	for name, sample := range fields {
		if _, err := tx.Exec(`
			INSERT INTO feed_fields (feed_id, field_name, sample_value) VALUES (?, ?, ?)
			ON CONFLICT(feed_id, field_name) DO UPDATE
			  SET sample_value = CASE
			        WHEN feed_fields.sample_value = '' AND excluded.sample_value != ''
			        THEN excluded.sample_value
			        ELSE feed_fields.sample_value
			      END`,
			feedID, name, sample,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("upsert field %q: %w", name, err)
		}
	}
	return tx.Commit()
}

// ListFeedFields returns all discovered fields for a feed, ordered by name.
// Returns an empty slice (not nil) if the feed has not been ingested yet.
func (s *Store) ListFeedFields(feedID int64) ([]models.FeedField, error) {
	rows, err := s.db.Query(
		`SELECT feed_id, field_name, sample_value FROM feed_fields WHERE feed_id = ? ORDER BY field_name`,
		feedID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fields := []models.FeedField{}
	for rows.Next() {
		var f models.FeedField
		if err := rows.Scan(&f.FeedID, &f.FieldName, &f.SampleValue); err != nil {
			return nil, err
		}
		fields = append(fields, f)
	}
	return fields, rows.Err()
}
