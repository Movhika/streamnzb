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

func TestMemorySegmentCache(t *testing.T) {
	c := NewMemorySegmentCache()
	data, ok := c.Get("id1")
	if ok {
		t.Fatal("expected miss")
	}
	c.Set("id1", SegmentData{Body: []byte("hello"), Size: 5})
	data, ok = c.Get("id1")
	if !ok || data.Size != 5 || string(data.Body) != "hello" {
		t.Fatalf("expected hit: ok=%v size=%d body=%q", ok, data.Size, string(data.Body))
	}
}
