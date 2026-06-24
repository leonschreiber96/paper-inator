// Package serviceWorker owns feed ingestion: fetching feeds, parsing them,
// applying per-feed field mappings, deduplicating, and storing publications. It
// runs as a background goroutine inside the unified binary.
package serviceWorker

import (
	"context"
	"log"
	"time"

	"paperinator/src/shared/models"
	"paperinator/src/shared/store"
)

// Worker polls enabled feeds on a schedule and ingests new publications.
type Worker struct {
	store           *store.Store
	defaultInterval time.Duration
	tick            time.Duration // how often the loop wakes to check which feeds are due
}

// New creates a Worker. defaultInterval is used for feeds that don't specify
// their own fetch interval.
func New(s *store.Store, defaultInterval time.Duration) *Worker {
	return &Worker{
		store:           s,
		defaultInterval: defaultInterval,
		tick:            time.Minute,
	}
}

// Run blocks until ctx is cancelled, polling due feeds on each tick. On startup
// it scrapes all enabled feeds immediately so the database is fresh regardless
// of when the server last ran.
func (w *Worker) Run(ctx context.Context) {
	log.Printf("service worker started (default interval %s)", w.defaultInterval)
	w.pollAll(ctx)

	ticker := time.NewTicker(w.tick)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Printf("service worker stopping")
			return
		case <-ticker.C:
			w.pollDue(ctx)
		}
	}
}

// TriggerFeed immediately ingests a single feed by ID in a new goroutine. It is
// safe to call from the API handler after creating a feed so the user sees
// results without waiting for the next scheduled tick. Errors are logged, not
// returned, because the caller cannot act on them.
func (w *Worker) TriggerFeed(feedID int64) {
	go func() {
		feed, err := w.store.GetFeed(feedID)
		if err != nil {
			log.Printf("worker: trigger feed %d: %v", feedID, err)
			return
		}
		if err := w.ingest(context.Background(), *feed); err != nil {
			log.Printf("worker: trigger feed %d (%s): %v", feedID, feed.Name, err)
		}
	}()
}

// pollAll ingests every enabled feed unconditionally. Used at startup so the
// database is always up-to-date when the server comes online.
func (w *Worker) pollAll(ctx context.Context) {
	feeds, err := w.store.ListEnabledFeeds()
	if err != nil {
		log.Printf("worker: list feeds: %v", err)
		return
	}
	for _, f := range feeds {
		if err := w.ingest(ctx, f); err != nil {
			log.Printf("worker: ingest feed %d (%s): %v", f.ID, f.Name, err)
		}
	}
}

// pollDue ingests every enabled feed whose interval has elapsed since its last fetch.
func (w *Worker) pollDue(ctx context.Context) {
	feeds, err := w.store.ListEnabledFeeds()
	if err != nil {
		log.Printf("worker: list feeds: %v", err)
		return
	}
	for _, f := range feeds {
		if !w.due(f) {
			continue
		}
		if err := w.ingest(ctx, f); err != nil {
			log.Printf("worker: ingest feed %d (%s): %v", f.ID, f.Name, err)
		}
	}
}

// due reports whether a feed should be polled now.
func (w *Worker) due(f models.Feed) bool {
	if f.LastFetchedAt == nil {
		return true
	}
	return time.Since(*f.LastFetchedAt) >= w.interval(f)
}

func (w *Worker) interval(f models.Feed) time.Duration {
	if f.FetchIntervalSec > 0 {
		return time.Duration(f.FetchIntervalSec) * time.Second
	}
	return w.defaultInterval
}

// ingest runs the full fetch -> parse -> map -> dedup -> store pipeline for one feed.
func (w *Worker) ingest(ctx context.Context, f models.Feed) error {
	data, err := fetch(ctx, f.URL)
	if err != nil {
		return err
	}
	items, err := Parse(data)
	if err != nil {
		return err
	}
	mappings, err := w.store.ListMappings(f.ID)
	if err != nil {
		return err
	}

	var added, skipped int
	for _, it := range items {
		if it.Title == "" {
			continue // a publication with no title cannot be deduplicated meaningfully
		}
		pub := MapItem(f.ID, it, mappings)
		inserted, err := w.store.InsertPublication(&pub)
		if err != nil {
			log.Printf("worker: store publication %q: %v", pub.Title, err)
			continue
		}
		if inserted {
			added++
		} else {
			skipped++ // duplicate: deterministic dedup by title+authors
		}
	}

	if err := w.store.MarkFeedFetched(f.ID); err != nil {
		return err
	}
	log.Printf("worker: feed %d (%s): %d new, %d duplicates", f.ID, f.Name, added, skipped)
	return nil
}
