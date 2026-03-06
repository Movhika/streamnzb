package loader

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"streamnzb/pkg/core/logger"
	"streamnzb/pkg/media/nzb"
)

func TestSegmentReaderLiveCount(t *testing.T) {
	oldLogger := logger.Log
	logger.Log = slog.New(slog.NewTextHandler(io.Discard, nil))
	defer func() {
		logger.Log = oldLogger
	}()

	before := LiveSegmentReaders()
	f := NewFile(context.Background(), &nzb.File{Subject: "test.mkv"}, nil, nil, nil, nil)
	r := NewSegmentReader(context.Background(), f, 0)
	if got := LiveSegmentReaders(); got != before+1 {
		_ = r.Close()
		t.Fatalf("expected %d live segment readers, got %d", before+1, got)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if got := LiveSegmentReaders(); got != before {
		t.Fatalf("expected %d live segment readers after close, got %d", before, got)
	}
}

func TestSegmentReaderLiveDetailsIncludeOwner(t *testing.T) {
	oldLogger := logger.Log
	logger.Log = slog.New(slog.NewTextHandler(io.Discard, nil))
	defer func() {
		logger.Log = oldLogger
	}()

	f := NewFile(context.Background(), &nzb.File{Subject: "test.mkv"}, nil, nil, nil, nil)
	f.SetOwnerSessionID("sess-42")
	r := NewSegmentReader(context.Background(), f, 0)
	defer func() { _ = r.Close() }()

	found := false
	for _, detail := range LiveSegmentReaderDetails() {
		if strings.Contains(detail, "session=sess-42") && strings.Contains(detail, `file="test.mkv"`) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected live reader details to include owner session and file, got %v", LiveSegmentReaderDetails())
	}

	if err := r.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	for _, detail := range LiveSegmentReaderDetails() {
		if strings.Contains(detail, "session=sess-42") {
			t.Fatalf("expected closed reader detail to be removed, got %v", LiveSegmentReaderDetails())
		}
	}
}
