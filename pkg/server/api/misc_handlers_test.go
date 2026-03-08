package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"streamnzb/pkg/core/logger"
)

func TestHandleDownloadLogsServesCurrentLogFile(t *testing.T) {
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir temp dir: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()

	const content = "time=2026-03-08T00:00:00.000+00:00 level=INFO msg=\"hello\"\n"
	if err := os.WriteFile(logger.GetCurrentLogPath(), []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/logs/download", nil)
	rr := httptest.NewRecorder()

	(&Server{}).handleDownloadLogs(rr, req)

	res := rr.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}
	if got := res.Header.Get("Content-Disposition"); got != `attachment; filename="streamnzb.log"` {
		t.Fatalf("Content-Disposition = %q", got)
	}
	if got := res.Header.Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(body) != content {
		t.Fatalf("body = %q, want %q", string(body), content)
	}
}

func TestHandleDownloadLogsReturnsNotFoundWhenMissing(t *testing.T) {
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir temp dir: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()

	req := httptest.NewRequest(http.MethodGet, "/api/logs/download", nil)
	rr := httptest.NewRecorder()

	(&Server{}).handleDownloadLogs(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestHandleDownloadLogsRejectsNonGet(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/logs/download", nil)
	rr := httptest.NewRecorder()

	(&Server{}).handleDownloadLogs(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}
