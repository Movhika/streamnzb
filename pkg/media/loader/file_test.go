package loader

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"streamnzb/pkg/media/decode"
)

func TestShouldPersistDownloadedSegment(t *testing.T) {
	if !shouldPersistDownloadedSegment(nil) {
		t.Fatal("nil context should allow segment persistence")
	}
	if !shouldPersistDownloadedSegment(context.Background()) {
		t.Fatal("active context should allow segment persistence")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if shouldPersistDownloadedSegment(ctx) {
		t.Fatal("canceled context should block segment persistence")
	}
}

type trackingReadCloser struct {
	io.Reader
	closeCalls int
}

func (t *trackingReadCloser) Close() error {
	t.closeCalls++
	return nil
}

func TestDecodeAndCloseBodyClosesOnSuccess(t *testing.T) {
	body := &trackingReadCloser{Reader: bytes.NewBufferString("abc")}
	frame := &decode.Frame{Data: []byte("ok")}

	got, err := decodeAndCloseBody(body, func(r io.Reader) (*decode.Frame, error) {
		_, readErr := io.ReadAll(r)
		return frame, readErr
	})
	if err != nil {
		t.Fatalf("decodeAndCloseBody returned error: %v", err)
	}
	if got != frame {
		t.Fatal("decodeAndCloseBody returned unexpected frame")
	}
	if body.closeCalls != 1 {
		t.Fatalf("expected body to be closed once, got %d", body.closeCalls)
	}
}

func TestDecodeAndCloseBodyClosesOnError(t *testing.T) {
	body := &trackingReadCloser{Reader: bytes.NewBufferString("abc")}
	wantErr := errors.New("decode failed")

	got, err := decodeAndCloseBody(body, func(r io.Reader) (*decode.Frame, error) {
		_, _ = io.ReadAll(r)
		return nil, wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if got != nil {
		t.Fatal("expected nil frame on error")
	}
	if body.closeCalls != 1 {
		t.Fatalf("expected body to be closed once, got %d", body.closeCalls)
	}
}
