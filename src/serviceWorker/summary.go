package serviceWorker

import (
	"paperinator/src/shared/models"
	"paperinator/src/shared/store"
)

// BuildSummaryItems selects the publications that a summary configuration would
// include: the most recent items from the summary's selected feeds, capped at
// MaxItems. This is the pure, testable core of summary generation.
//
// Email rendering and SMTP delivery are intentionally not implemented in this
// milestone (see plan: summaries are deferred). They will build on top of this
// selection function so the selection logic can be tested independently.
func BuildSummaryItems(s *store.Store, summary models.Summary) ([]models.Publication, error) {
	limit := summary.MaxItems
	if limit <= 0 {
		limit = 10
	}

	// If no feeds are selected, treat the summary as spanning all feeds.
	if len(summary.FeedIDs) == 0 {
		return s.ListPublications(store.PublicationFilter{Limit: limit, Desc: true})
	}

	var out []models.Publication
	for _, feedID := range summary.FeedIDs {
		pubs, err := s.ListPublications(store.PublicationFilter{FeedID: feedID, Limit: limit, Desc: true})
		if err != nil {
			return nil, err
		}
		out = append(out, pubs...)
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// TODO(summaries): render BuildSummaryItems output to an email body and send it
// via net/smtp using the SMTP settings in config.Config. Add scheduling so
// enabled summaries are delivered on their configured cadence.
