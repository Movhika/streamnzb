package pool

import (
	"testing"
)

func TestNewPool_NoProviders(t *testing.T) {
	_, err := NewPool(&Config{Providers: nil})
	if err != ErrNoProvidersConfigured {
		t.Fatalf("expected ErrNoProvidersConfigured, got %v", err)
	}
	_, err = NewPool(&Config{Providers: []ProviderConfig{}})
	if err != ErrNoProvidersConfigured {
		t.Fatalf("expected ErrNoProvidersConfigured, got %v", err)
	}
}

func TestNewPool_NilCacheUsesNoop(t *testing.T) {

	c := NoopSegmentCache()
	data, ok := c.Get("any")
	if ok || data.Size != 0 || len(data.Body) != 0 {
		t.Fatalf("noop cache should never return hit")
	}
	c.Set("any", SegmentData{Body: []byte("x"), Size: 1})
	data, ok = c.Get("any")
	if ok {
		t.Fatalf("noop cache should not persist")
	}
}

func TestMemorySegmentCache_LRU(t *testing.T) {
	c := NewMemorySegmentCacheWithCapacity(2)
	c.Set("id1", SegmentData{Body: []byte("v1"), Size: 2})
	c.Set("id2", SegmentData{Body: []byte("v2"), Size: 2})

	_, ok := c.Get("id1")
	if !ok {
		t.Fatal("expected id1 to be present")
	}

	// Set id3, should evict id2 because id1 was accessed last
	c.Set("id3", SegmentData{Body: []byte("v3"), Size: 2})

	_, ok = c.Get("id1")
	if !ok {
		t.Fatal("expected id1 to be present (accessed last)")
	}
	_, ok = c.Get("id2")
	if ok {
		t.Fatal("expected id2 to be evicted")
	}
	_, ok = c.Get("id3")
	if !ok {
		t.Fatal("expected id3 to be present")
	}
}
func TestMemorySegmentCache_BudgetLRU(t *testing.T) {
	// Budget for 4 bytes
	budget := &SegmentCacheBudget{maxBytes: 4}

	c := NewMemorySegmentCacheWithBudget(budget)
	c.Set("id1", SegmentData{Body: []byte("v1"), Size: 2})
	c.Set("id2", SegmentData{Body: []byte("v2"), Size: 2})

	_, ok := c.Get("id1")
	if !ok {
		t.Fatal("expected id1 to be present")
	}

	// Set id3 (2 bytes), should evict id2 (oldest) to fit
	c.Set("id3", SegmentData{Body: []byte("v3"), Size: 2})

	_, ok = c.Get("id1")
	if !ok {
		t.Fatal("expected id1 to be present (accessed last)")
	}
	_, ok = c.Get("id2")
	if ok {
		t.Fatal("expected id2 to be evicted")
	}
	_, ok = c.Get("id3")
	if !ok {
		t.Fatal("expected id3 to be present")
	}
}
