package loader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"streamnzb/pkg/core/logger"
	"streamnzb/pkg/media/decode"
	"streamnzb/pkg/media/nzb"
	"streamnzb/pkg/media/unpack"
	"streamnzb/pkg/usenet/nntp"
	"streamnzb/pkg/usenet/pool"
)

type SegmentFetcher interface {
	FetchSegment(ctx context.Context, segment *nzb.Segment, groups []string) (pool.SegmentData, error)
}

const MaxZeroFills = 10

// isArticleNotFound reports whether err indicates the article is missing (430 No Such Article).
// Used to fail fast on the first segment instead of zero-filling through many segments.
func isArticleNotFound(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "430") || strings.Contains(s, "no such article")
}

func (f *File) IsFailed() bool {
	f.zeroFillMu.Lock()
	defer f.zeroFillMu.Unlock()
	return f.zeroFillCount >= MaxZeroFills
}

type Segment struct {
	nzb.Segment
	StartOffset int64
	EndOffset   int64
}

type File struct {
	nzbFile   *nzb.File
	pools     []*nntp.ClientPool
	fetcher   SegmentFetcher
	estimator *SegmentSizeEstimator
	segments  []*Segment
	totalSize int64
	detected  bool
	ctx       context.Context
	mu        sync.Mutex

	segCache   map[int][]byte
	segCacheMu sync.RWMutex

	zeroFillMu    sync.Mutex
	zeroFillCount int
}

func NewFile(ctx context.Context, f *nzb.File, pools []*nntp.ClientPool, estimator *SegmentSizeEstimator, fetcher SegmentFetcher) *File {
	segments := make([]*Segment, len(f.Segments))
	var offset int64
	for i, s := range f.Segments {
		segments[i] = &Segment{
			Segment:     s,
			StartOffset: offset,
			EndOffset:   offset + s.Bytes,
		}
		offset += s.Bytes
	}
	return &File{
		nzbFile:   f,
		pools:     pools,
		fetcher:   fetcher,
		estimator: estimator,
		segments:  segments,
		totalSize: offset,
		ctx:       ctx,
		segCache:  make(map[int][]byte),
	}
}

func (f *File) Name() string { return f.nzbFile.Subject }

func (f *File) Size() int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.totalSize
}

func (f *File) Segments() []*Segment { return f.segments }

func (f *File) SegmentCount() int { return len(f.segments) }

func (f *File) TotalConnections() int {
	if f.fetcher != nil {
		return 20
	}
	total := 0
	for _, p := range f.pools {
		total += p.MaxConn()
	}
	return total
}

func (f *File) SegmentMapDetected() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.detected
}

func (f *File) EnsureSegmentMap() error {
	f.mu.Lock()
	if f.detected {
		f.mu.Unlock()
		return nil
	}
	f.mu.Unlock()
	return f.detectSegmentSize()
}

func (f *File) detectSegmentSize() error {
	f.mu.Lock()
	if f.detected {
		f.mu.Unlock()
		return nil
	}
	f.mu.Unlock()

	firstEncoded := f.segments[0].Bytes
	if f.estimator != nil {
		if decoded, ok := f.estimator.Get(firstEncoded); ok {
			f.mu.Lock()
			if f.detected {
				f.mu.Unlock()
				return nil
			}
			logger.Trace("Using estimated segment size", "name", f.Name(), "size", decoded)
			f.applySegmentSize(decoded)
			f.mu.Unlock()
			return nil
		}
	}

	data, err := f.DownloadSegment(f.ctx, 0)
	if err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	if f.detected {
		return nil
	}
	if len(data) == 0 {
		return errors.New("empty first segment")
	}

	segSize := int64(len(data))
	logger.Debug("Detected segment size", "name", f.Name(), "size", segSize, "nzb_size", f.segments[0].Bytes)
	if f.estimator != nil {
		f.estimator.Set(f.segments[0].Bytes, segSize)
	}
	f.applySegmentSize(segSize)
	return nil
}

func (f *File) applySegmentSize(segSize int64) {
	var offset int64
	for i := range f.segments {
		f.segments[i].StartOffset = offset
		if i < len(f.segments)-1 {
			f.segments[i].EndOffset = offset + segSize
			offset += segSize
		} else {
			ratio := float64(segSize) / float64(f.segments[0].Bytes)
			estSize := int64(float64(f.segments[i].Bytes) * ratio)
			f.segments[i].EndOffset = offset + estSize
			offset += estSize
		}
	}
	f.totalSize = offset
	f.detected = true
	logger.Trace("Recalculated total decoded size", "size", f.totalSize)
}

