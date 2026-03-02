package pool

import (
	"sync"
)

type SegmentData struct {
	Body []byte
	Size int64
}

type SegmentCache interface {
	Get(messageID string) (SegmentData, bool)
	Set(messageID string, data SegmentData)
}

func NewMemorySegmentCache() SegmentCache {
	return &memorySegmentCache{m: make(map[string]SegmentData)}
}

type memorySegmentCache struct {
	mu sync.RWMutex
	m  map[string]SegmentData
}

func (c *memorySegmentCache) Get(messageID string) (SegmentData, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	data, ok := c.m[messageID]
	return data, ok
}

func (c *memorySegmentCache) Set(messageID string, data SegmentData) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[messageID] = data
}

func NoopSegmentCache() SegmentCache { return &noopSegmentCache{} }

type noopSegmentCache struct{}

func (n *noopSegmentCache) Get(messageID string) (SegmentData, bool) { return SegmentData{}, false }
func (n *noopSegmentCache) Set(messageID string, data SegmentData)   {}
