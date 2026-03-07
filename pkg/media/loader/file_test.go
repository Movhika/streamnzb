package loader

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"streamnzb/pkg/core/logger"
	"streamnzb/pkg/media/decode"
	"streamnzb/pkg/media/nzb"
	"streamnzb/pkg/usenet/pool"
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

type dedupBlockingSegmentFetcher struct {
	data         []byte
	err          error
	callObserved chan int
	release      chan struct{}
	once         sync.Once
	mu           sync.Mutex
	calls        int
}

func newDedupBlockingSegmentFetcher(data []byte, err error) *dedupBlockingSegmentFetcher {
	return &dedupBlockingSegmentFetcher{
		data:         data,
		err:          err,
		callObserved: make(chan int, 4),
		release:      make(chan struct{}),
	}
}

func (f *dedupBlockingSegmentFetcher) FetchSegment(ctx context.Context, segment *nzb.Segment, groups []string) (pool.SegmentData, error) {
	f.mu.Lock()
	f.calls++
	call := f.calls
	f.mu.Unlock()

	f.callObserved <- call

	select {
	case <-ctx.Done():
		return pool.SegmentData{}, ctx.Err()
	case <-f.release:
	}

	if f.err != nil {
		return pool.SegmentData{}, f.err
	}
	return pool.SegmentData{Body: append([]byte(nil), f.data...)}, nil
}

func (f *dedupBlockingSegmentFetcher) Release() {
	f.once.Do(func() {
		close(f.release)
	})
}

type staleInflightSegmentFetcher struct {
	data                  []byte
	callObserved          chan int
	cancelObserved        chan int
	releaseCanceledCall   chan struct{}
	releaseSuccessfulCall chan struct{}
	mu                    sync.Mutex
	calls                 int
}

func newStaleInflightSegmentFetcher(data []byte) *staleInflightSegmentFetcher {
	return &staleInflightSegmentFetcher{
		data:                  data,
		callObserved:          make(chan int, 4),
		cancelObserved:        make(chan int, 4),
		releaseCanceledCall:   make(chan struct{}),
		releaseSuccessfulCall: make(chan struct{}),
	}
}

func (f *staleInflightSegmentFetcher) FetchSegment(ctx context.Context, segment *nzb.Segment, groups []string) (pool.SegmentData, error) {
	f.mu.Lock()
	f.calls++
	call := f.calls
	f.mu.Unlock()

	f.callObserved <- call

	select {
	case <-ctx.Done():
		f.cancelObserved <- call
		<-f.releaseCanceledCall
		return pool.SegmentData{}, ctx.Err()
	case <-f.releaseSuccessfulCall:
		return pool.SegmentData{Body: append([]byte(nil), f.data...)}, nil
	}
}

func (f *staleInflightSegmentFetcher) ReleaseCanceledCall() {
	close(f.releaseCanceledCall)
}

func (f *staleInflightSegmentFetcher) ReleaseSuccessfulCall() {
	close(f.releaseSuccessfulCall)
}

func testNZBFileWithSegments(sizes ...int64) *nzb.File {
	segments := make([]nzb.Segment, len(sizes))
	for i, size := range sizes {
		segments[i] = nzb.Segment{ID: fmt.Sprintf("<seg-%d>", i), Number: i + 1, Bytes: size}
	}
	return &nzb.File{Subject: "test.mkv", Groups: []string{"alt.test"}, Segments: segments}
}

func TestDownloadSegmentDeduplicatesConcurrentCalls(t *testing.T) {
	oldLogger := logger.Log
	logger.Log = slog.New(slog.NewTextHandler(io.Discard, nil))
	defer func() {
		logger.Log = oldLogger
	}()

	fetcher := newDedupBlockingSegmentFetcher([]byte("abc"), nil)
	f := NewFile(context.Background(), testNZBFileWithSegments(3), nil, nil, fetcher, nil)

	results := make(chan []byte, 2)
	errs := make(chan error, 2)
	go func() {
		data, err := f.DownloadSegment(context.Background(), 0)
		results <- data
		errs <- err
	}()

	if got := <-fetcher.callObserved; got != 1 {
		t.Fatalf("expected first fetch call to be 1, got %d", got)
	}

	go func() {
		data, err := f.DownloadSegment(context.Background(), 0)
		results <- data
		errs <- err
	}()

	select {
	case got := <-fetcher.callObserved:
		t.Fatalf("expected one underlying fetch, saw call %d", got)
	case <-time.After(100 * time.Millisecond):
	}

	fetcher.Release()
	for i := 0; i < 2; i++ {
		if err := <-errs; err != nil {
			t.Fatalf("DownloadSegment returned error: %v", err)
		}
		if got := string(<-results); got != "abc" {
			t.Fatalf("expected shared data %q, got %q", "abc", got)
		}
	}
}

