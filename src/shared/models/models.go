// Package models defines the core data structures shared across the service
// worker, the API, and the persistence layer. Keeping them in one place avoids
// duplicating field definitions and keeps JSON/DB representations consistent.
package models

import "time"

// Feed is an RSS/Atom source that the service worker polls for publications.
type Feed struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	URL             string     `json:"url"`
	Enabled         bool       `json:"enabled"`
	FetchIntervalSec int       `json:"fetch_interval_sec"`
	LastFetchedAt   *time.Time `json:"last_fetched_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// FieldMapping describes how a single field in a feed's items maps onto one of
// the internal Publication fields. Mappings are configured per feed so that
// heterogeneous RSS formats can be normalized into a common shape.
type FieldMapping struct {
	ID          int64  `json:"id"`
	FeedID      int64  `json:"feed_id"`
	SourceField string `json:"source_field"` // e.g. "dc:creator", "summary"
	TargetField string `json:"target_field"` // one of the Publication fields below
}

// Publication is a normalized, deduplicated academic publication.
type Publication struct {
	ID          int64      `json:"id"`
	FeedID      int64      `json:"feed_id"`
	Title       string     `json:"title"`
	Authors     string     `json:"authors"`
	Abstract    string     `json:"abstract"`
	Link        string     `json:"link"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	FetchedAt   time.Time  `json:"fetched_at"`
	DedupKey    string     `json:"dedup_key"`
	Raw         string     `json:"raw,omitempty"` // original item as JSON, for debugging/remapping
}

// Summary is a user-configured email digest of new publications.
type Summary struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	Recipient string  `json:"recipient"`
	FeedIDs   []int64 `json:"feed_ids"`
	MaxItems  int     `json:"max_items"`
	Schedule  string  `json:"schedule"` // cron-like or simple keyword; defined when feature lands
	Enabled   bool    `json:"enabled"`
}

// Setting is a single key/value entry in the persisted settings store, used for
// frontend and global configuration.
type Setting struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// FeedField records a source field name discovered in a feed during ingestion.
// SampleValue holds the first non-empty value seen, shown in the mapping UI
// so users can identify fields without knowing the RSS spec.
type FeedField struct {
	FeedID      int64  `json:"feed_id"`
	FieldName   string `json:"field_name"`
	SampleValue string `json:"sample_value"`
}

// ValidTargetFields enumerates the Publication fields that a FieldMapping may
// target. Used by validation and by the mapping layer's defaults.
var ValidTargetFields = []string{"title", "authors", "abstract", "link", "published_at"}
