package serviceWorker

import (
	"context"
	"fmt"
	"log"
	"time"

	"paperinator/src/shared/store"
)

const (
	enrichBatchSize = 20
	llmPauseBetween = 200 * time.Millisecond
)

// Enricher runs as a background goroutine and assigns relevance scores to
// publications that don't have one yet. It reads the scorer type and interest
// profile from the settings table on every tick, so configuration changes take
// effect without restarting the server.
type Enricher struct {
	store *store.Store
	tick  time.Duration
}

// NewEnricher creates an Enricher that wakes every tick to process unscored
// publications. The default tick (2 minutes) keeps the database current without
// hammering external LLM APIs.
func NewEnricher(s *store.Store) *Enricher {
	return &Enricher{store: s, tick: 2 * time.Minute}
}

// Run blocks until ctx is cancelled, running an enrichment batch on each tick
// and once immediately at startup.
func (e *Enricher) Run(ctx context.Context) {
	log.Printf("enricher started (tick %s)", e.tick)
	e.enrichBatch(ctx)

	ticker := time.NewTicker(e.tick)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Printf("enricher stopping")
			return
		case <-ticker.C:
			e.enrichBatch(ctx)
		}
	}
}

// enrichBatch scores up to enrichBatchSize unscored publications using the
// currently configured scorer. Errors per publication are logged and skipped;
// they will be retried on the next tick.
func (e *Enricher) enrichBatch(ctx context.Context) {
	profile, err := e.store.GetSetting("interest_profile")
	if err != nil || profile == "" {
		return // scoring disabled until the user sets an interest profile
	}

	scorer, scorerType, err := e.buildScorer()
	if err != nil {
		log.Printf("enricher: build scorer: %v", err)
		return
	}

	pubs, err := e.store.ListUnscoredPublications(enrichBatchSize)
	if err != nil {
		log.Printf("enricher: list unscored: %v", err)
		return
	}
	if len(pubs) == 0 {
		return
	}

	var scored, failed int
	for _, pub := range pubs {
		if ctx.Err() != nil {
			break
		}
		score, notes, err := scorer.Score(ctx, pub.Title, pub.Abstract, profile)
		if err != nil {
			log.Printf("enricher: score pub %d (%q): %v", pub.ID, pub.Title, err)
			failed++
			continue
		}
		if err := e.store.UpsertPublicationScore(pub.ID, score, notes, scorerType); err != nil {
			log.Printf("enricher: store score for pub %d: %v", pub.ID, err)
			failed++
			continue
		}
		scored++
		// Pace LLM requests to avoid bursting external APIs.
		if scorerType == "llm" {
			select {
			case <-ctx.Done():
				break
			case <-time.After(llmPauseBetween):
			}
		}
	}
	log.Printf("enricher: scored %d, failed %d", scored, failed)
}

// buildScorer reads scorer configuration from settings and constructs the
// appropriate RelevanceScorer. Returns the scorer and its type name ("keyword"
// or "llm") so the type can be recorded alongside the score.
func (e *Enricher) buildScorer() (RelevanceScorer, string, error) {
	scorerType, _ := e.store.GetSetting("relevance_scorer")
	if scorerType == "" {
		scorerType = "keyword"
	}

	switch scorerType {
	case "keyword":
		return KeywordScorer{}, "keyword", nil
	case "llm":
		baseURL, _ := e.store.GetSetting("llm_base_url")
		apiKey, _ := e.store.GetSetting("llm_api_key")
		model, _ := e.store.GetSetting("llm_model")
		if baseURL == "" || model == "" {
			return nil, "", fmt.Errorf("llm scorer requires llm_base_url and llm_model settings")
		}
		return NewLLMScorer(baseURL, apiKey, model), "llm", nil
	default:
		return nil, "", fmt.Errorf("unknown relevance_scorer: %q", scorerType)
	}
}
