package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"streamnzb/pkg/auth"
	"streamnzb/pkg/core/config"
)

const apiDevicesPrefix = "/api/devices"

func (s *Server) handleDevices(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, apiDevicesPrefix)
	path = strings.Trim(path, "/")
	if path == "configs" {
		s.handlePutDeviceConfigs(w, r)
		return
	}
	if path == "" {
		if r.Method == http.MethodGet {
			s.handleDevicesList(w, r)
			return
		}
		if r.Method == http.MethodPost {
			s.handleDevicesCreate(w, r)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.handleDeviceByUsername(w, r)
}

func (s *Server) handleDevicesList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	device, _ := auth.DeviceFromContext(r)
	if device == nil || device.Username != s.config.GetAdminUsername() {
		http.Error(w, "Only admin can access devices list", http.StatusForbidden)
		return
	}
	devices := s.deviceManager.GetAllDevices()
	list := make([]map[string]interface{}, 0, len(devices))
	for _, d := range devices {
		list = append(list, map[string]interface{}{
			"username":          d.Username,
			"token":             d.Token,
			"indexer_overrides": d.IndexerOverrides,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (s *Server) handleDevicesCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	device, _ := auth.DeviceFromContext(r)
	if device == nil || device.Username != s.config.GetAdminUsername() {
		http.Error(w, "Only admin can create devices", http.StatusForbidden)
		return
	}
	var req struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	d, err := s.deviceManager.CreateDevice(req.Username, "", s.config.GetAdminUsername())
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"user":    map[string]interface{}{"username": d.Username, "token": d.Token},
	})
}

func (s *Server) handleDeviceByUsername(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, apiDevicesPrefix)
	path = strings.Trim(path, "/")
	parts := strings.SplitN(path, "/", 2)
	username := parts[0]
	if username == "" {
		http.Error(w, "username required", http.StatusBadRequest)
		return
	}
	device, _ := auth.DeviceFromContext(r)
	if device == nil || device.Username != s.config.GetAdminUsername() {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	suffix := ""
	if len(parts) > 1 {
		suffix = parts[1]
	}
	switch r.Method {
	case http.MethodGet:
		if suffix != "" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		d, err := s.deviceManager.GetDevice(username, s.config.GetAdminUsername())
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"username":          d.Username,
			"token":             d.Token,
			"indexer_overrides": d.IndexerOverrides,
		})
	case http.MethodDelete:
		if suffix != "" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		if err := s.deviceManager.DeleteDevice(username); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("Device %s deleted successfully", username),
		})
	case http.MethodPost:
		if suffix != "regenerate-token" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		token, err := s.deviceManager.RegenerateToken(username)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "token": token})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handlePutDeviceConfigs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	device, _ := auth.DeviceFromContext(r)
	if device == nil || device.Username != s.config.GetAdminUsername() {
		http.Error(w, "Only admin can save device configurations", http.StatusForbidden)
		return
	}
	var deviceConfigs map[string]struct {
		IndexerOverrides map[string]config.IndexerSearchConfig `json:"indexer_overrides"`
	}
	if err := json.NewDecoder(r.Body).Decode(&deviceConfigs); err != nil {
		s.writeSaveStatus(w, "error", "Invalid device config data", nil)
		return
	}
	var errors []string
	for username, dc := range deviceConfigs {
		if username == s.config.GetAdminUsername() {
			continue
		}
		if err := s.deviceManager.UpdateDeviceIndexerOverrides(username, dc.IndexerOverrides); err != nil {
			errors = append(errors, fmt.Sprintf("Failed to update indexer overrides for %s: %v", username, err))
		}
	}
	if len(errors) > 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "error",
			"message": "Some device configs failed to save",
			"errors":  errors,
		})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Device configurations saved successfully",
	})
}
