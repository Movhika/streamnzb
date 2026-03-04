package pool

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
	"streamnzb/pkg/usenet/nntp"

	"golang.org/x/sync/singleflight"
)

type countReader struct {
	io.Reader
	n int64
}

func (c *countReader) Read(p []byte) (int, error) {
	n, err := c.Reader.Read(p)
	c.n += int64(n)
	return n, err
}

var (
	ErrNoProvidersConfigured = errors.New("usenet/pool: no providers configured")
	ErrNoProvidersAvailable  = errors.New("usenet/pool: no providers available")
)

// isArticleNotFound reports whether err indicates 430 No Such Article (article missing on server).
// On 430 we return immediately instead of trying other providers.
func isArticleNotFound(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "430") || strings.Contains(s, "no such article")
}

type ProviderConfig struct {
	ID         string
	Priority   int
	IsBackup   bool
	ClientPool *nntp.ClientPool
}

type Config struct {
	Providers    []ProviderConfig
	SegmentCache SegmentCache
}

type Pool struct {
	providers []ProviderConfig
	cache     SegmentCache
	sf        singleflight.Group
	mu        sync.RWMutex
}

func NewPool(cfg *Config) (*Pool, error) {
	if cfg == nil || len(cfg.Providers) == 0 {
		return nil, ErrNoProvidersConfigured
	}
	providers := make([]ProviderConfig, len(cfg.Providers))
	copy(providers, cfg.Providers)
	sort.Slice(providers, func(i, j int) bool {
		return providers[i].Priority < providers[j].Priority
	})
	cache := cfg.SegmentCache
	if cache == nil {
		cache = NoopSegmentCache()
	}
	return &Pool{
		providers: providers,
		cache:     cache,
	}, nil
}

func (p *Pool) FetchSegment(ctx context.Context, segment *nzb.Segment, groups []string) (SegmentData, error) {
	messageID := strings.TrimSpace(segment.ID)
	if messageID == "" {
		return SegmentData{}, fmt.Errorf("empty segment message ID")
	}

	v, err, _ := p.sf.Do(messageID, func() (interface{}, error) {
		return p.fetchSegmentOnce(ctx, messageID, segment, groups)
	})
	if err != nil {
		return SegmentData{}, err
	}
	return v.(SegmentData), nil
}

// FetchSegmentFirst tries all providers in parallel for the first segment (e.g. segment 0).
// It returns as soon as one provider succeeds, or the last error if all fail.
// Call this for segment 0 to reduce latency when the article is missing on all providers.
func (p *Pool) FetchSegmentFirst(ctx context.Context, segment *nzb.Segment, groups []string) (SegmentData, error) {
	messageID := strings.TrimSpace(segment.ID)
	if messageID == "" {
		return SegmentData{}, fmt.Errorf("empty segment message ID")
	}
	if data, ok := p.cache.Get(messageID); ok {
		logger.Trace("fetch segment cache hit", "message_id", messageID)
		return data, nil
	}

	fetchCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	p.mu.RLock()
	providers := p.providers
	p.mu.RUnlock()

	// Exclude set for each provider: all other provider IDs so getConnection returns that provider.
	providerIDs := make([]string, len(providers))
	for i := range providers {
		providerIDs[i] = providers[i].ID
	}

	type segResult struct {
		data SegmentData
		err  error
	}
	ch := make(chan segResult, len(providers))

	for i := range providers {
		exclude := make([]string, 0, len(providers)-1)
		for j := range providerIDs {
			if j != i {
				exclude = append(exclude, providerIDs[j])
			}
		}
		go func(exclude []string) {
			conn, release, discard, providerID, err := p.getConnection(fetchCtx, exclude, 999, false)
			if err != nil {
				ch <- segResult{err: err}
				return
			}

			// Connection leak guard: if fetchCtx is cancelled (e.g. another provider succeeded
			// or the caller gave up), discard the connection to interrupt the blocking read.
			stopWatch := make(chan struct{})
			go func() {
				select {
				case <-fetchCtx.Done():
					discard()
				case <-stopWatch:
				}
			}()
			defer func() {
				close(stopWatch)
				release()
			}()

			if len(groups) > 0 {
				if err := conn.Group(groups[0]); err != nil {
					logger.Debug("fetch segment group failed", "provider", providerID, "err", err)
					ch <- segResult{err: err}
					return
				}
			}
			r, err := conn.Body(messageID)
			if err != nil {
				logger.Debug("fetch segment body failed", "provider", providerID, "err", err)
				ch <- segResult{err: err}
				return
			}
			cr := &countReader{Reader: r}
			frame, err := decode.DecodeToBytes(cr)
			if err != nil {
				logger.Debug("fetch segment decode failed", "provider", providerID, "err", err)
				ch <- segResult{err: err}
				return
			}
			ch <- segResult{data: SegmentData{Body: frame.Data, Size: int64(len(frame.Data))}}
		}(exclude)
	}

	var lastErr error
	for range providers {
		res := <-ch
		if res.err == nil {
			cancel()
			p.cache.Set(messageID, res.data)
			logger.Trace("fetch segment ok (parallel)", "message_id", messageID, "size", res.data.Size)
			return res.data, nil
		}
		lastErr = res.err
		if isArticleNotFound(res.err) {
			cancel()
			return SegmentData{}, fmt.Errorf("fetch segment %s: %w", messageID, res.err)
		}
	}
	if lastErr != nil {
		return SegmentData{}, fmt.Errorf("fetch segment %s: failed after retries: %w", messageID, lastErr)
	}
	return SegmentData{}, fmt.Errorf("fetch segment %s: failed after retries", messageID)
}

