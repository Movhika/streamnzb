package stremio

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"streamnzb/pkg/core/config"
	"streamnzb/pkg/core/logger"
	"streamnzb/pkg/release"
	"streamnzb/pkg/search/triage"
	"streamnzb/pkg/session"
)

var testLoggerOnce sync.Once

func initFailoverTestLogger() {
	testLoggerOnce.Do(func() {
		logger.Init("ERROR")
	})
}

func TestSwitchToNextFallbackSkipsUnresolvableCandidate(t *testing.T) {
	initFailoverTestLogger()
	t.Parallel()

	manager := session.NewManager(nil, nil, time.Minute)
	t.Cleanup(manager.Shutdown)
	server := &Server{config: &config.Config{}, sessionManager: manager}
	key := StreamSlotKey{StreamID: "stream_test", ContentType: "movie", ID: "tt123"}
	currentID := key.SlotPath(0)
	skippedID := key.SlotPath(1)
	wantID := key.SlotPath(2)

	server.playlistCache.Store(key.CacheKey(), &playlistCacheEntry{
		result: &playlistResult{
			Candidates: []triage.Candidate{
				{Release: &release.Release{Link: "https://example.invalid/0"}},
				{},
				{Release: &release.Release{Link: "https://example.invalid/2"}},
			},
			Params: &SearchParams{ContentType: key.ContentType, ID: key.ID},
		},
		until: time.Now().Add(time.Minute),
	})

	nextSess, nextID, err := server.switchToNextFallback(context.Background(), &session.Session{ID: currentID}, nil)
	if err != nil {
		t.Fatalf("switchToNextFallback returned error: %v", err)
	}
	if nextID != wantID {
		t.Fatalf("nextID = %q, want %q", nextID, wantID)
	}
	if nextSess == nil || nextSess.ID != wantID {
		t.Fatalf("next session = %#v, want id %q", nextSess, wantID)
	}
	if !manager.GetSlotFailedDuringPlayback(skippedID) {
		t.Fatalf("expected skipped slot %q to be marked failed", skippedID)
	}
	if got, err := manager.GetSession(wantID); err != nil || got == nil {
		t.Fatalf("expected resolved fallback session %q, got (%v, %v)", wantID, got, err)
	}
	if manager.GetSlotFailedDuringPlayback(wantID) {
		t.Fatalf("did not expect resolved slot %q to be marked failed", wantID)
	}
}

func TestForceDisconnectRedirectsToErrorVideo(t *testing.T) {
	initFailoverTestLogger()
	t.Parallel()

	recorder := httptest.NewRecorder()
	forceDisconnect(recorder, "http://localhost:11470/")
	response := recorder.Result()

	if response.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusTemporaryRedirect)
	}
	if got := response.Header.Get("Location"); got != "http://localhost:11470/error/failure.mp4" {
		t.Fatalf("Location = %q, want %q", got, "http://localhost:11470/error/failure.mp4")
	}
	if got := response.Header.Get("Connection"); got != "close" {
		t.Fatalf("Connection = %q, want %q", got, "close")
	}
	if got := response.Header.Get("Cache-Control"); got != "no-cache, no-store, must-revalidate" {
		t.Fatalf("Cache-Control = %q, want %q", got, "no-cache, no-store, must-revalidate")
	}
}

func TestClassifyPlaybackStartupErrWrapsOwnTimeout(t *testing.T) {
	initFailoverTestLogger()
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	<-ctx.Done()

	err := classifyPlaybackStartupErr("probe", ctx, context.DeadlineExceeded)
	if !errors.Is(err, ErrPlaybackStartupTimeout) {
		t.Fatalf("expected startup timeout error, got %v", err)
	}
	if isPlayPrepareCancellation(err) {
		t.Fatalf("startup timeout should trigger failover, got cancellation classification for %v", err)
	}
}

func TestClassifyPlaybackStartupErrPreservesParentCancellation(t *testing.T) {
	initFailoverTestLogger()
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := classifyPlaybackStartupErr("open", ctx, context.Canceled)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
	if !isPlayPrepareCancellation(err) {
		t.Fatalf("expected canceled prepare error to stay classified as cancellation, got %v", err)
	}
}
