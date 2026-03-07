package loader

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"streamnzb/pkg/core/logger"
	"streamnzb/pkg/media/nzb"
	"streamnzb/pkg/usenet/pool"
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

func TestSegmentReaderSeekDoesNotWaitForCanceledPrefetch(t *testing.T) {
	oldLogger := logger.Log
	logger.Log = slog.New(slog.NewTextHandler(io.Discard, nil))
	defer func() {
		logger.Log = oldLogger
	}()

	fetcher := &blockingSegmentFetcher{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	f := NewFile(context.Background(), testNZBFile("seek-test.mkv", 4, 4, 4), nil, nil, fetcher, nil)
	r := NewSegmentReader(context.Background(), f, 0)

	select {
	case <-fetcher.started:
	case <-time.After(500 * time.Millisecond):
		close(fetcher.release)
		_ = r.Close()
		t.Fatal("timed out waiting for background prefetch to start")
	}

	done := make(chan error, 1)
	go func() {
		_, err := r.Seek(1, io.SeekStart)
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			close(fetcher.release)
			_ = r.Close()
			t.Fatalf("Seek returned error: %v", err)
		}
	case <-time.After(250 * time.Millisecond):
		close(fetcher.release)
		_ = r.Close()
		t.Fatal("Seek blocked on canceled prefetch work")
	}

	close(fetcher.release)
	if err := r.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

func TestSegmentReaderSeekDoesNotCancelInFlightForegroundRead(t *testing.T) {
	oldLogger := logger.Log
	logger.Log = slog.New(slog.NewTextHandler(io.Discard, nil))
	defer func() {
		logger.Log = oldLogger
	}()

	fetcher := &blockingForegroundSegmentFetcher{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	f := NewFile(context.Background(), testNZBFile("seek-read-test.mkv", 4, 4, 4), nil, nil, fetcher, nil)
	r := NewSegmentReader(context.Background(), f, 0)
	var releaseOnce sync.Once
	releaseFetcher := func() {
		releaseOnce.Do(func() {
			close(fetcher.release)
		})
	}
	defer func() {
		releaseFetcher()
		_ = r.Close()
	}()

	select {
	case <-fetcher.started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for initial segment fetch to start")
	}

	type readResult struct {
		data string
		err  error
	}
	readDone := make(chan readResult, 1)
	go func() {
		buf := make([]byte, 4)
		n, err := r.Read(buf)
		readDone <- readResult{data: string(buf[:n]), err: err}
	}()

	waitForInflightWaiters(t, f, 0, 2)

	if _, err := r.Seek(1, io.SeekStart); err != nil {
		t.Fatalf("Seek returned error: %v", err)
	}

	select {
	case res := <-readDone:
		t.Fatalf("foreground read returned before fetch release: data=%q err=%v", res.data, res.err)
	default:
	}

	releaseFetcher()

	select {
	case res := <-readDone:
		if res.err != nil {
			t.Fatalf("expected foreground read to succeed after seek, got %v", res.err)
		}
		if res.data != "aaaa" {
			t.Fatalf("expected foreground read data %q, got %q", "aaaa", res.data)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for foreground read to complete")
	}
}

func TestSegmentReaderEvictsPlayedSegmentOnTransition(t *testing.T) {
	oldLogger := logger.Log
	logger.Log = slog.New(slog.NewTextHandler(io.Discard, nil))
	defer func() {
		logger.Log = oldLogger
	}()

	f := NewFile(context.Background(), testNZBFile("evict-test.mkv", 4, 4, 4), nil, nil, &staticSegmentFetcher{}, nil)
	f.PutCachedSegment(0, []byte("aaaa"))
	f.PutCachedSegment(1, []byte("bbbb"))
	f.PutCachedSegment(2, []byte("cccc"))
	r := NewSegmentReader(context.Background(), f, 0)
	defer func() { _ = r.Close() }()

	buf := make([]byte, 4)
	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}
	if n != len(buf) {
		t.Fatalf("expected to read %d bytes, got %d", len(buf), n)
	}
	if got := string(buf); got != "aaaa" {
		t.Fatalf("expected first segment bytes, got %q", got)
	}
	if _, ok := f.GetCachedSegment(0); ok {
		t.Fatal("expected segment 0 to be evicted after advancing to segment 1")
	}
	if _, ok := f.GetCachedSegment(1); !ok {
		t.Fatal("expected current segment to remain cached")
	}
}

type blockingSegmentFetcher struct {
	started   chan struct{}
	release   chan struct{}
	startOnce sync.Once
}

func (f *blockingSegmentFetcher) FetchSegment(ctx context.Context, segment *nzb.Segment, groups []string) (pool.SegmentData, error) {
	if segment.Number > 1 {
		f.startOnce.Do(func() { close(f.started) })
		<-f.release
	}
	return pool.SegmentData{Body: bytesForSegment(segment.Number, segment.Bytes), Size: segment.Bytes}, nil
}

type blockingForegroundSegmentFetcher struct {
	started   chan struct{}
	release   chan struct{}
	startOnce sync.Once
}

func (f *blockingForegroundSegmentFetcher) FetchSegment(ctx context.Context, segment *nzb.Segment, groups []string) (pool.SegmentData, error) {
	if segment.Number == 1 {
		f.startOnce.Do(func() { close(f.started) })
		select {
		case <-f.release:
		case <-ctx.Done():
			return pool.SegmentData{}, ctx.Err()
		}
	}
	return pool.SegmentData{Body: bytesForSegment(segment.Number, segment.Bytes), Size: segment.Bytes}, nil
}

func waitForInflightWaiters(t *testing.T, f *File, index, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		f.downloadMu.Lock()
		req := f.inflightDownloads[index]
		got := 0
		if req != nil {
			got = req.waiters
		}
		f.downloadMu.Unlock()
		if got == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for inflight segment %d to reach %d waiters", index, want)
}

type staticSegmentFetcher struct{}

func (f *staticSegmentFetcher) FetchSegment(ctx context.Context, segment *nzb.Segment, groups []string) (pool.SegmentData, error) {
	return pool.SegmentData{Body: bytesForSegment(segment.Number, segment.Bytes), Size: segment.Bytes}, nil
}

func testNZBFile(subject string, sizes ...int64) *nzb.File {
	segments := make([]nzb.Segment, 0, len(sizes))
	for i, size := range sizes {
		segments = append(segments, nzb.Segment{
			ID:     "seg-" + strings.Repeat("x", i+1),
			Bytes:  size,
			Number: i + 1,
		})
	}
	return &nzb.File{Subject: subject, Segments: segments}
}

func bytesForSegment(number int, size int64) []byte {
	if size <= 0 {
		return nil
	}
	b := byte('a' + number - 1)
	data := make([]byte, int(size))
	for i := range data {
		data[i] = b
	}
	return data
}