func (p *Pool) fetchSegmentOnce(ctx context.Context, messageID string, segment *nzb.Segment, groups []string) (SegmentData, error) {
	if data, ok := p.cache.Get(messageID); ok {
		logger.Trace("fetch segment cache hit", "message_id", messageID)
		return data, nil
	}

	fetchCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	var exclude []string
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		conn, release, discard, providerID, err := p.getConnection(fetchCtx, exclude, 999, false)
		if err != nil {
			if errors.Is(err, ErrNoProvidersAvailable) && len(exclude) > 0 {
				exclude = nil
				continue
			}
			return SegmentData{}, err
		}

		// Interrupt pending body read if session is closed/cancelled.
		stopWatch := make(chan struct{})
		go func() {
			select {
			case <-fetchCtx.Done():
				discard()
			case <-stopWatch:
			}
		}()

		if len(groups) > 0 {
			if err := conn.Group(groups[0]); err != nil {
				close(stopWatch)
				release()
				logger.Debug("fetch segment group failed", "provider", providerID, "err", err)
				exclude = append(exclude, providerID)
				continue
			}
		}

		r, err := conn.Body(messageID)
		if err != nil {
			close(stopWatch)
			release()
			lastErr = err
			logger.Debug("fetch segment body failed", "provider", providerID, "err", err)
			if isArticleNotFound(err) {
				return SegmentData{}, fmt.Errorf("fetch segment %s: %w", messageID, err)
			}
			exclude = append(exclude, providerID)
			continue
		}

		cr := &countReader{Reader: r}
		frame, err := decode.DecodeToBytes(cr)
		close(stopWatch)
		if err != nil {
			discard()
			errStr := err.Error()
			if strings.Contains(errStr, "expected size") && strings.Contains(errStr, "but got") {
				logger.Debug("fetch segment decode failed", "provider", providerID, "err", err, "raw_body_bytes", cr.n)
			} else {
				logger.Debug("fetch segment decode failed", "provider", providerID, "err", err)
			}
			exclude = append(exclude, providerID)
			continue
		}
		release()

		data := SegmentData{
			Body: frame.Data,
			Size: int64(len(frame.Data)),
		}
		p.cache.Set(messageID, data)
		logger.Trace("fetch segment ok", "message_id", messageID, "size", data.Size)
		return data, nil
	}

	if lastErr != nil {
		return SegmentData{}, fmt.Errorf("fetch segment %s: failed after retries: %w", messageID, lastErr)
	}
	return SegmentData{}, fmt.Errorf("fetch segment %s: failed after retries", messageID)
}

