package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"paperinator/src/shared/models"
)

// ErrNotFound is returned when a requested row does not exist.
var ErrNotFound = errors.New("not found")

// ErrConflict is returned when a write violates a uniqueness constraint, e.g.
// creating a feed whose URL already exists.
var ErrConflict = errors.New("conflict")

// isUniqueViolation reports whether err is a SQLite UNIQUE constraint failure.
// The pure-Go driver surfaces this in the error message; matching on it lets the
// API translate it into a 409 rather than a generic 500.
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// CreateFeed inserts a new feed and populates its ID and CreatedAt.
func (s *Store) CreateFeed(f *models.Feed) error {
	res, err := s.db.Exec(
		`INSERT INTO feeds (name, url, enabled, fetch_interval_sec) VALUES (?, ?, ?, ?)`,
		f.Name, f.URL, f.Enabled, f.FetchIntervalSec,
	)
	if isUniqueViolation(err) {
		return fmt.Errorf("%w: a feed with this URL already exists", ErrConflict)
	}
	if err != nil {
		return fmt.Errorf("insert feed: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	created, err := s.GetFeed(id)
	if err != nil {
		return err
	}
	*f = *created
	return nil
}

// GetFeed returns a single feed by ID, or ErrNotFound.
func (s *Store) GetFeed(id int64) (*models.Feed, error) {
	row := s.db.QueryRow(
		`SELECT id, name, url, enabled, fetch_interval_sec, last_fetched_at, created_at
		   FROM feeds WHERE id = ?`, id)
	return scanFeed(row)
}

// ListFeeds returns all feeds ordered by creation time.
func (s *Store) ListFeeds() ([]models.Feed, error) {
	rows, err := s.db.Query(
		`SELECT id, name, url, enabled, fetch_interval_sec, last_fetched_at, created_at
		   FROM feeds ORDER BY created_at, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds []models.Feed
	for rows.Next() {
		f, err := scanFeed(rows)
		if err != nil {
			return nil, err
		}
		feeds = append(feeds, *f)
	}
	return feeds, rows.Err()
}

// ListEnabledFeeds returns feeds the worker should poll.
func (s *Store) ListEnabledFeeds() ([]models.Feed, error) {
	all, err := s.ListFeeds()
	if err != nil {
		return nil, err
	}
	enabled := all[:0]
	for _, f := range all {
		if f.Enabled {
			enabled = append(enabled, f)
		}
	}
	return enabled, nil
}

// UpdateFeed updates the mutable fields of an existing feed.
func (s *Store) UpdateFeed(f *models.Feed) error {
	res, err := s.db.Exec(
		`UPDATE feeds SET name = ?, url = ?, enabled = ?, fetch_interval_sec = ? WHERE id = ?`,
		f.Name, f.URL, f.Enabled, f.FetchIntervalSec, f.ID,
	)
	if err != nil {
		return fmt.Errorf("update feed: %w", err)
	}
	return checkAffected(res)
}

// DeleteFeed removes a feed and (via cascade) its mappings and publications.
func (s *Store) DeleteFeed(id int64) error {
	res, err := s.db.Exec(`DELETE FROM feeds WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete feed: %w", err)
	}
	return checkAffected(res)
}

// MarkFeedFetched records that a feed was just polled.
func (s *Store) MarkFeedFetched(id int64) error {
	_, err := s.db.Exec(`UPDATE feeds SET last_fetched_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	return err
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanFeed(sc scanner) (*models.Feed, error) {
	var f models.Feed
	var last sql.NullString
	var created string
	err := sc.Scan(&f.ID, &f.Name, &f.URL, &f.Enabled, &f.FetchIntervalSec, &last, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if f.LastFetchedAt, err = scanTime(last); err != nil {
		return nil, err
	}
	if f.CreatedAt, err = scanRequiredTime(created); err != nil {
		return nil, err
	}
	return &f, nil
}

func checkAffected(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
