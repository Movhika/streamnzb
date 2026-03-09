package loader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"streamnzb/pkg/core/logger"
)

type SegmentReader struct {
	file            *File
	ctx             context.Context
	cancel          context.CancelFunc
	prefetchCtx     context.Context
	prefetchCancel  context.CancelFunc
	parent          context.Context
	traceID         uint64
	prefetchEnabled bool

	mu     sync.Mutex
	segIdx int
	segOff int64
	offset int64
	closed bool

	prefetchWg         sync.WaitGroup
	prefetching        map[int]uint64
	prefetchGeneration uint64
}

type SegmentReaderOptions struct {
	EnablePrefetch bool
}

var liveSegmentReaders atomic.Int64
var nextSegmentReaderID atomic.Uint64
var liveSegmentReaderRegistry sync.Map

func LiveSegmentReaders() int64 {
	return liveSegmentReaders.Load()
}

func LiveSegmentReaderDetails() []string {
	details := make([]string, 0)
	liveSegmentReaderRegistry.Range(func(_, value any) bool {
		r, ok := value.(*SegmentReader)
		if !ok || r == nil {
			return true
		}
		details = append(details, r.traceDetail())
		return true
	})
	sort.Strings(details)
	return details
}

func NewSegmentReader(parent context.Context, f *File, startOffset int64) *SegmentReader {
	return NewSegmentReaderWithOptions(parent, f, startOffset, SegmentReaderOptions{EnablePrefetch: true})
}

func NewSegmentReaderWithOptions(parent context.Context, f *File, startOffset int64, opts SegmentReaderOptions) *SegmentReader {
	if parent == nil {
		parent = context.Background()
	}

	ctx, cancel := context.WithCancel(parent)
	sr := &SegmentReader{
		file:            f,
		ctx:             ctx,
		cancel:          cancel,
		parent:          parent,
		traceID:         nextSegmentReaderID.Add(1),
		offset:          startOffset,
		prefetching:     make(map[int]uint64),
		prefetchEnabled: opts.EnablePrefetch,
	}
	if sr.prefetchEnabled {
		sr.prefetchCtx, sr.prefetchCancel = context.WithCancel(parent)
	} else {
		sr.prefetchCtx = parent
		sr.prefetchCancel = func() {}
	}

	idx := f.FindSegmentIndex(startOffset)
	if idx == -1 {
		if startOffset >= f.Size() {
			sr.segIdx = len(f.segments)
		} else {
			sr.segIdx = 0
		}
	} else {
		sr.segIdx = idx
		sr.segOff = startOffset - f.segments[idx].StartOffset
	}

	sr.applyCacheWindow(sr.segIdx)
	if sr.prefetchEnabled {
		sr.startPrefetch()
	}
	liveSegmentReaders.Add(1)
	liveSegmentReaderRegistry.Store(sr.traceID, sr)

	return sr
}

func (r *SegmentReader) traceDetail() string {
	r.mu.Lock()
	id := r.traceID
	segIdx := r.segIdx
	segOff := r.segOff
	offset := r.offset
	closed := r.closed
	r.mu.Unlock()

	sessionID := "unknown"
	fileName := ""
	if r.file != nil {
		fileName = r.file.Name()
		if ownerID := r.file.OwnerSessionID(); ownerID != "" {
			sessionID = ownerID
		}
	}

	return fmt.Sprintf("id=%d session=%s file=%q offset=%d seg=%d seg_off=%d closed=%t", id, sessionID, fileName, offset, segIdx, segOff, closed)
}

const maxPrefetchAhead = 16 // cap "ahead" cache so memory stays bounded during playback

func (r *SegmentReader) applyCacheWindow(segIdx int) {
	if segIdx < 0 {
		return
	}
	r.file.EvictCachedSegmentsBefore(segIdx)
	r.file.EvictCachedSegmentsAfter(segIdx + maxPrefetchAhead)
}

func (r *SegmentReader) Read(p []byte) (int, error) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return 0, io.ErrClosedPipe
	}
	if r.segIdx >= len(r.file.segments) {
		r.mu.Unlock()
		return 0, io.EOF
	}
	segIdx := r.segIdx
	segOff := r.segOff
	r.mu.Unlock()

	data, err := r.waitForSegment(segIdx)
	if err != nil {
		return 0, err
	}

	if segOff >= int64(len(data)) {
		r.mu.Lock()
		r.segIdx++
		r.segOff = 0
		nextSegIdx := r.segIdx
		r.mu.Unlock()
		r.applyCacheWindow(nextSegIdx)
		if nextSegIdx >= len(r.file.segments) {
			return 0, io.EOF
		}
		r.startPrefetch()
		return r.Read(p)
	}

	n := copy(p, data[segOff:])

	r.mu.Lock()
	currentSegIdx := r.segIdx
	r.segOff += int64(n)
	r.offset += int64(n)
	if r.segOff >= int64(len(data)) {
		r.segIdx++
		r.segOff = 0
	}
	nextSegIdx := r.segIdx
	r.mu.Unlock()

	if nextSegIdx != currentSegIdx {
		r.applyCacheWindow(nextSegIdx)
	}

	r.startPrefetch()

	return n, nil
}

