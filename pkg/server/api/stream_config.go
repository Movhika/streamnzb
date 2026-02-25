package api

import (
	"encoding/json"
	"net/http"

	"streamnzb/pkg/auth"
	"streamnzb/pkg/core/logger"
	"streamnzb/pkg/stream"
)

// handleStreamConfig dispatches GET/PUT /api/stream/config. Admin only.
func (s *Server) handleStreamConfig(w http.ResponseWriter, r *http.Request) {
	device, _ := auth.DeviceFromContext(r)
	if device == nil || device.Username != s.config.GetAdminUsername() {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleGetStreamConfig(w, r)
	case http.MethodPut:
		s.handlePutStreamConfig(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetStreamConfig returns the global stream config. Admin only.
func (s *Server) handleGetStreamConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.streamManager == nil {
		http.Error(w, "Stream config not available", http.StatusServiceUnavailable)
		return
	}
	str := s.streamManager.GetGlobal()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(str); err != nil {
		logger.Debug("Stream config encode failed", "err", err)
	}
}

// handlePutStreamConfig updates the global stream config (PUT /api/stream/config). Admin only.
func (s *Server) handlePutStreamConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.streamManager == nil {
		http.Error(w, "Stream config not available", http.StatusServiceUnavailable)
		return
	}
	var next stream.Stream
	if err := json.NewDecoder(r.Body).Decode(&next); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	next.ID = stream.GlobalStreamID
	if err := s.streamManager.SetGlobal(&next); err != nil {
		logger.Debug("SetGlobal failed", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(s.streamManager.GetGlobal()); err != nil {
		logger.Debug("Stream config encode failed", "err", err)
	}
}