func TestConcurrentDownloadFailureCountsOnce(t *testing.T) {
	oldLogger := logger.Log
	logger.Log = slog.New(slog.NewTextHandler(io.Discard, nil))
	defer func() {
		logger.Log = oldLogger
	}()

	fetcher := newDedupBlockingSegmentFetcher(nil, errors.New("boom"))
	f := NewFile(context.Background(), testNZBFileWithSegments(3, 4), nil, nil, fetcher, nil)

	results := make(chan []byte, 2)
	errs := make(chan error, 2)
	go func() {
		data, err := f.DownloadSegment(context.Background(), 1)
		results <- data
		errs <- err
	}()

	if got := <-fetcher.callObserved; got != 1 {
		t.Fatalf("expected first fetch call to be 1, got %d", got)
	}

	go func() {
		data, err := f.DownloadSegment(context.Background(), 1)
		results <- data
		errs <- err
	}()

	select {
	case got := <-fetcher.callObserved:
		t.Fatalf("expected one underlying failing fetch, saw call %d", got)
	case <-time.After(100 * time.Millisecond):
	}

	fetcher.Release()
	for i := 0; i < 2; i++ {
		if err := <-errs; err != nil {
			t.Fatalf("DownloadSegment returned error: %v", err)
		}
		if got := <-results; len(got) != 4 {
			t.Fatalf("expected zero-filled segment of length 4, got %d", len(got))
		}
	}
	if f.zeroFillCount != 1 {
		t.Fatalf("expected one shared zero-fill count, got %d", f.zeroFillCount)
	}
}

func TestDownloadSegmentLeaderCancellationDoesNotCancelFollower(t *testing.T) {
	oldLogger := logger.Log
	logger.Log = slog.New(slog.NewTextHandler(io.Discard, nil))
	defer func() {
		logger.Log = oldLogger
	}()

	fetcher := newDedupBlockingSegmentFetcher([]byte("abc"), nil)
	f := NewFile(context.Background(), testNZBFileWithSegments(3), nil, nil, fetcher, nil)

	leaderCtx, cancelLeader := context.WithCancel(context.Background())
	defer cancelLeader()

	leaderErrs := make(chan error, 1)
	go func() {
		_, err := f.DownloadSegment(leaderCtx, 0)
		leaderErrs <- err
	}()

	if got := <-fetcher.callObserved; got != 1 {
		t.Fatalf("expected first fetch call to be 1, got %d", got)
	}

	followerData := make(chan []byte, 1)
	followerErrs := make(chan error, 1)
	go func() {
		data, err := f.DownloadSegment(context.Background(), 0)
		followerData <- data
		followerErrs <- err
	}()

	select {
	case got := <-fetcher.callObserved:
		t.Fatalf("expected follower to join existing fetch, saw call %d", got)
	case <-time.After(100 * time.Millisecond):
	}

	cancelLeader()

	select {
	case err := <-leaderErrs:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected leader error %v, got %v", context.Canceled, err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for leader cancellation")
	}

	fetcher.Release()

	select {
	case err := <-followerErrs:
		if err != nil {
			t.Fatalf("follower DownloadSegment returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for follower result")
	}
	if got := string(<-followerData); got != "abc" {
		t.Fatalf("expected follower data %q, got %q", "abc", got)
	}
}

func TestDownloadSegmentDoesNotJoinWaiterlessCanceledInflightRequest(t *testing.T) {
	oldLogger := logger.Log
	logger.Log = slog.New(slog.NewTextHandler(io.Discard, nil))
	defer func() {
		logger.Log = oldLogger
	}()

	fetcher := newStaleInflightSegmentFetcher([]byte("fresh"))
	f := NewFile(context.Background(), testNZBFileWithSegments(5), nil, nil, fetcher, nil)

	leaderCtx, cancelLeader := context.WithCancel(context.Background())
	defer cancelLeader()

	leaderErrs := make(chan error, 1)
	go func() {
		_, err := f.DownloadSegment(leaderCtx, 0)
		leaderErrs <- err
	}()

	if got := <-fetcher.callObserved; got != 1 {
		t.Fatalf("expected first fetch call to be 1, got %d", got)
	}

	cancelLeader()

	select {
	case err := <-leaderErrs:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected leader error %v, got %v", context.Canceled, err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for leader cancellation")
	}

	select {
	case got := <-fetcher.cancelObserved:
		if got != 1 {
			t.Fatalf("expected canceled fetch call to be 1, got %d", got)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for canceled fetch observation")
	}

	followerData := make(chan []byte, 1)
	followerErrs := make(chan error, 1)
	go func() {
		data, err := f.DownloadSegment(context.Background(), 0)
		followerData <- data
		followerErrs <- err
	}()

	select {
	case got := <-fetcher.callObserved:
		if got != 2 {
			t.Fatalf("expected fresh underlying fetch call to be 2, got %d", got)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for second underlying fetch; stale request was likely reused")
	}

	fetcher.ReleaseSuccessfulCall()

	select {
	case err := <-followerErrs:
		if err != nil {
			t.Fatalf("follower DownloadSegment returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for follower result")
	}
	if got := string(<-followerData); got != "fresh" {
		t.Fatalf("expected follower data %q, got %q", "fresh", got)
	}

	fetcher.ReleaseCanceledCall()
}