func (f *File) FindSegmentIndex(offset int64) int {
	idx := sort.Search(len(f.segments), func(i int) bool {
		return f.segments[i].EndOffset > offset
	})
	if idx < len(f.segments) && offset >= f.segments[idx].StartOffset {
		return idx
	}
	return -1
}

func (f *File) GetCachedSegment(index int) ([]byte, bool) {
	f.segCacheMu.RLock()
	data, ok := f.segCache[index]
	f.segCacheMu.RUnlock()
	return data, ok
}

func (f *File) PutCachedSegment(index int, data []byte) {
	f.segCacheMu.Lock()
	f.segCache[index] = data
	f.segCacheMu.Unlock()
}

func (f *File) EvictCachedSegmentsBefore(minIndex int) {
	f.segCacheMu.Lock()
	for idx := range f.segCache {
		if idx < minIndex {
			delete(f.segCache, idx)
		}
	}
	f.segCacheMu.Unlock()
}

func (f *File) PrewarmSegment(index int) {
	if index < 0 || index >= len(f.segments) {
		return
	}
	if _, ok := f.GetCachedSegment(index); ok {
		return
	}
	go f.DownloadSegment(f.ctx, index)
}

func (f *File) StartDownloadSegment(ctx context.Context, index int) <-chan struct{} {
	done := make(chan struct{})
	if _, ok := f.GetCachedSegment(index); ok {
		close(done)
		return done
	}
	go func() {
		_, _ = f.DownloadSegment(ctx, index)
		close(done)
	}()
	return done
}

func (f *File) DownloadSegment(ctx context.Context, index int) ([]byte, error) {
	if data, ok := f.GetCachedSegment(index); ok {
		return data, nil
	}
	return f.doDownloadSegment(ctx, index)
}

func (f *File) doDownloadSegment(ctx context.Context, index int) ([]byte, error) {
	if f.fetcher != nil {
		return f.doDownloadSegmentViaFetcher(ctx, index)
	}
	return f.doDownloadSegmentViaPools(ctx, index)
}

func (f *File) doDownloadSegmentViaFetcher(ctx context.Context, index int) ([]byte, error) {
	downloadCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	seg := f.segments[index]
	data, err := f.fetcher.FetchSegment(downloadCtx, &seg.Segment, f.nzbFile.Groups)
	if err != nil {
		if index == 0 && isArticleNotFound(err) {
			return nil, fmt.Errorf("first segment unavailable: %w", err)
		}
		f.zeroFillMu.Lock()
		count := f.zeroFillCount
		if count >= MaxZeroFills {
			f.zeroFillMu.Unlock()
			return nil, fmt.Errorf("too many failed segments (%d/%d): %w", count+1, MaxZeroFills, errors.Join(unpack.ErrTooManyZeroFills, err))
		}
		f.zeroFillCount++
		f.zeroFillMu.Unlock()
		logger.Trace("Segment fetch failed, zero-filling", "index", index, "count", count+1, "max", MaxZeroFills, "err", err)
		size := int(seg.EndOffset - seg.StartOffset)
		if size < 0 {
			size = 0
		}
		zeroData := make([]byte, size)
		f.PutCachedSegment(index, zeroData)
		return zeroData, nil
	}
	f.PutCachedSegment(index, data.Body)
	return data.Body, nil
}

