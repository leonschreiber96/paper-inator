package api

import (
	"net/http"
	"strconv"

	"paperinator/src/shared/models"
	"paperinator/src/shared/store"
)

// listPublications returns publications filtered and sorted via query parameters:
//
//	feed_id    restrict to one feed
//	q          case-insensitive search over title/authors
//	sort       published_at (default) | fetched_at | title | relevance
//	order      desc (default) | asc
//	limit      page size (default 50)
//	offset     page offset
//	min_score  only return publications with relevance_score >= this value (0.0–1.0)
func (s *Server) listPublications(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := store.PublicationFilter{
		Search: q.Get("q"),
		SortBy: q.Get("sort"),
		Desc:   q.Get("order") != "asc",
		Limit:  atoiDefault(q.Get("limit"), 50),
		Offset: atoiDefault(q.Get("offset"), 0),
	}
	if fid := atoiDefault(q.Get("feed_id"), 0); fid > 0 {
		filter.FeedID = int64(fid)
	}
	if ms := q.Get("min_score"); ms != "" {
		if v, err := strconv.ParseFloat(ms, 64); err == nil && v >= 0 && v <= 1 {
			filter.MinScore = &v
		}
	}

	pubs, err := s.store.ListPublications(filter)
	if err != nil {
		writeError(w, err)
		return
	}
	if pubs == nil {
		pubs = []models.Publication{}
	}
	writeJSON(w, http.StatusOK, pubs)
}

func atoiDefault(s string, def int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}
