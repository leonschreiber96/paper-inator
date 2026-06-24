package api

import (
	"net/http"

	"paperinator/src/shared/models"
	"paperinator/src/shared/validation"
)

func (s *Server) getMappings(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	mappings, err := s.store.ListMappings(id)
	if err != nil {
		writeError(w, err)
		return
	}
	if mappings == nil {
		mappings = []models.FieldMapping{}
	}
	writeJSON(w, http.StatusOK, mappings)
}

// getFeedFields returns the source fields discovered for a feed during the last
// ingest run. Returns an empty array before the first ingest.
func (s *Server) getFeedFields(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if _, err := s.store.GetFeed(id); err != nil {
		writeError(w, err)
		return
	}
	fields, err := s.store.ListFeedFields(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, fields)
}

// triggerIngest starts an immediate ingest of the feed and returns 202 Accepted.
// The ingest runs asynchronously; the caller should poll /api/feeds/{id}/fields
// afterwards to see refreshed discovered fields.
func (s *Server) triggerIngest(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if _, err := s.store.GetFeed(id); err != nil {
		writeError(w, err)
		return
	}
	s.ingester.TriggerFeed(id)
	w.WriteHeader(http.StatusAccepted)
}

// putMappings replaces the full set of mappings for a feed with the request body.
func (s *Server) putMappings(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if _, err := s.store.GetFeed(id); err != nil {
		writeError(w, err)
		return
	}
	var mappings []models.FieldMapping
	if err := decodeJSON(r, &mappings); err != nil {
		writeBadRequest(w, "invalid JSON: "+err.Error())
		return
	}
	for i := range mappings {
		mappings[i].FeedID = id
		if err := validation.FieldMapping(&mappings[i]); err != nil {
			writeBadRequest(w, err.Error())
			return
		}
	}
	if err := s.store.ReplaceMappings(id, mappings); err != nil {
		writeError(w, err)
		return
	}
	s.getMappings(w, r)
}
