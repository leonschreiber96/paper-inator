package store

import (
	"path/filepath"
	"testing"
	"time"

	"paperinator/src/shared/models"
)

// newTestStore opens a fresh on-disk SQLite DB in a temp dir (auto-removed) and
// runs migrations. A file (not :memory:) is used so WAL and foreign keys behave
// exactly as in production.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestMigrationsApplyAndAreIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	s1, err := Open(path)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	s1.Close()

	// Re-opening must not attempt to re-apply migrations.
	s2, err := Open(path)
	if err != nil {
		t.Fatalf("second open (idempotency): %v", err)
	}
	defer s2.Close()

	var count int
	if err := s2.DB().QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if count < 1 {
		t.Errorf("expected at least 1 applied migration, got %d", count)
	}
}

func TestFeedCRUD(t *testing.T) {
	s := newTestStore(t)

	feed := &models.Feed{Name: "Journal", URL: "https://example.org/rss", Enabled: true}
	if err := s.CreateFeed(feed); err != nil {
		t.Fatalf("create: %v", err)
	}
	if feed.ID == 0 {
		t.Fatal("expected feed ID to be set")
	}

	got, err := s.GetFeed(feed.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "Journal" {
		t.Errorf("name = %q", got.Name)
	}

	got.Name = "Renamed"
	if err := s.UpdateFeed(got); err != nil {
		t.Fatalf("update: %v", err)
	}
	reloaded, _ := s.GetFeed(feed.ID)
	if reloaded.Name != "Renamed" {
		t.Errorf("update not persisted, name = %q", reloaded.Name)
	}

	if err := s.DeleteFeed(feed.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.GetFeed(feed.ID); err != ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestInsertPublicationDeduplicates(t *testing.T) {
	s := newTestStore(t)
	feed := &models.Feed{Name: "J", URL: "https://example.org/x", Enabled: true}
	if err := s.CreateFeed(feed); err != nil {
		t.Fatalf("create feed: %v", err)
	}

	pub := &models.Publication{FeedID: feed.ID, Title: "Paper", Authors: "A", DedupKey: "key-1"}
	inserted, err := s.InsertPublication(pub)
	if err != nil || !inserted {
		t.Fatalf("first insert: inserted=%v err=%v", inserted, err)
	}

	// Same dedup key must be rejected (skipped), not error.
	dup := &models.Publication{FeedID: feed.ID, Title: "Paper (reposted)", Authors: "A", DedupKey: "key-1"}
	inserted, err = s.InsertPublication(dup)
	if err != nil {
		t.Fatalf("duplicate insert errored: %v", err)
	}
	if inserted {
		t.Error("expected duplicate to be skipped, but it was inserted")
	}

	pubs, err := s.ListPublications(PublicationFilter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(pubs) != 1 {
		t.Errorf("expected 1 publication after dedup, got %d", len(pubs))
	}
}

func TestPublicationTimeRoundTrip(t *testing.T) {
	// Regression: TIMESTAMP columns must survive a write/read cycle through the
	// pure-Go SQLite driver, which returns them as text.
	s := newTestStore(t)
	feed := &models.Feed{Name: "J", URL: "https://example.org/t", Enabled: true}
	if err := s.CreateFeed(feed); err != nil {
		t.Fatalf("create feed: %v", err)
	}

	published := time.Date(2020, 5, 1, 12, 0, 0, 0, time.UTC)
	pub := &models.Publication{FeedID: feed.ID, Title: "T", Authors: "A", PublishedAt: &published, DedupKey: "k"}
	if _, err := s.InsertPublication(pub); err != nil {
		t.Fatalf("insert: %v", err)
	}

	got, err := s.ListPublications(PublicationFilter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 publication, got %d", len(got))
	}
	if got[0].PublishedAt == nil || !got[0].PublishedAt.Equal(published) {
		t.Errorf("published_at round-trip failed: got %v, want %v", got[0].PublishedAt, published)
	}
	if got[0].FetchedAt.IsZero() {
		t.Error("fetched_at should be populated from the DB default")
	}
}

func TestFeedFieldsUpsertAndList(t *testing.T) {
	s := newTestStore(t)
	feed := &models.Feed{Name: "J", URL: "https://example.org/ff", Enabled: true}
	if err := s.CreateFeed(feed); err != nil {
		t.Fatalf("create feed: %v", err)
	}

	// First upsert — populates sample values
	if err := s.UpsertFeedFields(feed.ID, map[string]string{
		"title":   "First Paper",
		"creator": "Ada Lovelace",
	}); err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	// Second upsert — must not overwrite existing samples; adds a new field
	if err := s.UpsertFeedFields(feed.ID, map[string]string{
		"title":   "Second Paper",   // existing — sample must stay "First Paper"
		"summary": "An abstract.",   // new
	}); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	got, err := s.ListFeedFields(feed.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(got))
	}
	byName := make(map[string]string)
	for _, f := range got {
		byName[f.FieldName] = f.SampleValue
	}
	if byName["title"] != "First Paper" {
		t.Errorf("sample should not be overwritten; got %q", byName["title"])
	}
	if byName["creator"] != "Ada Lovelace" {
		t.Errorf("creator sample missing; got %q", byName["creator"])
	}
	if byName["summary"] != "An abstract." {
		t.Errorf("new field sample wrong; got %q", byName["summary"])
	}
}

func TestSettingsRoundTrip(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.GetSetting("theme"); err != ErrNotFound {
		t.Errorf("expected ErrNotFound for missing setting, got %v", err)
	}
	if err := s.SetSetting("theme", "dark"); err != nil {
		t.Fatalf("set: %v", err)
	}
	if err := s.SetSetting("theme", "light"); err != nil { // upsert
		t.Fatalf("upsert: %v", err)
	}
	v, err := s.GetSetting("theme")
	if err != nil || v != "light" {
		t.Errorf("got %q, %v; want \"light\"", v, err)
	}
}
