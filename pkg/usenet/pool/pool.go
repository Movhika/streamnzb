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
		conn, release, _, providerID, err := p.getConnection(fetchCtx, exclude, 999, false)
		if err != nil {
			if errors.Is(err, ErrNoProvidersAvailable) && len(exclude) > 0 {
				exclude = nil
				continue
			}
			return SegmentData{}, err
		}

		if len(groups) > 0 {
			if err := conn.Group(groups[0]); err != nil {
				release()
				logger.Debug("fetch segment group failed", "provider", providerID, "err", err)
				exclude = append(exclude, providerID)
				continue
			}
		}

		r, err := conn.Body(messageID)
		if err != nil {
			release()
			lastErr = err
			logger.Debug("fetch segment body failed", "provider", providerID, "err", err)
			exclude = append(exclude, providerID)
			continue
		}

		cr := &countReader{Reader: r}
		frame, err := decode.DecodeToBytes(cr)
		release()
		if err != nil {
			errStr := err.Error()
			if strings.Contains(errStr, "expected size") && strings.Contains(errStr, "but got") {
				logger.Debug("fetch segment decode failed", "provider", providerID, "err", err, "raw_body_bytes", cr.n)
			} else {
				logger.Debug("fetch segment decode failed", "provider", providerID, "err", err)
			}
			exclude = append(exclude, providerID)
			continue
		}

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
		release := func() { pool.Put(c) }
		discard := func() { pool.Discard(c) }
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
