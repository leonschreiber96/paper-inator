package api

import (
	"net/http"
	"strings"

	"paperinator/src/shared/models"
)

// Summary endpoints persist configuration. Email rendering and delivery are
// deferred to a later milestone (see src/serviceWorker/summary.go), so these
// handlers manage the stored configuration only.

func (s *Server) listSummaries(w http.ResponseWriter, r *http.Request) {
	summaries, err := s.store.ListSummaries()
	if err != nil {
		writeError(w, err)
		return
	}
	if summaries == nil {
		summaries = []models.Summary{}
	}
	writeJSON(w, http.StatusOK, summaries)
}

func (s *Server) createSummary(w http.ResponseWriter, r *http.Request) {
	var sm models.Summary
	if err := decodeJSON(r, &sm); err != nil {
		writeBadRequest(w, "invalid JSON: "+err.Error())
		return
	}
	if err := validateSummary(&sm); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	if err := s.store.CreateSummary(&sm); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, sm)
}

func (s *Server) updateSummary(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var sm models.Summary
	if err := decodeJSON(r, &sm); err != nil {
		writeBadRequest(w, "invalid JSON: "+err.Error())
		return
	}
	sm.ID = id
	if err := validateSummary(&sm); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	if err := s.store.UpdateSummary(&sm); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sm)
}

func (s *Server) deleteSummary(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.store.DeleteSummary(id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func validateSummary(sm *models.Summary) error {
	if strings.TrimSpace(sm.Name) == "" {
		return errBadField("name is required")
	}
	if !strings.Contains(sm.Recipient, "@") {
		return errBadField("recipient must be an email address")
	}
	if sm.MaxItems < 0 {
		return errBadField("max_items must not be negative")
	}
	return nil
}

type badFieldError string

func (e badFieldError) Error() string { return string(e) }
func errBadField(msg string) error    { return badFieldError(msg) }
