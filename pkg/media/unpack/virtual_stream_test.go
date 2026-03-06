package unpack

import (
	"context"
	"testing"
)

func TestVirtualStreamLiveCount(t *testing.T) {
	before := LiveVirtualStreams()
	stream := NewVirtualStream(context.Background(), nil, 0, 0)
	if got := LiveVirtualStreams(); got != before+1 {
		_ = stream.Close()
		t.Fatalf("expected %d live virtual streams, got %d", before+1, got)
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if got := LiveVirtualStreams(); got != before {
		t.Fatalf("expected %d live virtual streams after close, got %d", before, got)
	}
}
