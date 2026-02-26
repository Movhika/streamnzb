package api

import (
	"encoding/json"
	"net/http"

	"streamnzb/pkg/auth"
	"streamnzb/pkg/core/config"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Success            bool   `json:"success"`
	Token              string `json:"token,omitempty"`
	User               string `json:"user,omitempty"`
	MustChangePassword bool   `json:"must_change_password,omitempty"`
	Error              string `json:"error,omitempty"`
}

// handleLogin handles user login
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	adminUsername := s.config.GetAdminUsername()
	if req.Username != adminUsername {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(LoginResponse{
			Success: false,
			Error:   "Invalid credentials",
		})
		return
	}

	device, err := s.deviceManager.Authenticate(req.Username, req.Password, adminUsername, s.config.AdminPasswordHash, s.config.AdminToken)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(LoginResponse{
			Success: false,
			Error:   "Invalid credentials",
		})
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_session",
		Value:    device.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400 * 7, // 7 days
	})

	var mustChangePassword bool
	if device.Username == s.config.GetAdminUsername() {
		mustChangePassword = s.config.AdminMustChangePassword
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoginResponse{
		Success:            true,
		Token:              device.Token, // Empty for admin
		User:               device.Username,
		MustChangePassword: mustChangePassword,
	})
}

// handleInfo returns app info (version) - public, no auth
func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	version := "dev"
	if s.strmServer != nil {
		version = s.strmServer.Version()
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"version": version})
}

// handleAuthCheck checks if user is authenticated
func (s *Server) handleAuthCheck(w http.ResponseWriter, r *http.Request) {
	device, ok := auth.DeviceFromContext(r)
	if !ok {
		// Try cookie
		cookie, err := r.Cookie("auth_session")
		if err == nil && cookie != nil {
			device, err = s.deviceManager.AuthenticateToken(cookie.Value, s.config.GetAdminUsername(), s.config.AdminToken)
			if err == nil {
				ok = true
			}
		}
	}

	if ok {
		var mustChangePassword bool
		if device.Username == s.config.GetAdminUsername() {
			mustChangePassword = s.config.AdminMustChangePassword
		}
		out := map[string]interface{}{
			"authenticated":        true,
			"username":             device.Username,
			"must_change_password": mustChangePassword,
		}
		if s.strmServer != nil {
			out["version"] = s.strmServer.Version()
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(out)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": false,
		})
	}
}

// handleChangePassword updates admin password (POST /api/auth/change-password). Admin only.
func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	device, _ := auth.DeviceFromContext(r)
	if device == nil || device.Username != s.config.GetAdminUsername() {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}
	newHash := auth.HashPassword(req.Password)
	s.mu.Lock()
	s.config.AdminPasswordHash = newHash
	s.config.AdminMustChangePassword = false
	s.mu.Unlock()
	if err := s.config.Save(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Password updated successfully",
	})
}

// hasCustomFilters checks if user has custom filter configuration (Include/Avoid).
func hasCustomFilters(filters config.FilterConfig) bool {
	return len(filters.AudioInclude) > 0 || len(filters.AudioAvoid) > 0 ||
		len(filters.BitDepthInclude) > 0 || len(filters.BitDepthAvoid) > 0 ||
		len(filters.ChannelsInclude) > 0 || len(filters.ChannelsAvoid) > 0 ||
		len(filters.CodecInclude) > 0 || len(filters.CodecAvoid) > 0 ||
		len(filters.ContainerInclude) > 0 || len(filters.ContainerAvoid) > 0 ||
		len(filters.EditionInclude) > 0 || len(filters.EditionAvoid) > 0 ||
		len(filters.HDRInclude) > 0 || len(filters.HDRAvoid) > 0 ||
		len(filters.LanguagesInclude) > 0 || len(filters.LanguagesAvoid) > 0 ||
		len(filters.NetworkInclude) > 0 || len(filters.NetworkAvoid) > 0 ||
		len(filters.QualityInclude) > 0 || len(filters.QualityAvoid) > 0 ||
		len(filters.RegionInclude) > 0 || len(filters.RegionAvoid) > 0 ||
		len(filters.ResolutionInclude) > 0 || len(filters.ResolutionAvoid) > 0 ||
		len(filters.ThreeDInclude) > 0 || len(filters.ThreeDAvoid) > 0 ||
		len(filters.GroupInclude) > 0 || len(filters.GroupAvoid) > 0 ||
		filters.DubbedAvoid != nil || filters.HardcodedAvoid != nil ||
		filters.ProperInclude != nil || filters.RepackInclude != nil || filters.RepackAvoid != nil ||
		filters.ExtendedInclude != nil || filters.UnratedInclude != nil ||
		filters.MinSizeGB > 0 || filters.MaxSizeGB > 0 ||
		filters.MinYear > 0 || filters.MaxYear > 0
}

// hasCustomSorting checks if user has custom sorting (order lists or weights).
func hasCustomSorting(sorting config.SortConfig) bool {
	return len(sorting.ResolutionOrder) > 0 || len(sorting.CodecOrder) > 0 ||
		len(sorting.AudioOrder) > 0 || len(sorting.QualityOrder) > 0 ||
		len(sorting.VisualTagOrder) > 0 || len(sorting.ChannelsOrder) > 0 ||
		len(sorting.BitDepthOrder) > 0 || len(sorting.ContainerOrder) > 0 ||
		len(sorting.LanguagesOrder) > 0 || len(sorting.GroupOrder) > 0 ||
		len(sorting.EditionOrder) > 0 || len(sorting.NetworkOrder) > 0 ||
		len(sorting.RegionOrder) > 0 || len(sorting.ThreeDOrder) > 0 ||
		sorting.GrabWeight != 0 || sorting.AgeWeight != 0
}

// REST endpoint removed - config saving now uses WebSocket
