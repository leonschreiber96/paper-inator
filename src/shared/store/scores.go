package store

import (
	"time"

	"paperinator/src/shared/models"
)

// UpsertPublicationScore records or updates the relevance score for a single
// publication. Calling this a second time overwrites the previous score, which
// is the intended behavior when the user triggers a re-analysis.
func (s *Store) UpsertPublicationScore(pubID int64, score float64, notes, scorerType string) error {
	_, err := s.db.Exec(`
		INSERT INTO publication_scores (publication_id, score, notes, scorer_type, scored_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(publication_id) DO UPDATE
		  SET score = excluded.score,
		      notes = excluded.notes,
		      scorer_type = excluded.scorer_type,
		      scored_at = excluded.scored_at`,
		pubID, score, notes, scorerType, formatDBTime(time.Now().UTC()),
	)
	return err
}

// ListUnscoredPublications returns up to limit publications that have no entry
// in publication_scores. Results are ordered by fetched_at descending so the
// most recently ingested papers are enriched first.
func (s *Store) ListUnscoredPublications(limit int) ([]models.Publication, error) {
	rows, err := s.db.Query(`
		SELECT p.id, p.feed_id, p.title, p.authors, p.abstract, p.link,
		       p.published_at, p.fetched_at, p.dedup_key, p.raw
		FROM publications p
		WHERE NOT EXISTS (SELECT 1 FROM publication_scores ps WHERE ps.publication_id = p.id)
		ORDER BY p.fetched_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pubs []models.Publication
	for rows.Next() {
		p, err := scanPublicationBase(rows)
		if err != nil {
			return nil, err
		}
		pubs = append(pubs, *p)
	}
	return pubs, rows.Err()
}

// DeleteAllScores removes every row from publication_scores, effectively
// marking all publications as unscored so the enrichment worker re-processes
// them on the next tick.
func (s *Store) DeleteAllScores() error {
	_, err := s.db.Exec(`DELETE FROM publication_scores`)
	return err
}
