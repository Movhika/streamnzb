package indexer

import (
	"context"
	"testing"
	"time"
)

type testIndexer struct {
	name     string
	searchFn func(req SearchRequest) (*SearchResponse, error)
}

func (t *testIndexer) Search(req SearchRequest) (*SearchResponse, error) { return t.searchFn(req) }
func (t *testIndexer) DownloadNZB(ctx context.Context, nzbURL string) ([]byte, error) {
	return nil, nil
}
func (t *testIndexer) Ping() error     { return nil }
func (t *testIndexer) Name() string    { return t.name }
func (t *testIndexer) GetUsage() Usage { return Usage{} }

func TestAggregatorSearchRunsPerIndexerQueriesInParallel(t *testing.T) {
	started := make(chan string, 3)
	release := make(chan struct{})
	idx := &testIndexer{
		name: "one",
		searchFn: func(req SearchRequest) (*SearchResponse, error) {
			started <- req.Query
			<-release
			return &SearchResponse{Channel: Channel{Items: []Item{{Title: req.Query, Size: int64(len(req.Query))}}}}, nil
		},
	}
	agg := NewAggregator(idx)
	done := make(chan *SearchResponse, 1)
	errCh := make(chan error, 1)

	go func() {
		resp, err := agg.Search(SearchRequest{PerIndexerQuery: map[string][]string{"one": {"q1", "q22", "q333"}}})
		if err != nil {
			errCh <- err
			return
		}
		done <- resp
	}()

	for i := 0; i < 3; i++ {
		select {
		case <-started:
		case err := <-errCh:
			t.Fatalf("Search() returned error: %v", err)
		case <-time.After(500 * time.Millisecond):
			close(release)
			t.Fatalf("expected all per-indexer queries to start in parallel; only saw %d", i)
		}
	}
	close(release)

	select {
	case err := <-errCh:
		t.Fatalf("Search() returned error: %v", err)
	case resp := <-done:
		if resp == nil || len(resp.Channel.Items) != 3 {
			t.Fatalf("expected 3 items, got %#v", resp)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Search() result")
	}
}
