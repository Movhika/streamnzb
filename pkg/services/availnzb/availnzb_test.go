package availnzb

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"streamnzb/pkg/core/logger"
)

func init() {
	logger.Log = slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestGetMeIncludesAPIKeyAndDecodesResponse(t *testing.T) {
	t.Parallel()

	const apiKey = "secret-key"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %q, want %q", r.Method, http.MethodGet)
		}
		if r.URL.Path != "/api/v1/me" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/api/v1/me")
		}
		if got := r.Header.Get("X-API-Key"); got != apiKey {
			t.Fatalf("X-API-Key = %q, want %q", got, apiKey)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"app-1","name":"StreamNZB","is_active":true,"app_source":"self_service","trust_level":"trusted","trust_score":97.5,"report_count":7,"public_report_count":2,"verified_report_count":5,"quarantined_report_count":1,"rolled_back_report_count":0,"last_report_at":"2026-03-09T12:34:56Z","last_rollback_at":null}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, apiKey)
	client.HTTP = server.Client()

	got, err := client.GetMe()
	if err != nil {
		t.Fatalf("GetMe: %v", err)
	}
	if got == nil {
		t.Fatal("GetMe returned nil response")
	}
	if got.ID != "app-1" || got.Name != "StreamNZB" || got.TrustLevel != "trusted" {
		t.Fatalf("unexpected response: %+v", got)
	}
	if got.LastReportAt == nil || !got.LastReportAt.Equal(time.Date(2026, 3, 9, 12, 34, 56, 0, time.UTC)) {
		t.Fatalf("LastReportAt = %v, want 2026-03-09T12:34:56Z", got.LastReportAt)
	}
	if got.LastRollbackAt != nil {
		t.Fatalf("LastRollbackAt = %v, want nil", got.LastRollbackAt)
	}
}

func TestGetMeDecodesNumericID(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":123,"name":"StreamNZB","is_active":true,"app_source":"self_service","trust_level":"trusted","trust_score":97.5,"report_count":7,"public_report_count":2,"verified_report_count":5,"quarantined_report_count":1,"rolled_back_report_count":0,"last_report_at":null,"last_rollback_at":null}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret-key")
	client.HTTP = server.Client()

	got, err := client.GetMe()
	if err != nil {
		t.Fatalf("GetMe: %v", err)
	}
	if got == nil {
		t.Fatal("GetMe returned nil response")
	}
	if got.ID != "123" {
		t.Fatalf("ID = %q, want %q", got.ID, "123")
	}
}

func TestGetMeDecodesNumericTrustLevel(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"app-1","name":"StreamNZB","is_active":true,"app_source":"self_service","trust_level":4,"trust_score":97.5,"report_count":7,"public_report_count":2,"verified_report_count":5,"quarantined_report_count":1,"rolled_back_report_count":0,"last_report_at":null,"last_rollback_at":null}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret-key")
	client.HTTP = server.Client()

	got, err := client.GetMe()
	if err != nil {
		t.Fatalf("GetMe: %v", err)
	}
	if got == nil {
		t.Fatal("GetMe returned nil response")
	}
	if got.TrustLevel != "4" {
		t.Fatalf("TrustLevel = %q, want %q", got.TrustLevel, "4")
	}
}

func TestGetMeDecodesLegacyTimestampFormat(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"app-1","name":"StreamNZB","is_active":true,"app_source":"self_service","trust_level":"trusted","trust_score":97.5,"report_count":7,"public_report_count":2,"verified_report_count":5,"quarantined_report_count":1,"rolled_back_report_count":0,"last_report_at":"2026-03-09 08:33:39","last_rollback_at":"2026-03-09 09:10:11"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret-key")
	client.HTTP = server.Client()

	got, err := client.GetMe()
	if err != nil {
		t.Fatalf("GetMe: %v", err)
	}
	if got == nil {
		t.Fatal("GetMe returned nil response")
	}
	if got.LastReportAt == nil || !got.LastReportAt.Equal(time.Date(2026, 3, 9, 8, 33, 39, 0, time.UTC)) {
		t.Fatalf("LastReportAt = %v, want 2026-03-09 08:33:39 UTC", got.LastReportAt)
	}
	if got.LastRollbackAt == nil || !got.LastRollbackAt.Equal(time.Date(2026, 3, 9, 9, 10, 11, 0, time.UTC)) {
		t.Fatalf("LastRollbackAt = %v, want 2026-03-09 09:10:11 UTC", got.LastRollbackAt)
	}
}

func TestGetMeReturnsErrorOnUnexpectedStatus(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret-key")
	client.HTTP = server.Client()

	got, err := client.GetMe()
	if err == nil {
		t.Fatal("GetMe error = nil, want error")
	}
	if got != nil {
		t.Fatalf("GetMe response = %+v, want nil", got)
	}
}

type stubKeyStore struct {
	raw      map[string][]byte
	setCalls int
	setErr   error
}

func (s *stubKeyStore) Get(key string, target interface{}) (bool, error) {
	raw, ok := s.raw[key]
	if !ok {
		return false, nil
	}
	return true, json.Unmarshal(raw, target)
}

