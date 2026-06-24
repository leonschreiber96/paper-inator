package serviceWorker

import "context"

// RelevanceScorer scores a single publication against a user-defined interest
// profile. Implementations must be safe for concurrent use.
//
// score is in [0.0, 1.0] where 1.0 is maximally relevant.
// notes is a human-readable explanation shown in the UI (e.g. matched terms,
// or the LLM's one-sentence reasoning).
// A non-nil error means the scoring failed transiently; the caller should leave
// the publication unscored and retry on the next enrichment cycle.
type RelevanceScorer interface {
	Score(ctx context.Context, title, abstract, profile string) (score float64, notes string, err error)
}
