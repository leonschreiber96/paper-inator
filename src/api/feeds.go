package api

import (
	"net/http"
	"strconv"

	"paperinator/src/shared/models"
	"paperinator/src/shared/validation"
)

func (s *Server) listFeeds(w http.ResponseWriter, r *http.Request) {
	feeds, err := s.store.ListFeeds()
	if err != nil {
		writeError(w, err)
		return
	}
	if feeds == nil {
		feeds = []models.Feed{}
	}
	writeJSON(w, http.StatusOK, feeds)
}

func (s *Server) getFeed(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	feed, err := s.store.GetFeed(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, feed)
}

func (s *Server) createFeed(w http.ResponseWriter, r *http.Request) {
	var feed models.Feed
	if err := decodeJSON(r, &feed); err != nil {
		writeBadRequest(w, "invalid JSON: "+err.Error())
		return
	}
	if err := validation.Feed(&feed); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	if err := s.store.CreateFeed(&feed); err != nil {
		writeError(w, err)
		return
	}
	s.ingester.TriggerFeed(feed.ID)
	writeJSON(w, http.StatusCreated, feed)
}

func (s *Server) updateFeed(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var feed models.Feed
	if err := decodeJSON(r, &feed); err != nil {
		writeBadRequest(w, "invalid JSON: "+err.Error())
		return
	}
	feed.ID = id
	if err := validation.Feed(&feed); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	if err := s.store.UpdateFeed(&feed); err != nil {
		writeError(w, err)
		return
	}
	updated, err := s.store.GetFeed(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) deleteFeed(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.store.DeleteFeed(id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// pathID parses the {id} path segment, writing a 400 and returning false if invalid.
func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeBadRequest(w, "invalid id")
		return 0, false
	}
	return id, true
}
