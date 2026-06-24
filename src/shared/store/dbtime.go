package store

import (
	"database/sql"
	"fmt"
	"time"
)

// The pure-Go SQLite driver stores and returns TIMESTAMP columns as text, so we
// convert times explicitly rather than scanning straight into time.Time. Storing
// in SQLite's own CURRENT_TIMESTAMP format ("2006-01-02 15:04:05", UTC) keeps
// values written by Go and by SQL defaults identical and comparable.

const dbTimeFormat = "2006-01-02 15:04:05"

// dbTimeLayouts lists the formats we accept when reading a stored timestamp,
// covering SQLite's default format and RFC3339 values that may arrive from feeds.
var dbTimeLayouts = []string{
	dbTimeFormat,
	"2006-01-02 15:04:05.999999999",
	"2006-01-02 15:04:05-07:00",
	time.RFC3339Nano,
	time.RFC3339,
}

// formatDBTime renders a time for storage in UTC using SQLite's default format.
func formatDBTime(t time.Time) string {
	return t.UTC().Format(dbTimeFormat)
}

// parseDBTime parses a stored timestamp string using the accepted layouts.
func parseDBTime(s string) (time.Time, error) {
	for _, l := range dbTimeLayouts {
		if t, err := time.Parse(l, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized time format %q", s)
}

// scanTime converts a nullable text timestamp into an optional time.Time.
func scanTime(ns sql.NullString) (*time.Time, error) {
	if !ns.Valid || ns.String == "" {
		return nil, nil
	}
	t, err := parseDBTime(ns.String)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// scanRequiredTime converts a non-null text timestamp into a time.Time.
func scanRequiredTime(s string) (time.Time, error) {
	return parseDBTime(s)
}
