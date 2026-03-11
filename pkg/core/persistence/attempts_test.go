package persistence

import (
	"os"
	"testing"
	"time"
)

func newTestStateManager(t *testing.T) *StateManager {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "persistence-attempts-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		globalManager = nil
		_ = os.RemoveAll(tempDir)
	})
	globalManager = nil
	mgr, err := GetManager(tempDir)
	if err != nil {
		t.Fatalf("failed to get manager: %v", err)
	}
	return mgr
}

func TestRecordPreloadAttemptUsesSharedWriteLock(t *testing.T) {
	mgr := newTestStateManager(t)
	mgr.mu.Lock()
	done := make(chan struct{})
	go func() {
		defer close(done)
		mgr.RecordPreloadAttempt(RecordAttemptParams{SlotPath: "slot-lock", ReleaseTitle: "Locked release"})
	}()

	select {
	case <-done:
		t.Fatal("RecordPreloadAttempt should wait for the shared write lock")
	case <-time.After(50 * time.Millisecond):
	}

	mgr.mu.Unlock()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("RecordPreloadAttempt did not complete after releasing the shared write lock")
	}

	list, err := mgr.ListAttempts(ListAttemptsOptions{Limit: 10})
	if err != nil {
		t.Fatalf("ListAttempts: %v", err)
	}
	if len(list) != 1 || !list[0].Preload {
		t.Fatalf("expected one preload attempt after lock release, got %+v", list)
	}
}

func TestRecordAttemptUsesSharedWriteLock(t *testing.T) {
	mgr := newTestStateManager(t)
	mgr.RecordPreloadAttempt(RecordAttemptParams{SlotPath: "slot-final", ReleaseTitle: "Preloaded release"})

	mgr.mu.Lock()
	done := make(chan struct{})
	go func() {
		defer close(done)
		mgr.RecordAttempt(RecordAttemptParams{SlotPath: "slot-final", ReleaseTitle: "Preloaded release", Success: true})
	}()

	select {
	case <-done:
		t.Fatal("RecordAttempt should wait for the shared write lock")
	case <-time.After(50 * time.Millisecond):
	}

	mgr.mu.Unlock()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("RecordAttempt did not complete after releasing the shared write lock")
	}

	list, err := mgr.ListAttempts(ListAttemptsOptions{Limit: 10})
	if err != nil {
		t.Fatalf("ListAttempts: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected one resolved attempt, got %d", len(list))
	}
	if list[0].Preload {
		t.Fatal("expected preload row to be resolved")
	}
	if !list[0].Success {
		t.Fatal("expected resolved attempt to be marked successful")
	}
}
