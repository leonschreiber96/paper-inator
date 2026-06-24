package serviceWorker

import (
	"path/filepath"
	"testing"

	"paperinator/src/shared/models"
	"paperinator/src/shared/store"
)

func TestBuildSummaryItemsRespectsMaxItems(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer s.Close()

	feed := &models.Feed{Name: "J", URL: "https://example.org/s", Enabled: true}
	if err := s.CreateFeed(feed); err != nil {
		t.Fatalf("create feed: %v", err)
	}
	for i := 0; i < 5; i++ {
		pub := MapItem(feed.ID, Item{Title: "Paper " + string(rune('A'+i)), Authors: "X"}, nil)
		if _, err := s.InsertPublication(&pub); err != nil {
			t.Fatalf("insert pub: %v", err)
		}
	}

	items, err := BuildSummaryItems(s, models.Summary{FeedIDs: []int64{feed.ID}, MaxItems: 3})
	if err != nil {
		t.Fatalf("build summary: %v", err)
	}
	if len(items) != 3 {
		t.Errorf("expected 3 items (capped by MaxItems), got %d", len(items))
	}
}
