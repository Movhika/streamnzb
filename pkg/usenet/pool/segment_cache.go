package pool

import (
	"container/list"
	"sync"
	"sync/atomic"
)

type SegmentData struct {
	Body []byte
	Size int64
}

// SegmentCacheBudget limits total segment cache memory across all sessions (0 = no limit).
type SegmentCacheBudget struct {
	maxBytes int64
	current  atomic.Int64
}

// NewSegmentCacheBudget creates a budget of maxMB megabytes (0 = no limit).
func NewSegmentCacheBudget(maxMB int) *SegmentCacheBudget {
	if maxMB <= 0 {
		return nil
	}
	return &SegmentCacheBudget{maxBytes: int64(maxMB) * 1024 * 1024}
}

// Reserve adds n bytes to current usage if under the cap. Returns true if reserved.
func (b *SegmentCacheBudget) Reserve(n int64) bool {
	if b == nil || n <= 0 {
		return true
	}
	for {
		c := b.current.Load()
		if c+n <= b.maxBytes && b.current.CompareAndSwap(c, c+n) {
			return true
		}
		if c+n > b.maxBytes {
			return false
		}
	}
}

// Release subtracts n bytes from current usage.
func (b *SegmentCacheBudget) Release(n int64) {
	if b == nil || n <= 0 {
		return
	}
	b.current.Add(-n)
}

type SegmentCache interface {
	Get(messageID string) (SegmentData, bool)
	Set(messageID string, data SegmentData)
}

const DefaultSegmentCacheCapacity = 1024

func NewMemorySegmentCache() SegmentCache {
	return NewMemorySegmentCacheWithBudget(nil)
}

func NewMemorySegmentCacheWithBudget(budget *SegmentCacheBudget) SegmentCache {
	return &memorySegmentCache{
		budget: budget,
		m:      make(map[string]*list.Element),
		lru:    list.New(),
	}
}

func NewMemorySegmentCacheWithCapacity(capacity int) SegmentCache {
	return &memorySegmentCache{
		capacity: capacity,
		m:        make(map[string]*list.Element),
		lru:      list.New(),
	}
}

type cacheEntry struct {
	key  string
	data SegmentData
}

type memorySegmentCache struct {
	mu       sync.Mutex
	budget   *SegmentCacheBudget
	capacity int // fallback/legacy limit if budget is nil
	m        map[string]*list.Element
	lru      *list.List
}

func (c *memorySegmentCache) Get(messageID string) (SegmentData, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if el, ok := c.m[messageID]; ok {
		c.lru.MoveToFront(el)
		return el.Value.(*cacheEntry).data, true
	}
	return SegmentData{}, false
}

func (c *memorySegmentCache) Set(messageID string, data SegmentData) {
	c.mu.Lock()
	defer c.mu.Unlock()

	size := int64(len(data.Body))

	if el, ok := c.m[messageID]; ok {
		oldSize := int64(len(el.Value.(*cacheEntry).data.Body))
		if c.budget != nil {
			c.budget.Release(oldSize)
			reserved := c.budget.Reserve(size)
			if !reserved {
				// If we can't reserve new size, evict and try again
				c.lru.MoveToFront(el)
				c.evictLocked()
				for c.lru.Len() > 0 && !reserved {
					c.evictLocked()
					reserved = c.budget.Reserve(size)
				}
				if !reserved {
					return // cannot cache
				}
			}
		}
		c.lru.MoveToFront(el)
		el.Value.(*cacheEntry).data = data
		return
	}

	if c.budget != nil {
		reserved := c.budget.Reserve(size)
		if !reserved {
			// Evict until we can reserve
			for c.lru.Len() > 0 && !reserved {
				c.evictLocked()
				reserved = c.budget.Reserve(size)
			}
			if !reserved {
				return // too large or failed to reserve
			}
		}
	} else if c.capacity > 0 && c.lru.Len() >= c.capacity {
		c.evictLocked()
	} else if c.capacity <= 0 && c.lru.Len() >= DefaultSegmentCacheCapacity {
		c.evictLocked()
	}

	ent := &cacheEntry{key: messageID, data: data}
	el := c.lru.PushFront(ent)
	c.m[messageID] = el
}

func (c *memorySegmentCache) evictLocked() {
	el := c.lru.Back()
	if el == nil {
		return
	}
	ent := el.Value.(*cacheEntry)
	if c.budget != nil {
		c.budget.Release(int64(len(ent.data.Body)))
	}
	delete(c.m, ent.key)
	c.lru.Remove(el)
}

func NoopSegmentCache() SegmentCache { return &noopSegmentCache{} }

type noopSegmentCache struct{}

func (n *noopSegmentCache) Get(messageID string) (SegmentData, bool) { return SegmentData{}, false }
func (n *noopSegmentCache) Set(messageID string, data SegmentData)   {}
