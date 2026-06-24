// Package store is the single persistence layer for paper-inator. It owns the
// SQLite connection, runs migrations, and exposes typed query helpers used by
// both the API and the service worker. No other package should talk to SQLite
// directly.
package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // pure-Go SQLite driver (no CGO), registered as "sqlite"
)

// Store wraps the database handle and provides query methods.
type Store struct {
	db *sql.DB
}

// Open opens (creating if needed) the SQLite database at path, enables sane
// pragmas, and applies any pending migrations.
func Open(path string) (*Store, error) {
	// _pragma options are applied per connection by the modernc driver via the DSN.
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=foreign_keys(on)&_pragma=journal_mode(WAL)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}
	return &Store{db: db}, nil
}

// Close releases the underlying database handle.
func (s *Store) Close() error { return s.db.Close() }

// DB exposes the raw handle for advanced/ad-hoc use (e.g. tests). Prefer the
// typed helpers on Store for application code.
func (s *Store) DB() *sql.DB { return s.db }
