package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"streamnzb/pkg/auth"
	"streamnzb/pkg/core/config"
	"streamnzb/pkg/core/logger"
	"streamnzb/pkg/stream"
)

const streamConfigsPrefix = "/api/stream/configs/"

func (s *Server) handleStreamConfigs(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/stream/configs" {
		http.NotFound(w, r)
		return
	}
	device, _ := auth.DeviceFromContext(r)
	if device == nil || device.Username != s.config.GetAdminUsername() {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleListStreamConfigs(w, r)
	case http.MethodPost:
		s.handleCreateStreamConfig(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleStreamConfigByID(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, streamConfigsPrefix) {
		http.NotFound(w, r)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, streamConfigsPrefix)
	if id == "" {
		http.Error(w, "stream id required", http.StatusBadRequest)
		return
	}
	device, _ := auth.DeviceFromContext(r)
	if device == nil || device.Username != s.config.GetAdminUsername() {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleGetStreamConfigByID(w, r, id)
	case http.MethodPut:
		s.handlePutStreamConfigByID(w, r, id)
	case http.MethodDelete:
		s.handleDeleteStreamConfig(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleListStreamConfigs(w http.ResponseWriter, r *http.Request) {
	if s.streamManager == nil {
		http.Error(w, "Stream config not available", http.StatusServiceUnavailable)
		return
	}
	list := s.streamManager.List()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(list); err != nil {
		logger.Debug("Stream configs list encode failed", "err", err)
	}
}

func (s *Server) handleGetStreamConfigByID(w http.ResponseWriter, r *http.Request, id string) {
	if s.streamManager == nil {
		http.Error(w, "Stream config not available", http.StatusServiceUnavailable)
		return
	}
	str, err := s.streamManager.Get(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(str); err != nil {
		logger.Debug("Stream config encode failed", "err", err)
	}
}

func (s *Server) handleCreateStreamConfig(w http.ResponseWriter, r *http.Request) {
	if s.streamManager == nil {
		http.Error(w, "Stream config not available", http.StatusServiceUnavailable)
		return
	}
	var body stream.Stream
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if len(body.Filters.QualityRequired) == 0 && len(body.Filters.QualityExcluded) == 0 && body.Filters.MinSizeGB == 0 && body.Filters.MaxSizeGB == 0 {
		body.Filters = config.DefaultFilterConfig()
	}
	if len(body.Sorting.PreferredResolution) == 0 && len(body.Sorting.PreferredCodec) == 0 {
		body.Sorting = config.DefaultSortConfig()
	}
	id, err := s.streamManager.Create(&body)
	if err != nil {
		logger.Debug("Create stream failed", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	created, _ := s.streamManager.Get(id)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", "/api/stream/configs/"+id)
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(created); err != nil {
		logger.Debug("Stream config encode failed", "err", err)
	}
}

func (s *Server) handlePutStreamConfigByID(w http.ResponseWriter, r *http.Request, id string) {
	if s.streamManager == nil {
		http.Error(w, "Stream config not available", http.StatusServiceUnavailable)
		return
	}
	var body stream.Stream
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.streamManager.Set(id, &body); err != nil {
		logger.Debug("Set stream failed", "id", id, "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	updated, _ := s.streamManager.Get(id)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(updated); err != nil {
		logger.Debug("Stream config encode failed", "err", err)
	}
}

func (s *Server) handleDeleteStreamConfig(w http.ResponseWriter, r *http.Request, id string) {
	if s.streamManager == nil {
		http.Error(w, "Stream config not available", http.StatusServiceUnavailable)
		return
	}
	if err := s.streamManager.Delete(id); err != nil {
		if strings.Contains(err.Error(), "last stream") {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