// StatSegment checks whether the article exists on any provider (STAT only, no body).
// Returns (true, nil) if found, (false, nil) if 430 on all providers, (false, err) on other errors.
// Use this before opening a stream to fail fast when the first segment is missing.
func (p *Pool) StatSegment(ctx context.Context, messageID string, groups []string) (exists bool, err error) {
	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		return false, fmt.Errorf("empty segment message ID")
	}

	statCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	p.mu.RLock()
	providers := p.providers
	p.mu.RUnlock()

	providerIDs := make([]string, len(providers))
	for i := range providers {
		providerIDs[i] = providers[i].ID
	}

	type statResult struct {
		exists bool
		err    error
	}
	ch := make(chan statResult, len(providers))

	for i := range providers {
		exclude := make([]string, 0, len(providers)-1)
		for j := range providerIDs {
			if j != i {
				exclude = append(exclude, providerIDs[j])
			}
		}
		go func(exclude []string) {
			conn, release, _, providerID, getErr := p.getConnection(statCtx, exclude, 999, false)
			if getErr != nil {
				ch <- statResult{err: getErr}
				return
			}
			defer release()
			if len(groups) > 0 {
				if groupErr := conn.Group(groups[0]); groupErr != nil {
					logger.Debug("stat segment group failed", "provider", providerID, "err", groupErr)
					ch <- statResult{err: groupErr}
					return
				}
			}
			exists, statErr := conn.StatArticle(messageID)
			if statErr != nil {
				logger.Debug("stat segment failed", "provider", providerID, "err", statErr)
				ch <- statResult{err: statErr}
				return
			}
			ch <- statResult{exists: exists}
		}(exclude)
	}

	var lastErr error
	for range providers {
		res := <-ch
		if res.err == nil && res.exists {
			cancel()
			logger.Trace("stat segment ok", "message_id", messageID)
			return true, nil
		}
		if res.err != nil {
			lastErr = res.err
		}
		if res.err == nil && !res.exists {
			lastErr = nil
		}
	}
	if lastErr != nil {
		return false, fmt.Errorf("stat segment %s: %w", messageID, lastErr)
	}
	logger.Trace("stat segment not found (430)", "message_id", messageID)
	return false, nil
}

func (p *Pool) getConnection(ctx context.Context, exclude []string, maxPriority int, useBackup bool) (client *nntp.Client, release, discard func(), providerID string, err error) {
	p.mu.RLock()
	providers := p.providers
	p.mu.RUnlock()

	excludeSet := make(map[string]bool)
	for _, id := range exclude {
		excludeSet[id] = true
	}

	for i := range providers {
		prov := &providers[i]
		if excludeSet[prov.ID] {
			continue
		}
		if prov.Priority > maxPriority {
			continue
		}
		if prov.IsBackup != useBackup {
			continue
		}

		c, ok := prov.ClientPool.TryGet(ctx)
		if !ok {
			var getErr error
			c, getErr = prov.ClientPool.Get(ctx)
			if getErr != nil {
				if errors.Is(getErr, context.Canceled) {
					return nil, nil, nil, "", getErr
				}
				continue
			}
		}

		pool := prov.ClientPool
		pid := prov.ID
		var once sync.Once
		release := func() {
			once.Do(func() {
				pool.Put(c)
			})
		}
		discard := func() {
			once.Do(func() {
				pool.Discard(c)
			})
		}
		return c, release, discard, pid, nil
	}

	return nil, nil, nil, "", ErrNoProvidersAvailable
}

func (p *Pool) GetConnection(ctx context.Context, exclude []string, maxPriority int, useBackup bool) (client *nntp.Client, release, discard func(), providerID string, err error) {
	return p.getConnection(ctx, exclude, maxPriority, useBackup)
}

func (p *Pool) DiscardConnection(client *nntp.Client, pool *nntp.ClientPool) {
	if client != nil && pool != nil {
		pool.Discard(client)
	}
}

func (p *Pool) CountProviders() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.providers)
}

func (p *Pool) ProviderOrder() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	ids := make([]string, 0, len(p.providers))
	for i := range p.providers {
		ids = append(ids, p.providers[i].ID)
	}
	return ids
}

func (p *Pool) ProviderHosts() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	hosts := make([]string, 0, len(p.providers))
	for i := range p.providers {
		if h := p.providers[i].ClientPool.Host(); h != "" {
			hosts = append(hosts, h)
		}
	}
	return hosts
}

func (p *Pool) Host(providerID string) string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for i := range p.providers {
		if p.providers[i].ID == providerID {
			return p.providers[i].ClientPool.Host()
		}
	}
	return ""
}
