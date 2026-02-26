package api

import (
	"encoding/json"
	"net/http"
	"path/filepath"

	"streamnzb/pkg/auth"
	"streamnzb/pkg/core/config"
	"streamnzb/pkg/core/paths"
)

// configPayload is the response for GET /api/config; includes env_overrides for admin.
type configPayload struct {
	config.Config
	EnvOverrides []string `json:"env_overrides,omitempty"`
}

// handleConfig routes GET and PUT /api/config.
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.handleGetConfig(w, r)
		return
	}
	if r.Method == http.MethodPut {
		s.handlePutConfig(w, r)
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// handleGetConfig returns config for the authenticated device (GET /api/config).
// Ensures admin_username is never returned empty so the UI does not overwrite with "" on refetch (e.g. after WS reconnect).
func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	device, _ := auth.DeviceFromContext(r)
	var cfg config.Config
	if device != nil && device.Username == s.config.GetAdminUsername() {
		cfg = s.config.RedactForAPI()
	} else if device != nil {
		cfg = *s.config
		cfg = cfg.RedactForAPI()
	} else {
		cfg = s.config.RedactForAPI()
	}
	if cfg.AdminUsername == "" {
		cfg.AdminUsername = s.config.GetAdminUsername()
	}
	w.Header().Set("Content-Type", "application/json")
	if device != nil && device.Username == s.config.GetAdminUsername() {
		envKeys := config.GetEnvOverrideKeys()
		json.NewEncoder(w).Encode(configPayload{Config: cfg, EnvOverrides: envKeys})
	} else {
		json.NewEncoder(w).Encode(cfg)
	}
}

// handlePutConfig saves global config (PUT /api/config). Admin only.
func (s *Server) handlePutConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	device, _ := auth.DeviceFromContext(r)
	if device == nil || device.Username != s.config.GetAdminUsername() {
		http.Error(w, "Only admin can save global configuration", http.StatusForbidden)
		return
	}
	var newCfg config.Config
	if err := json.NewDecoder(r.Body).Decode(&newCfg); err != nil {
		s.writeSaveStatus(w, "error", "Invalid config data", nil)
		return
	}
	fieldErrors := s.validateConfig(&newCfg)
	if len(fieldErrors) > 0 {
		s.writeSaveStatus(w, "error", "Validation failed", fieldErrors)
		return
	}
	s.mu.RLock()
	currentCfg := s.config
	currentLoadedPath := s.config.LoadedPath
	s.mu.RUnlock()
	config.CopyEnvOverridesFrom(currentCfg, &newCfg)
	newCfg.AdminPasswordHash = currentCfg.AdminPasswordHash
	newCfg.AdminToken = currentCfg.AdminToken
	newCfg.AdminMustChangePassword = currentCfg.AdminMustChangePassword
	if newCfg.AdminUsername == "" {
		newCfg.AdminUsername = currentCfg.GetAdminUsername()
	}
	newCfg.ApplyProviderDefaults()
	if currentLoadedPath == "" {
		currentLoadedPath = filepath.Join(paths.GetDataDir(), "config.json")
	}
	newCfg.LoadedPath = currentLoadedPath
	s.mu.Lock()
	s.config = &newCfg
	s.mu.Unlock()
	if err := s.config.Save(); err != nil {
		s.writeSaveStatus(w, "error", "Failed to save config: "+err.Error(), nil)
		return
	}
	s.reloadConfigAsync(&newCfg)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Configuration saved and reloaded.",
	})
}

func (s *Server) writeSaveStatus(w http.ResponseWriter, status, message string, errors map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	if status == "error" {
		w.WriteHeader(http.StatusBadRequest)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  status,
		"message": message,
		"errors":  errors,
	})
}
