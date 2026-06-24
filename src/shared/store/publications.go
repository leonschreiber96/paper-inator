package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"paperinator/src/shared/models"
)

// PublicationFilter describes optional constraints and paging for ListPublications.
type PublicationFilter struct {
	FeedID int64  // 0 means "any feed"
	Search string // case-insensitive substring match on title/authors; "" means no filter
	Limit  int    // 0 means default (50)
	Offset int
	SortBy string // "published_at" (default) or "fetched_at" or "title"
	Desc   bool   // sort descending (default true for time fields)
}

// InsertPublication stores a publication if its DedupKey is not already present.
// It returns (inserted, error). A return of (false, nil) means the publication
// was a duplicate and was deliberately skipped — this is the explainable dedup
// behavior required by the project rules.
func (s *Store) InsertPublication(p *models.Publication) (bool, error) {
	if p.DedupKey == "" {
		return false, fmt.Errorf("publication has empty dedup_key")
	}
	res, err := s.db.Exec(
		`INSERT OR IGNORE INTO publications
		   (feed_id, title, authors, abstract, link, published_at, dedup_key, raw)
		   VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		p.FeedID, p.Title, p.Authors, p.Abstract, p.Link, nullTime(p.PublishedAt), p.DedupKey, p.Raw,
	)
	if err != nil {
		return false, fmt.Errorf("insert publication: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	if n == 0 {
		return false, nil // duplicate dedup_key — skipped
	}
	id, err := res.LastInsertId()
	if err != nil {
		return true, err
	}
	p.ID = id
	return true, nil
}

// ListPublications returns publications matching the filter, newest first by default.
func (s *Store) ListPublications(f PublicationFilter) ([]models.Publication, error) {
	var where []string
	var args []any
	if f.FeedID > 0 {
		where = append(where, "feed_id = ?")
		args = append(args, f.FeedID)
	}
	if s := strings.TrimSpace(f.Search); s != "" {
		where = append(where, "(LOWER(title) LIKE ? OR LOWER(authors) LIKE ?)")
		like := "%" + strings.ToLower(s) + "%"
		args = append(args, like, like)
	}

	query := `SELECT id, feed_id, title, authors, abstract, link, published_at, fetched_at, dedup_key, raw
	            FROM publications`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}

	query += " ORDER BY " + orderClause(f)

	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}
	query += " LIMIT ? OFFSET ?"
	args = append(args, limit, f.Offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pubs []models.Publication
	for rows.Next() {
		p, err := scanPublication(rows)
		if err != nil {
			return nil, err
		}
		pubs = append(pubs, *p)
	}
	return pubs, rows.Err()
}

// orderClause builds a safe ORDER BY from a small allowlist of columns, never
// from raw user input, to avoid SQL injection.
func orderClause(f PublicationFilter) string {
	col := "published_at"
	switch f.SortBy {
	case "fetched_at":
		col = "fetched_at"
	case "title":
		col = "title"
	}
	dir := "DESC"
	if !f.Desc {
		dir = "ASC"
	}
	// Tie-break on id for stable ordering.
	return col + " " + dir + ", id " + dir
}

func scanPublication(sc scanner) (*models.Publication, error) {
	var p models.Publication
	var published sql.NullString
	var fetched string
	err := sc.Scan(&p.ID, &p.FeedID, &p.Title, &p.Authors, &p.Abstract, &p.Link,
		&published, &fetched, &p.DedupKey, &p.Raw)
	if err != nil {
		return nil, err
	}
	if p.PublishedAt, err = scanTime(published); err != nil {
		return nil, err
	}
	if p.FetchedAt, err = scanRequiredTime(fetched); err != nil {
		return nil, err
	}
	return &p, nil
}

// nullTime converts an optional time into a value suitable for SQLite, storing
// NULL when no time is set and otherwise the canonical text format used across
// the schema (see dbtime.go).
func nullTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return formatDBTime(*t)
}