func (r *SegmentReader) waitForSegment(index int) ([]byte, error) {

	if data, ok := r.file.GetCachedSegment(index); ok {
		return data, nil
	}

	return r.file.DownloadSegment(r.ctx, index)
}

func (r *SegmentReader) startPrefetch() {
	r.mu.Lock()
	if r.closed || !r.prefetchEnabled {
		r.mu.Unlock()
		return
	}
	current := r.segIdx
	ctx := r.prefetchCtx
	generation := r.prefetchGeneration

	// Cap concurrent prefetch goroutines to the actual pool connection count so we
	// don't spawn 16 goroutines when the pool only has e.g. 2 connections — the
	// excess goroutines would immediately block on pool.Get(), wasting stack memory
	// and adding lock contention.
	maxWorkers := r.file.TotalConnections()
	if maxWorkers > 20 {
		maxWorkers = 20
	}
	if maxWorkers < 1 {
		maxWorkers = 1
	}

	inFlight := len(r.prefetching)
	for i := 0; i < maxPrefetchAhead; i++ {
		if inFlight >= maxWorkers {
			break
		}
		idx := current + i
		if idx >= len(r.file.segments) {
			break
		}
		if _, ok := r.file.GetCachedSegment(idx); ok {
			continue
		}
		if currentGen, ok := r.prefetching[idx]; ok && currentGen == generation {
			continue
		}
		r.prefetching[idx] = generation
		inFlight++
		r.prefetchWg.Add(1)
		go func(segIdx int, ctx context.Context, generation uint64) {
			defer r.prefetchWg.Done()
			defer func() {
				r.mu.Lock()
				if currentGen, ok := r.prefetching[segIdx]; ok && currentGen == generation {
					delete(r.prefetching, segIdx)
				}
				r.mu.Unlock()
			}()
			// Use PrefetchSegment instead of DownloadSegment so that transient provider
			// failures in background goroutines do not increment zeroFillCount and
			// prematurely mark the file as failed via IsFailed() before the player reads it.
			// The blocking read path (DownloadSegment) will count failures if it also
			// exhausts all providers for the same segment.
			_, err := r.file.PrefetchSegment(ctx, segIdx)
			if err != nil && !isContextErr(err) {
				logger.Trace("Prefetch failed", "seg", segIdx, "err", err)
			}
		}(idx, ctx, generation)
	}
	r.mu.Unlock()
}

func (r *SegmentReader) Seek(offset int64, whence int) (int64, error) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return 0, io.ErrClosedPipe
	}

	var target int64
	switch whence {
	case io.SeekStart:
		target = offset
	case io.SeekCurrent:
		target = r.offset + offset
	case io.SeekEnd:
		target = r.file.Size() + offset
	default:
		r.mu.Unlock()
		return 0, errors.New("invalid whence")
	}

	if target < 0 || target > r.file.Size() {
		r.mu.Unlock()
		return 0, errors.New("seek out of bounds")
	}

	if target == r.offset {
		r.mu.Unlock()
		return target, nil
	}

	if r.prefetchEnabled {
		r.prefetchCancel()
		r.prefetchCtx, r.prefetchCancel = context.WithCancel(r.parent)
		r.prefetchGeneration++
		r.prefetching = make(map[int]uint64)
	}
	r.offset = target
	if target >= r.file.Size() {
		r.segIdx = len(r.file.segments)
		r.segOff = 0
	} else {
		idx := r.file.FindSegmentIndex(target)
		if idx == -1 {
			r.segIdx = len(r.file.segments)
			r.segOff = 0
		} else {
			r.segIdx = idx
			r.segOff = target - r.file.segments[idx].StartOffset
		}
	}
	currentSegIdx := r.segIdx
	r.mu.Unlock()
	r.applyCacheWindow(currentSegIdx)
	if r.prefetchEnabled {
		r.startPrefetch()
	}

	return target, nil
}

func (r *SegmentReader) Close() error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil
	}
	r.closed = true
	r.mu.Unlock()

	logger.Debug("loader SegmentReader Close", "file", r.file.Name())
	r.cancel()
	r.prefetchCancel()

	done := make(chan struct{})
	go func() {
		r.prefetchWg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
	}
	// Drop cached segments for this file so already-played data is released (single file: at stream end; RAR: when we move to the next volume).
	r.file.ClearSegmentCache()
	liveSegmentReaderRegistry.Delete(r.traceID)
	liveSegmentReaders.Add(-1)
	return nil
}

func isContextErr(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	s := err.Error()
	return strings.Contains(s, "canceled") || strings.Contains(s, "cancelled")
}
