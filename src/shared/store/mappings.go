package store

import (
	"fmt"

	"paperinator/src/shared/models"
)

// ListMappings returns the field mappings configured for a feed.
func (s *Store) ListMappings(feedID int64) ([]models.FieldMapping, error) {
	rows, err := s.db.Query(
		`SELECT id, feed_id, source_field, target_field FROM field_mappings WHERE feed_id = ? ORDER BY id`,
		feedID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ms []models.FieldMapping
	for rows.Next() {
		var m models.FieldMapping
		if err := rows.Scan(&m.ID, &m.FeedID, &m.SourceField, &m.TargetField); err != nil {
			return nil, err
		}
		ms = append(ms, m)
	}
	return ms, rows.Err()
}

// ReplaceMappings atomically replaces all mappings for a feed with the supplied
// set. This is the natural operation for the "edit feed mappings" UI: the client
// sends the full desired state.
func (s *Store) ReplaceMappings(feedID int64, mappings []models.FieldMapping) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM field_mappings WHERE feed_id = ?`, feedID); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("clear mappings: %w", err)
	}
	for _, m := range mappings {
		if _, err := tx.Exec(
			`INSERT INTO field_mappings (feed_id, source_field, target_field) VALUES (?, ?, ?)`,
			feedID, m.SourceField, m.TargetField,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("insert mapping: %w", err)
		}
	}
	return tx.Commit()
}
