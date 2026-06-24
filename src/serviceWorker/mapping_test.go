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
