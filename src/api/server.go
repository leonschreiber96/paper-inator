// Package api exposes the REST interface and serves the embedded frontend. It is
// a thin layer over the shared store: handlers parse requests, call store
// methods, and encode JSON responses. All routes are registered on the standard
// library's http.ServeMux using Go 1.22+ method+pattern matching, so no
// third-party router is required.
package api

import (
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"time"

	"paperinator/src/frontend"
	"paperinator/src/shared/store"
)

// FeedIngester is implemented by the service worker. The interface keeps the api
// package decoupled from the serviceWorker package.
type FeedIngester interface {
	TriggerFeed(feedID int64)
}

// Server holds dependencies shared by the HTTP handlers.
type Server struct {
	store    *store.Store
	ingester FeedIngester
}

// NewServer builds the HTTP handler tree: REST API under /api/ plus the embedded
// static frontend at /.
func NewServer(s *store.Store, ingester FeedIngester) http.Handler {
	srv := &Server{store: s, ingester: ingester}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", srv.handleHealth)

	mux.HandleFunc("GET /api/feeds", srv.listFeeds)
	mux.HandleFunc("POST /api/feeds", srv.createFeed)
	mux.HandleFunc("GET /api/feeds/{id}", srv.getFeed)
	mux.HandleFunc("PUT /api/feeds/{id}", srv.updateFeed)
	mux.HandleFunc("DELETE /api/feeds/{id}", srv.deleteFeed)

	mux.HandleFunc("GET /api/feeds/{id}/mappings", srv.getMappings)
	mux.HandleFunc("PUT /api/feeds/{id}/mappings", srv.putMappings)

	mux.HandleFunc("GET /api/publications", srv.listPublications)

	mux.HandleFunc("GET /api/summaries", srv.listSummaries)
	mux.HandleFunc("POST /api/summaries", srv.createSummary)
	mux.HandleFunc("PUT /api/summaries/{id}", srv.updateSummary)
	mux.HandleFunc("DELETE /api/summaries/{id}", srv.deleteSummary)

	mux.HandleFunc("GET /api/settings/{key}", srv.getSetting)
	mux.HandleFunc("PUT /api/settings/{key}", srv.putSetting)

	// Serve the embedded frontend for everything else.
	static, err := fs.Sub(frontend.Static, "static")
	if err != nil {
		log.Fatalf("frontend assets: %v", err)
	}
	mux.Handle("/", http.FileServer(http.FS(static)))

	return logRequests(mux)
}

// handleHealth is a trivial liveness endpoint.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- shared response helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

// writeError maps store/validation errors to appropriate HTTP status codes.
func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, store.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, store.ErrConflict):
		status = http.StatusConflict
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeBadRequest(w http.ResponseWriter, msg string) {
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
}

// decodeJSON reads and decodes a JSON request body into v.
func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// logRequests is a minimal logging middleware.
func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}
