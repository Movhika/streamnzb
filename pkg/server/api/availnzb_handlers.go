package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"streamnzb/pkg/auth"
	"streamnzb/pkg/services/availnzb"
)

func (s *Server) handleAvailNZBStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	device, _ := auth.DeviceFromContext(r)
	if device == nil || device.Username != s.config.GetAdminUsername() {
		http.Error(w, "Only admin can access AvailNZB key status", http.StatusForbidden)
		return
	}

	s.mu.RLock()
	availNZBURL := s.availNZBURL
	availNZBAPIKey := s.availNZBAPIKey
	s.mu.RUnlock()

	if availNZBURL == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "AvailNZB URL is not configured"})
		return
	}
	if availNZBAPIKey == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "AvailNZB API key is not configured"})
		return
	}

	status, err := availnzb.NewClient(availNZBURL, availNZBAPIKey).GetMe()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Failed to fetch AvailNZB key status: %v", err)})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}