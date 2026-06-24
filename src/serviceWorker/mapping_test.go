package serviceWorker

import (
	"testing"

	"paperinator/src/shared/models"
)

func TestMapItemDefaults(t *testing.T) {
	it := Item{Title: "T", Authors: "A", Summary: "S", Link: "L"}
	pub := MapItem(7, it, nil)

	if pub.FeedID != 7 {
		t.Errorf("feed id = %d", pub.FeedID)
	}
	if pub.Title != "T" || pub.Authors != "A" || pub.Abstract != "S" || pub.Link != "L" {
		t.Errorf("default mapping wrong: %+v", pub)
	}
	if pub.DedupKey == "" {
		t.Error("dedup key should be populated")
	}
	if pub.FetchedAt.IsZero() {
		t.Error("fetched_at should be set")
	}
}

func TestMapItemOverrideFromExtra(t *testing.T) {
	// A feed where the author lives in a non-standard "creator" element that the
	// parser captured into Extra; a mapping promotes it to the authors field.
	it := Item{Title: "T", Authors: "", Extra: map[string]string{"creator": "Ada Lovelace"}}
	mappings := []models.FieldMapping{{SourceField: "creator", TargetField: "authors"}}

	pub := MapItem(1, it, mappings)
	if pub.Authors != "Ada Lovelace" {
		t.Errorf("authors = %q, expected mapping from Extra", pub.Authors)
	}
}

func TestMapItemEmptyOverrideDoesNotClobber(t *testing.T) {
	// A mapping pointing at a missing source must not wipe out a good default.
	it := Item{Title: "Good Title"}
	mappings := []models.FieldMapping{{SourceField: "nonexistent", TargetField: "title"}}

	pub := MapItem(1, it, mappings)
	if pub.Title != "Good Title" {
		t.Errorf("title = %q, expected default to be preserved", pub.Title)
	}
}

func TestCollectDiscoveredFields(t *testing.T) {
	items := []Item{
		{Title: "Paper One", Authors: "Ada", Summary: "First abstract", Link: "https://example.org/1",
			Extra: map[string]string{"guid": "guid-1", "category": "cs.AI"}},
		{Title: "Paper Two", Authors: "", Summary: "Second abstract", Link: "https://example.org/2",
			Extra: map[string]string{"guid": "guid-2", "custom": "extra value"}},
	}
	got := CollectDiscoveredFields(items)

	// Standard fields from Item struct
	if got["title"] != "Paper One" {
		t.Errorf("title sample = %q", got["title"])
	}
	if got["authors"] != "Ada" {
		t.Errorf("authors sample = %q, should be first non-empty", got["authors"])
	}
	if got["summary"] != "First abstract" {
		t.Errorf("summary sample = %q", got["summary"])
	}
	if got["link"] != "https://example.org/1" {
		t.Errorf("link sample = %q", got["link"])
	}
	// Extra fields
	if got["guid"] != "guid-1" {
		t.Errorf("guid sample = %q", got["guid"])
	}
	if got["category"] != "cs.AI" {
		t.Errorf("category sample = %q", got["category"])
	}
	if got["custom"] != "extra value" {
		t.Errorf("custom sample = %q", got["custom"])
	}
	// First sample must not be overwritten by later items
	if got["authors"] == "" {
		t.Error("authors should come from first item (non-empty), not second (empty)")
	}
}

func TestCollectDiscoveredFieldsEmpty(t *testing.T) {
	if got := CollectDiscoveredFields(nil); len(got) != 0 {
		t.Errorf("expected empty map for nil items, got %v", got)
	}
}

func TestAutoAssignMappings(t *testing.T) {
	fields := map[string]string{
		"title":        "A Paper",
		"authors":      "Ada Lovelace",
		"summary":      "An abstract",
		"link":         "https://example.org",
		"published_at": "2024-01-01",
		"guid":         "some-guid",
	}
	mappings := AutoAssignMappings(fields)

	byTarget := make(map[string]string)
	for _, m := range mappings {
		byTarget[m.TargetField] = m.SourceField
	}
	if byTarget["title"] != "title" {
		t.Errorf("title source = %q", byTarget["title"])
	}
	if byTarget["authors"] != "authors" {
		t.Errorf("authors source = %q", byTarget["authors"])
	}
	if byTarget["abstract"] != "summary" {
		t.Errorf("abstract source = %q (expected summary)", byTarget["abstract"])
	}
	if byTarget["link"] != "link" {
		t.Errorf("link source = %q", byTarget["link"])
	}
	if byTarget["published_at"] != "published_at" {
		t.Errorf("published_at source = %q", byTarget["published_at"])
	}
	// guid has no target — must not appear
	for _, m := range mappings {
		if m.SourceField == "guid" {
			t.Error("guid should not be auto-assigned to any target")
		}
	}
}

func TestAutoAssignMappingsFallback(t *testing.T) {
	// Feed only has Extra "creator" for authors (e.g. dc:creator parsed into Extra)
	// and "date" for published_at
	fields := map[string]string{
		"title":   "Paper",
		"creator": "Charles Babbage",
		"date":    "2024-06-01",
		"link":    "https://example.org",
	}
	mappings := AutoAssignMappings(fields)
	byTarget := make(map[string]string)
	for _, m := range mappings {
		byTarget[m.TargetField] = m.SourceField
	}
	if byTarget["authors"] != "creator" {
		t.Errorf("authors fallback to creator: got %q", byTarget["authors"])
	}
	if byTarget["published_at"] != "date" {
		t.Errorf("published_at fallback to date: got %q", byTarget["published_at"])
	}
	// No abstract source — target must be absent
	if _, ok := byTarget["abstract"]; ok {
		t.Errorf("abstract should not be assigned when no matching source exists, got %q", byTarget["abstract"])
	}
}
