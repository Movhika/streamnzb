package loader

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"time"

	"streamnzb/pkg/core/logger"
)

type SegmentReader struct {
	file   *File
	ctx    context.Context
	cancel context.CancelFunc
	parent context.Context

	mu     sync.Mutex
	segIdx int
	segOff int64
	offset int64
	closed bool

	prefetchWg  sync.WaitGroup
	prefetching map[int]bool
}

func NewSegmentReader(parent context.Context, f *File, startOffset int64) *SegmentReader {
	ctx, cancel := context.WithCancel(parent)
	sr := &SegmentReader{
		file:        f,
		ctx:         ctx,
		cancel:      cancel,
		parent:      parent,
		offset:      startOffset,
		prefetching: make(map[int]bool),
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

	sr.startPrefetch()

	return sr
}

const maxPrefetchAhead = 16 // cap "ahead" cache so memory stays bounded during playback

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

	// Evict already-played segments on every Read so cache doesn't grow with playback position.
	r.file.EvictCachedSegmentsBefore(segIdx)
	// Cap how far ahead we keep; prefetch will refill only up to this window.
	r.file.EvictCachedSegmentsAfter(segIdx + maxPrefetchAhead)

	data, err := r.waitForSegment(segIdx)
	if err != nil {
		return 0, err
	}

	if segOff >= int64(len(data)) {
		r.mu.Lock()
		r.segIdx++
		r.segOff = 0
		r.mu.Unlock()
		if r.segIdx >= len(r.file.segments) {
			return 0, io.EOF
		}
		return r.Read(p)
	}

	n := copy(p, data[segOff:])

	r.mu.Lock()
	r.segOff += int64(n)
	r.offset += int64(n)
	if r.segOff >= int64(len(data)) {
		r.segIdx++
		r.segOff = 0
	}
	r.mu.Unlock()

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
	current := r.segIdx
	r.mu.Unlock()

	maxWorkers := r.file.TotalConnections()
	if maxWorkers > 20 {
		maxWorkers = 20
	}
	if maxWorkers < 1 {
		maxWorkers = 1
	}

	r.mu.Lock()
	for i := 0; i < maxPrefetchAhead; i++ {
		idx := current + i
		if idx >= len(r.file.segments) {
			break
		}
		if _, ok := r.file.GetCachedSegment(idx); ok {
			continue
		}
		if r.prefetching[idx] {
			continue
		}
		r.prefetching[idx] = true
		r.prefetchWg.Add(1)
		go func(segIdx int) {
			defer r.prefetchWg.Done()
			defer func() {
				r.mu.Lock()
				delete(r.prefetching, segIdx)
				r.mu.Unlock()
			}()
			// Use PrefetchSegment instead of DownloadSegment so that transient provider
			// failures in background goroutines do not increment zeroFillCount and
			// prematurely mark the file as failed via IsFailed() before the player reads it.
			// The blocking read path (DownloadSegment) will count failures if it also
			// exhausts all providers for the same segment.
			_, err := r.file.PrefetchSegment(r.ctx, segIdx)
			if err != nil && !isContextErr(err) {
				logger.Trace("Prefetch failed", "seg", segIdx, "err", err)
			}
		}(idx)
	}
	r.mu.Unlock()
}

func (r *SegmentReader) Seek(offset int64, whence int) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
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
		return 0, errors.New("invalid whence")
	}

	if target < 0 || target > r.file.Size() {
		return 0, errors.New("seek out of bounds")
	}

	if target == r.offset {
		return target, nil
	}

	r.cancel()

	r.ctx, r.cancel = context.WithCancel(r.parent)

	r.prefetching = make(map[int]bool)
	r.mu.Unlock()

	r.prefetchWg.Wait()

	r.mu.Lock()
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

	r.mu.Unlock()
	r.startPrefetch()
	r.mu.Lock()

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
	return nil
}

func isContextErr(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	s := err.Error()
	return strings.Contains(s, "canceled") || strings.Contains(s, "cancelled")
}
