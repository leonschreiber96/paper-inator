package store

import (
	"encoding/json"
	"fmt"

	"paperinator/src/shared/models"
)

// CreateSummary inserts a new email-summary configuration.
func (s *Store) CreateSummary(sm *models.Summary) error {
	feedIDs, err := json.Marshal(sm.FeedIDs)
	if err != nil {
		return err
	}
	res, err := s.db.Exec(
		`INSERT INTO summaries (name, recipient, feed_ids, max_items, schedule, enabled)
		   VALUES (?, ?, ?, ?, ?, ?)`,
		sm.Name, sm.Recipient, string(feedIDs), sm.MaxItems, sm.Schedule, sm.Enabled,
	)
	if err != nil {
		return fmt.Errorf("insert summary: %w", err)
	}
	sm.ID, err = res.LastInsertId()
	return err
}

// ListSummaries returns all configured summaries.
func (s *Store) ListSummaries() ([]models.Summary, error) {
	rows, err := s.db.Query(
		`SELECT id, name, recipient, feed_ids, max_items, schedule, enabled FROM summaries ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []models.Summary
	for rows.Next() {
		var sm models.Summary
		var feedIDs string
		if err := rows.Scan(&sm.ID, &sm.Name, &sm.Recipient, &feedIDs, &sm.MaxItems, &sm.Schedule, &sm.Enabled); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(feedIDs), &sm.FeedIDs); err != nil {
			return nil, fmt.Errorf("decode feed_ids for summary %d: %w", sm.ID, err)
		}
		summaries = append(summaries, sm)
	}
	return summaries, rows.Err()
}

// UpdateSummary updates an existing summary configuration.
func (s *Store) UpdateSummary(sm *models.Summary) error {
	feedIDs, err := json.Marshal(sm.FeedIDs)
	if err != nil {
		return err
	}
	res, err := s.db.Exec(
		`UPDATE summaries SET name = ?, recipient = ?, feed_ids = ?, max_items = ?, schedule = ?, enabled = ?
		   WHERE id = ?`,
		sm.Name, sm.Recipient, string(feedIDs), sm.MaxItems, sm.Schedule, sm.Enabled, sm.ID,
	)
	if err != nil {
		return fmt.Errorf("update summary: %w", err)
	}
	return checkAffected(res)
}

// DeleteSummary removes a summary configuration.
func (s *Store) DeleteSummary(id int64) error {
	res, err := s.db.Exec(`DELETE FROM summaries WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete summary: %w", err)
	}
	return checkAffected(res)
}