func (s *stubKeyStore) Set(key string, value interface{}) error {
	if s.setErr != nil {
		return s.setErr
	}
	if s.raw == nil {
		s.raw = make(map[string][]byte)
	}
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	s.raw[key] = b
	s.setCalls++
	return nil
}

func TestRegisterKeyPostsAppNameAndDecodesResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want %q", r.Method, http.MethodPost)
		}
		if r.URL.Path != "/api/v1/keys" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/api/v1/keys")
		}
		var req KeyCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Decode request: %v", err)
		}
		if req.Name != DefaultAppName {
			t.Fatalf("name = %q, want %q", req.Name, DefaultAppName)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"app-1","name":"StreamNZB","token":"new-token","recovery_secret":"recover-1","created_at":"2026-03-09T12:34:56Z"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	client.HTTP = server.Client()

	got, err := client.RegisterKey(DefaultAppName)
	if err != nil {
		t.Fatalf("RegisterKey: %v", err)
	}
	if got == nil {
		t.Fatal("RegisterKey returned nil response")
	}
	if got.Token != "new-token" || got.RecoverySecret != "recover-1" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestRegisterKeyDecodesNumericID(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":456,"name":"StreamNZB","token":"new-token","recovery_secret":"recover-1","created_at":"2026-03-09T12:34:56Z"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	client.HTTP = server.Client()

	got, err := client.RegisterKey(DefaultAppName)
	if err != nil {
		t.Fatalf("RegisterKey: %v", err)
	}
	if got == nil {
		t.Fatal("RegisterKey returned nil response")
	}
	if got.ID != "456" {
		t.Fatalf("ID = %q, want %q", got.ID, "456")
	}
}

func TestResolveAPIKeyReturnsExplicitOverride(t *testing.T) {
	t.Parallel()

	got, err := ResolveAPIKey(nil, "https://snzb.stream", "configured-key", DefaultAppName)
	if err != nil {
		t.Fatalf("ResolveAPIKey: %v", err)
	}
	if got != "configured-key" {
		t.Fatalf("ResolveAPIKey() = %q, want %q", got, "configured-key")
	}
}

func TestResolveAPIKeyReturnsStoredLegacyKey(t *testing.T) {
	t.Parallel()

	store := &stubKeyStore{raw: map[string][]byte{apiKeyStateKey: []byte(`"stored-token"`)}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("unexpected registration request")
	}))
	defer server.Close()

	got, err := ResolveAPIKey(store, server.URL, "", DefaultAppName)
	if err != nil {
		t.Fatalf("ResolveAPIKey: %v", err)
	}
	if got != "stored-token" {
		t.Fatalf("ResolveAPIKey() = %q, want %q", got, "stored-token")
	}
	if store.setCalls != 0 {
		t.Fatalf("setCalls = %d, want 0", store.setCalls)
	}
}

func TestResolveAPIKeyRegistersAndPersistsWhenMissing(t *testing.T) {
	t.Parallel()

	store := &stubKeyStore{raw: map[string][]byte{}}
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var req KeyCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Decode request: %v", err)
		}
		if req.Name != DefaultAppName {
			t.Fatalf("name = %q, want %q", req.Name, DefaultAppName)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":2,"name":"StreamNZB","token":"registered-token","recovery_secret":"recover-2","created_at":"2026-03-09T13:00:00Z"}`))
	}))
	defer server.Close()

	got, err := ResolveAPIKey(store, server.URL, "", DefaultAppName)
	if err != nil {
		t.Fatalf("ResolveAPIKey: %v", err)
	}
	if got != "registered-token" {
		t.Fatalf("ResolveAPIKey() = %q, want %q", got, "registered-token")
	}
	if callCount != 1 {
		t.Fatalf("register call count = %d, want 1", callCount)
	}

	var saved apiKeyState
	if found, err := store.Get(apiKeyStateKey, &saved); err != nil {
		t.Fatalf("Get persisted key: %v", err)
	} else if !found {
		t.Fatal("persisted key not found")
	}
	if saved.ID != "2" || saved.Token != "registered-token" || saved.RecoverySecret != "recover-2" {
		t.Fatalf("unexpected persisted state: %+v", saved)
	}
}

func TestLoadStoredRecoverySecretReturnsPersistedValue(t *testing.T) {
	t.Parallel()

	store := &stubKeyStore{raw: map[string][]byte{apiKeyStateKey: []byte(`{"token":"stored-token","recovery_secret":"recover-2"}`)}}

	got, err := LoadStoredRecoverySecret(store)
	if err != nil {
		t.Fatalf("LoadStoredRecoverySecret: %v", err)
	}
	if got != "recover-2" {
		t.Fatalf("LoadStoredRecoverySecret() = %q, want %q", got, "recover-2")
	}
}

func TestLoadStoredRecoverySecretReturnsEmptyForLegacyKeyState(t *testing.T) {
	t.Parallel()

	store := &stubKeyStore{raw: map[string][]byte{apiKeyStateKey: []byte(`"stored-token"`)}}

	got, err := LoadStoredRecoverySecret(store)
	if err != nil {
		t.Fatalf("LoadStoredRecoverySecret: %v", err)
	}
	if got != "" {
		t.Fatalf("LoadStoredRecoverySecret() = %q, want empty string", got)
	}
}
