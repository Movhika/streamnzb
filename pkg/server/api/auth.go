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

// hasCustomFilters checks if user has custom filter configuration (Included/Required/Excluded).
func hasCustomFilters(filters config.FilterConfig) bool {
	return len(filters.AudioIncluded) > 0 || len(filters.AudioRequired) > 0 || len(filters.AudioExcluded) > 0 ||
		len(filters.BitDepthIncluded) > 0 || len(filters.BitDepthRequired) > 0 || len(filters.BitDepthExcluded) > 0 ||
		len(filters.ChannelsIncluded) > 0 || len(filters.ChannelsRequired) > 0 || len(filters.ChannelsExcluded) > 0 ||
		len(filters.CodecIncluded) > 0 || len(filters.CodecRequired) > 0 || len(filters.CodecExcluded) > 0 ||
		len(filters.ContainerIncluded) > 0 || len(filters.ContainerRequired) > 0 || len(filters.ContainerExcluded) > 0 ||
		len(filters.EditionIncluded) > 0 || len(filters.EditionRequired) > 0 || len(filters.EditionExcluded) > 0 ||
		len(filters.HDRIncluded) > 0 || len(filters.HDRRequired) > 0 || len(filters.HDRExcluded) > 0 ||
		len(filters.LanguagesIncluded) > 0 || len(filters.LanguagesRequired) > 0 || len(filters.LanguagesExcluded) > 0 ||
		len(filters.NetworkIncluded) > 0 || len(filters.NetworkRequired) > 0 || len(filters.NetworkExcluded) > 0 ||
		len(filters.QualityIncluded) > 0 || len(filters.QualityRequired) > 0 || len(filters.QualityExcluded) > 0 ||
		len(filters.RegionIncluded) > 0 || len(filters.RegionRequired) > 0 || len(filters.RegionExcluded) > 0 ||
		len(filters.ResolutionIncluded) > 0 || len(filters.ResolutionRequired) > 0 || len(filters.ResolutionExcluded) > 0 ||
		len(filters.ThreeDIncluded) > 0 || len(filters.ThreeDRequired) > 0 || len(filters.ThreeDExcluded) > 0 ||
		len(filters.GroupIncluded) > 0 || len(filters.GroupRequired) > 0 || len(filters.GroupExcluded) > 0 ||
		filters.DubbedExcluded != nil || filters.HardcodedExcluded != nil ||
		filters.ProperRequired != nil || filters.RepackRequired != nil || filters.RepackExcluded != nil ||
		filters.ExtendedRequired != nil || filters.UnratedRequired != nil ||
		filters.MinSizeGB > 0 || filters.MaxSizeGB > 0 ||
		filters.MinYear > 0 || filters.MaxYear > 0 ||
		filters.MinAgeHours > 0 || filters.MaxAgeHours > 0 ||
		len(filters.KeywordsExcluded) > 0 || len(filters.KeywordsRequired) > 0 ||
		len(filters.RegexExcluded) > 0 || len(filters.RegexRequired) > 0 ||
		filters.AvailNZBRequired != nil ||
		len(filters.SizePerResolution) > 0 ||
		filters.MinBitrateKbps > 0 || filters.MaxBitrateKbps > 0
}

// hasCustomSorting checks if user has custom sorting (preferred lists, weights, or custom scoring).
func hasCustomSorting(sorting config.SortConfig) bool {
	return len(sorting.PreferredResolution) > 0 || len(sorting.PreferredCodec) > 0 ||
		len(sorting.PreferredAudio) > 0 || len(sorting.PreferredQuality) > 0 ||
		len(sorting.PreferredVisualTag) > 0 || len(sorting.PreferredChannels) > 0 ||
		len(sorting.PreferredBitDepth) > 0 || len(sorting.PreferredContainer) > 0 ||
		len(sorting.PreferredLanguages) > 0 || len(sorting.PreferredGroup) > 0 ||
		len(sorting.PreferredEdition) > 0 || len(sorting.PreferredNetwork) > 0 ||
		len(sorting.PreferredRegion) > 0 || len(sorting.PreferredThreeD) > 0 ||
		sorting.GrabWeight != 0 || sorting.AgeWeight != 0 ||
		len(sorting.KeywordsPreferred) > 0 || sorting.KeywordsWeight != 0 ||
		len(sorting.RegexPreferred) > 0 || sorting.RegexWeight != 0 ||
		sorting.AvailNZBWeight != 0 ||
		(sorting.UseCustomScoring != nil && *sorting.UseCustomScoring) ||
		len(sorting.GroupOrderTier1) > 0 || len(sorting.GroupOrderTier2) > 0 || len(sorting.GroupOrderTier3) > 0
}

// REST endpoint removed - config saving now uses WebSocket
