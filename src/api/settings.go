package api

import (
	"net/http"
	"strings"

	"paperinator/src/shared/models"
)

// Settings are a simple key/value store the frontend uses to persist UI/global
// configuration. Keys are taken from the URL path; values from the request body.

func (s *Server) getSetting(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	value, err := s.store.GetSetting(key)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, models.Setting{Key: key, Value: value})
}

func (s *Server) putSetting(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimSpace(r.PathValue("key"))
	if key == "" {
		writeBadRequest(w, "setting key is required")
		return
	}
	var body models.Setting
	if err := decodeJSON(r, &body); err != nil {
		writeBadRequest(w, "invalid JSON: "+err.Error())
		return
	}
	if err := s.store.SetSetting(key, body.Value); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, models.Setting{Key: key, Value: body.Value})
}
