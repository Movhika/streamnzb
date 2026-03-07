package stremio

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"streamnzb/pkg/session"
)

type testReadResult struct {
	data []byte
	err  error
}

type testReadSeekCloser struct {
	reads []testReadResult
	idx   int
}

func (t *testReadSeekCloser) Read(p []byte) (int, error) {
	if t.idx >= len(t.reads) {
		return 0, io.EOF
	}
	res := t.reads[t.idx]
	t.idx++
	n := copy(p, res.data)
	return n, res.err
}

func (t *testReadSeekCloser) Seek(offset int64, whence int) (int64, error) {
	return 0, nil
}

func (t *testReadSeekCloser) Close() error {
	return nil
}

type bytesReadSeekCloser struct {
	*bytes.Reader
}

func (b *bytesReadSeekCloser) Close() error {
	return nil
}

func TestBufferedResponseWriterSnapshotTracksStatusAndBytes(t *testing.T) {
	recorder := httptest.NewRecorder()
	bw := newBufferedResponseWriter(recorder, 8)

	bw.WriteHeader(http.StatusPartialContent)
	if _, err := bw.Write([]byte("abc")); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	bw.Flush()

	snap := bw.Snapshot()
	if snap.StatusCode != http.StatusPartialContent {
		t.Fatalf("expected status %d, got %d", http.StatusPartialContent, snap.StatusCode)
	}
	if !snap.WroteHeader {
		t.Fatal("expected wroteHeader to be true")
	}
	if snap.BytesWritten != 3 {
		t.Fatalf("expected 3 bytes written, got %d", snap.BytesWritten)
	}
	if snap.WriteCalls != 1 {
		t.Fatalf("expected 1 write call, got %d", snap.WriteCalls)
	}
	if snap.FlushCalls != 1 {
		t.Fatalf("expected 1 flush call, got %d", snap.FlushCalls)
	}
	if snap.FlushError != "" {
		t.Fatalf("expected empty flush error, got %q", snap.FlushError)
	}
}

func TestStreamMonitorSnapshotTracksEOFWithoutErrorString(t *testing.T) {
	monitor := &StreamMonitor{
		ReadSeekCloser: &bytesReadSeekCloser{Reader: bytes.NewReader([]byte("abc"))},
		manager:        &session.Manager{},
		lastUpdate:     time.Now(),
	}

	buf := make([]byte, 4)
	if n, err := monitor.Read(buf); err != nil || n != 3 {
		t.Fatalf("first Read = (%d, %v), want (3, nil)", n, err)
	}
	if n, err := monitor.Read(buf); !errors.Is(err, io.EOF) || n != 0 {
		t.Fatalf("second Read = (%d, %v), want (0, EOF)", n, err)
	}

	snap := monitor.Snapshot()
	if snap.BytesRead != 3 {
		t.Fatalf("expected 3 bytes read, got %d", snap.BytesRead)
	}
	if snap.ReadCalls != 2 {
		t.Fatalf("expected 2 read calls, got %d", snap.ReadCalls)
	}
	if !snap.SawEOF {
		t.Fatal("expected EOF to be recorded")
	}
	if snap.LastReadError != "" {
		t.Fatalf("expected empty last read error, got %q", snap.LastReadError)
	}
}

func TestStreamMonitorSnapshotTracksReadErrorAndReadErrorOnce(t *testing.T) {
	boom := errors.New("boom")
	callbackCalls := 0
	monitor := &StreamMonitor{
		ReadSeekCloser: &testReadSeekCloser{reads: []testReadResult{{data: []byte("ab"), err: boom}, {err: boom}}},
		manager:        &session.Manager{},
		lastUpdate:     time.Now(),
		onReadError: func(_ string, err error) {
			callbackCalls++
			if !errors.Is(err, boom) {
				t.Fatalf("callback error = %v, want boom", err)
			}
		},
	}

	buf := make([]byte, 4)
	if n, err := monitor.Read(buf); !errors.Is(err, boom) || n != 2 {
		t.Fatalf("first Read = (%d, %v), want (2, boom)", n, err)
	}
	if _, err := monitor.Read(buf); !errors.Is(err, boom) {
		t.Fatalf("second Read error = %v, want boom", err)
	}

	snap := monitor.Snapshot()
	if snap.BytesRead != 2 {
		t.Fatalf("expected 2 bytes read, got %d", snap.BytesRead)
	}
	if snap.ReadCalls != 2 {
		t.Fatalf("expected 2 read calls, got %d", snap.ReadCalls)
	}
	if snap.SawEOF {
		t.Fatal("did not expect EOF to be recorded")
	}
	if snap.LastReadError != boom.Error() {
		t.Fatalf("expected last read error %q, got %q", boom.Error(), snap.LastReadError)
	}
	if callbackCalls != 1 {
		t.Fatalf("expected onReadError to be called once, got %d", callbackCalls)
	}
}