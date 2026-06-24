package api

import "net/http"

// reanalyze deletes all existing relevance scores so the enrichment worker will
// re-process every publication on its next tick. Returns 202 Accepted because
// the actual scoring happens asynchronously.
func (s *Server) reanalyze(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteAllScores(); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}