func (f *File) doDownloadSegmentViaPools(ctx context.Context, index int) ([]byte, error) {
	downloadCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	seg := f.segments[index]
	tried := make([]bool, len(f.pools))
	var lastErr error

	for attempt := 0; attempt < len(f.pools); attempt++ {
		select {
		case <-downloadCtx.Done():
			return nil, downloadCtx.Err()
		default:
		}

		var client *nntp.Client
		var clientPool *nntp.ClientPool
		var poolIdx int = -1

		for i, p := range f.pools {
			if !tried[i] {
				if c, ok := p.TryGet(downloadCtx); ok {
					client = c
					clientPool = p
					poolIdx = i
					break
				}
			}
		}

		if client == nil {
			for i, p := range f.pools {
				if !tried[i] {
					var err error
					client, err = p.Get(downloadCtx)
					if err != nil {
						tried[i] = true
						lastErr = err
						if errors.Is(err, context.Canceled) {
							return nil, err
						}
						continue
					}
					clientPool = p
					poolIdx = i
					break
				}
			}
		}

		if client == nil {
			break
		}

		if len(f.nzbFile.Groups) > 0 {
			client.Group(f.nzbFile.Groups[0])
		}

		r, err := client.Body(seg.ID)
		if err != nil {
			clientPool.Put(client)
			tried[poolIdx] = true
			lastErr = err
			continue
		}

		type decodeResult struct {
			frame *decode.Frame
			err   error
		}
		done := make(chan decodeResult, 1)
		go func() {
			frame, err := decode.DecodeToBytes(r)
			done <- decodeResult{frame, err}
		}()

		select {
		case <-downloadCtx.Done():
			clientPool.Discard(client)
			return nil, downloadCtx.Err()
		case res := <-done:
			if res.err != nil {
				clientPool.Put(client)
				tried[poolIdx] = true
				lastErr = res.err
				continue
			}
			clientPool.Put(client)
			f.PutCachedSegment(index, res.frame.Data)
			return res.frame.Data, nil
		}
	}

	if index == 0 && lastErr != nil && isArticleNotFound(lastErr) {
		return nil, fmt.Errorf("first segment unavailable: %w", lastErr)
	}

	f.zeroFillMu.Lock()
	count := f.zeroFillCount
	if count >= MaxZeroFills {
		f.zeroFillMu.Unlock()
		return nil, fmt.Errorf("too many failed segments (%d/%d): %w", count+1, MaxZeroFills, errors.Join(unpack.ErrTooManyZeroFills, lastErr))
	}
	f.zeroFillCount++
	f.zeroFillMu.Unlock()

	logger.Trace("Segment failed on all providers, zero-filling", "index", index, "count", count+1, "max", MaxZeroFills, "err", lastErr)
	size := int(seg.EndOffset - seg.StartOffset)
	if size < 0 {
		size = 0
	}
	zeroData := make([]byte, size)
	f.PutCachedSegment(index, zeroData)
	return zeroData, nil
}

func (f *File) ReadAt(p []byte, off int64) (n int, err error) {
	if err := f.EnsureSegmentMap(); err != nil {
		return 0, err
	}
	if off >= f.totalSize {
		return 0, io.EOF
	}

	startIdx := f.FindSegmentIndex(off)
	if startIdx == -1 {
		return 0, io.EOF
	}

	currentOffset := off
	totalRead := 0
	for i := startIdx; i < len(f.segments) && totalRead < len(p); i++ {
		seg := f.segments[i]
		segOff := currentOffset - seg.StartOffset

		data, err := f.DownloadSegment(f.ctx, i)
		if err != nil {
			return totalRead, err
		}
		if segOff >= int64(len(data)) {
			continue
		}

		copied := copy(p[totalRead:], data[segOff:])
		totalRead += copied
		currentOffset += int64(copied)
	}

	if totalRead < len(p) && currentOffset >= f.totalSize {
		return totalRead, io.EOF
	}
	return totalRead, nil
}

func (f *File) OpenStream() (io.ReadSeekCloser, error) {
	return f.OpenStreamCtx(f.ctx)
}

func (f *File) OpenStreamCtx(ctx context.Context) (io.ReadSeekCloser, error) {
	if err := f.EnsureSegmentMap(); err != nil {
		return nil, err
	}
	return NewSegmentReader(ctx, f, 0), nil
}

func (f *File) OpenReaderAt(ctx context.Context, offset int64) (io.ReadCloser, error) {
	if err := f.EnsureSegmentMap(); err != nil {
		return nil, err
	}
	return NewSegmentReader(ctx, f, offset), nil
}

type SegmentSizeEstimator struct {
	entries []sizeEntry
	mu      sync.RWMutex
}

type sizeEntry struct {
	encoded int64
	decoded int64
}

func NewSegmentSizeEstimator() *SegmentSizeEstimator {
	return &SegmentSizeEstimator{entries: make([]sizeEntry, 0, 4)}
}

func (e *SegmentSizeEstimator) Get(encodedSize int64) (int64, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	for _, entry := range e.entries {
		diff := entry.encoded - encodedSize
		if diff < 0 {
			diff = -diff
		}
		if diff < 4096 {
			return entry.decoded, true
		}
	}
	return 0, false
}

func (e *SegmentSizeEstimator) Set(encodedSize, decodedSize int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, entry := range e.entries {
		diff := entry.encoded - encodedSize
		if diff < 0 {
			diff = -diff
		}
		if diff < 4096 {
			return
		}
	}
	e.entries = append(e.entries, sizeEntry{encoded: encodedSize, decoded: decodedSize})
}
