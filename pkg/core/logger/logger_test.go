package logger

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestSanitizeStringRedactsSensitiveData(t *testing.T) {
	token := strings.Repeat("a", 64)
	input := `https://user:pass@example.com/` + token + `/play/1?api_key=secret&foo=ok Authorization=Bearer topsecret auth_session=session123 password=hunter2`
	got := sanitizeString(input)
	for _, secret := range []string{"user:pass", token, "secret", "topsecret", "session123", "hunter2"} {
		if strings.Contains(got, secret) {
			t.Fatalf("sanitizeString leaked %q in %q", secret, got)
		}
	}
	if !strings.Contains(got, redactedValue) {
		t.Fatalf("sanitizeString did not redact anything: %q", got)
	}
}

func TestSanitizeAttrRedactsSensitiveKeysAndStringValues(t *testing.T) {
	if got := sanitizeAttr(slog.String("api_key", "secret")).Value.String(); got != redactedValue {
		t.Fatalf("api_key = %q, want %q", got, redactedValue)
	}
	if got := sanitizeAttr(slog.String("base_url", "https://example.com/addon")).Value.String(); got != redactedValue {
		t.Fatalf("base_url = %q, want %q", got, redactedValue)
	}
	urlAttr := sanitizeAttr(slog.String("url", "https://example.com/search?token=secret"))
	if strings.Contains(urlAttr.Value.String(), "secret") {
		t.Fatalf("url attr leaked secret: %q", urlAttr.Value.String())
	}
	group := sanitizeAttr(slog.Group("headers", slog.String("Authorization", "Bearer secret")))
	if got := group.Value.Group()[0].Value.String(); got != redactedValue {
		t.Fatalf("group authorization = %q, want %q", got, redactedValue)
	}
}

func TestGlobalBroadcastHandlerRedactsUnderlyingOutputAndHistory(t *testing.T) {
	historyMu.Lock()
	history = nil
	historyMu.Unlock()
	broadcastCh = nil
	logFileMu.Lock()
	logFile = nil
	logFileMu.Unlock()
	locationMu.Lock()
	logLocation = time.UTC
	locationMu.Unlock()

	var buf bytes.Buffer
	h := &GlobalBroadcastHandler{Handler: slog.NewTextHandler(&buf, nil)}
	r := slog.NewRecord(time.Unix(0, 0), slog.LevelInfo, "fetch https://example.com?api_key=secret", 0)
	r.AddAttrs(
		slog.String("Authorization", "Bearer supersecret"),
		slog.String("url", "https://user:pass@example.com/"+strings.Repeat("b", 64)+"/play/1?token=abc"),
	)
	if err := h.Handle(context.Background(), r); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	output := buf.String()
	historyEntries := GetHistory()
	if len(historyEntries) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(historyEntries))
	}
	for _, text := range []string{output, historyEntries[0]} {
		for _, secret := range []string{"secret", "supersecret", "user:pass", strings.Repeat("b", 64), "token=abc"} {
			if strings.Contains(text, secret) {
				t.Fatalf("log output leaked %q in %q", secret, text)
			}
		}
		if !strings.Contains(text, redactedValue) {
			t.Fatalf("expected redacted output, got %q", text)
		}
	}
}