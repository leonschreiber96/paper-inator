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
