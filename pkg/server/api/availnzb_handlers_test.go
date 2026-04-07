package api

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"streamnzb/pkg/auth"
	"streamnzb/pkg/core/config"
	"streamnzb/pkg/core/logger"
	"streamnzb/pkg/core/persistence"
)

func init() {
	logger.Log = slog.New(slog.NewTextHandler(io.Discard, nil))
}

var (
	availNZBStatusTestStateOnce sync.Once
	availNZBStatusTestStateMgr  *persistence.StateManager
	availNZBStatusTestStateErr  error
)

func testAvailNZBStatusStateManager(t *testing.T) *persistence.StateManager {
	t.Helper()

	availNZBStatusTestStateOnce.Do(func() {
		tempDir, err := os.MkdirTemp("", "streamnzb-api-availnzb-status-")
		if err != nil {
			availNZBStatusTestStateErr = err
			return
		}
		availNZBStatusTestStateMgr, availNZBStatusTestStateErr = persistence.GetManager(tempDir)
	})
	if availNZBStatusTestStateErr != nil {
		t.Fatalf("GetManager: %v", availNZBStatusTestStateErr)
	}
	return availNZBStatusTestStateMgr
}

func TestHandleAvailNZBStatusReturnsStatusForAdmin(t *testing.T) {
	const apiKey = "secret-key"
	stateMgr := testAvailNZBStatusStateManager(t)
	if err := stateMgr.Set("availnzb_api_key", map[string]string{
		"token":           apiKey,
		"recovery_secret": "recover-1",
	}); err != nil {
		t.Fatalf("Set recovery secret: %v", err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/me" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/api/v1/me")
		}
		if got := r.Header.Get("X-API-Key"); got != apiKey {
			t.Fatalf("X-API-Key = %q, want %q", got, apiKey)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"app-1","name":"StreamNZB","is_active":true,"app_source":"self_service","trust_level":"trusted","trust_score":98.5,"report_count":12,"public_report_count":3,"verified_report_count":9,"quarantined_report_count":1,"rolled_back_report_count":0,"last_report_at":"2026-03-09T12:00:00Z","last_rollback_at":null}`))
	}))
	defer upstream.Close()

	s := &Server{
		config:         &config.Config{AdminUsername: "admin"},
		availNZBURL:    upstream.URL,
		availNZBAPIKey: apiKey,
		attemptLister:  stateMgr,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/availnzb/status", nil)
	req = req.WithContext(auth.ContextWithStream(req.Context(), &auth.Stream{Username: "admin"}))
	rr := httptest.NewRecorder()

	s.handleAvailNZBStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var got availNZBStatusResponse
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.RecoverySecret != "recover-1" {
		t.Fatalf("RecoverySecret = %q, want %q", got.RecoverySecret, "recover-1")
	}
	if got.Status == nil {
		t.Fatal("expected status payload")
	}
	if got.Status.ID != "app-1" || got.Status.Name != "StreamNZB" || !got.Status.IsActive {
		t.Fatalf("unexpected payload: %+v", got)
	}
	if got.Status.LastReportAt == nil || !got.Status.LastReportAt.Equal(time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("LastReportAt = %v, want 2026-03-09T12:00:00Z", got.Status.LastReportAt)
	}
}

func TestHandleAvailNZBStatusReturnsRecoverySecretWhenStatusFetchFails(t *testing.T) {
	const apiKey = "secret-key"
	stateMgr := testAvailNZBStatusStateManager(t)
	if err := stateMgr.Set("availnzb_api_key", map[string]string{
		"token":           apiKey,
		"recovery_secret": "recover-2",
	}); err != nil {
		t.Fatalf("Set recovery secret: %v", err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"upstream unavailable"}`))
	}))
	defer upstream.Close()

	s := &Server{
		config:         &config.Config{AdminUsername: "admin"},
		availNZBURL:    upstream.URL,
		availNZBAPIKey: apiKey,
		attemptLister:  stateMgr,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/availnzb/status", nil)
	req = req.WithContext(auth.ContextWithStream(req.Context(), &auth.Stream{Username: "admin"}))
	rr := httptest.NewRecorder()

	s.handleAvailNZBStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var got availNZBStatusResponse
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.RecoverySecret != "recover-2" {
		t.Fatalf("RecoverySecret = %q, want %q", got.RecoverySecret, "recover-2")
	}
	if got.Status != nil {
		t.Fatalf("expected nil status payload, got %+v", got.Status)
	}
	if got.StatusError == "" {
		t.Fatal("expected status error")
	}
}

func TestHandleAvailNZBStatusRejectsNonAdmin(t *testing.T) {
	s := &Server{config: &config.Config{AdminUsername: "admin"}}
	req := httptest.NewRequest(http.MethodGet, "/api/availnzb/status", nil)
	req = req.WithContext(auth.ContextWithStream(req.Context(), &auth.Stream{Username: "user"}))
	rr := httptest.NewRecorder()

	s.handleAvailNZBStatus(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestHandleAvailNZBStatusRequiresAPIKey(t *testing.T) {
	s := &Server{
		config:      &config.Config{AdminUsername: "admin"},
		availNZBURL: "https://snzb.stream",
	}
	req := httptest.NewRequest(http.MethodGet, "/api/availnzb/status", nil)
	req = req.WithContext(auth.ContextWithStream(req.Context(), &auth.Stream{Username: "admin"}))
	rr := httptest.NewRecorder()

	s.handleAvailNZBStatus(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if body["error"] != "AvailNZB API key is not configured" {
		t.Fatalf("error = %q", body["error"])
	}
}
