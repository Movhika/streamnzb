package search

import (
	"context"
	"sync"
	"testing"

	"streamnzb/pkg/indexer"
)

type recordingIndexer struct {
	mu   sync.Mutex
	name string
	reqs []indexer.SearchRequest
}

func (r *recordingIndexer) Search(req indexer.SearchRequest) (*indexer.SearchResponse, error) {
	r.mu.Lock()
	r.reqs = append(r.reqs, req)
	r.mu.Unlock()
	return &indexer.SearchResponse{}, nil
}

func (r *recordingIndexer) DownloadNZB(context.Context, string) ([]byte, error) { return nil, nil }
func (r *recordingIndexer) Ping() error                                         { return nil }
func (r *recordingIndexer) Name() string                                        { return r.name }
func (r *recordingIndexer) GetUsage() indexer.Usage                             { return indexer.Usage{} }

type staticIndexer struct {
	name string
	resp *indexer.SearchResponse
}

func (s *staticIndexer) Search(req indexer.SearchRequest) (*indexer.SearchResponse, error) {
	return s.resp, nil
}

func (s *staticIndexer) DownloadNZB(context.Context, string) ([]byte, error) { return nil, nil }
func (s *staticIndexer) Ping() error                                         { return nil }
func (s *staticIndexer) Name() string                                        { return s.name }
func (s *staticIndexer) GetUsage() indexer.Usage                             { return indexer.Usage{} }

func TestRunIndexerSearchesPerIndexerTextRequestCarriesSeasonEpisodeWhenEnabled(t *testing.T) {
	idx := &recordingIndexer{name: "TestIndexer"}
	req := indexer.SearchRequest{
		Cat:                    "5000",
		Limit:                  100,
		IMDbID:                 "tt1234567",
		Season:                 "1",
		Episode:                "5",
		UseSeasonEpisodeParams: true,
		PerIndexerQuery: map[string][]string{
			"TestIndexer": {"The Walking Dead"},
		},
	}

	if _, err := RunIndexerSearches(idx, nil, req, "series", nil, "", "", nil); err != nil {
		t.Fatalf("RunIndexerSearches() error = %v", err)
	}

	if len(idx.reqs) != 1 {
		t.Fatalf("expected 1 Search call, got %d", len(idx.reqs))
	}

	var textReq *indexer.SearchRequest
	for i := range idx.reqs {
		reqCopy := idx.reqs[i]
		if reqCopy.PerIndexerQuery != nil {
			textReq = &reqCopy
		}
	}

	if textReq == nil {
		t.Fatal("expected a text search request")
	}
	if textReq.Season != "1" || textReq.Episode != "5" {
		t.Fatalf("expected text request to keep season/episode when enabled, got season=%q episode=%q", textReq.Season, textReq.Episode)
	}
	if got := textReq.PerIndexerQuery["TestIndexer"]; len(got) != 1 || got[0] != "The Walking Dead" {
		t.Fatalf("expected text request queries to be preserved, got %#v", got)
	}
}

func TestRunIndexerSearchesSkipsPostFilterWhenDisabled(t *testing.T) {
	idx := &staticIndexer{
		name: "SceneNZBs",
		resp: &indexer.SearchResponse{
			Channel: indexer.Channel{
				Items: []indexer.Item{
					{Title: "Der.Patriot.2000.German.DL.1080p.BluRay.x264.iNTERNAL-VideoStar", ActualIndexer: "SceneNZBs"},
				},
			},
		},
	}
	req := indexer.SearchRequest{
		Cat:         "2100",
		IMDbID:      "tt0187393",
		TMDBID:      "2024",
		FilterQuery: "The Patriot 2000",
		PerIndexerQuery: map[string][]string{
			"SceneNZBs": {"Der Patriot 2000"},
		},
		DisableResultFiltering: true,
	}

	got, err := RunIndexerSearches(idx, nil, req, "movie", nil, "", "", nil)
	if err != nil {
		t.Fatalf("RunIndexerSearches() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 result with post-filter disabled, got %d: %+v", len(got), got)
	}
}

func TestRunIndexerSearchesQueryWithIDsDoesNotAlsoRunIDSearch(t *testing.T) {
	idx := &recordingIndexer{name: "TestIndexer"}
	req := indexer.SearchRequest{
		Query:  "Meal Ticket 2026",
		Cat:    "2000",
		Limit:  100,
		IMDbID: "tt40232255",
		TMDBID: "1649758",
	}

	if _, err := RunIndexerSearches(idx, nil, req, "movie", nil, "", "", nil); err != nil {
		t.Fatalf("RunIndexerSearches() error = %v", err)
	}

	if len(idx.reqs) != 1 {
		t.Fatalf("expected exactly 1 Search call, got %d", len(idx.reqs))
	}
	if idx.reqs[0].SearchMode != "text" {
		t.Fatalf("expected query request to remain text-only, got mode %q", idx.reqs[0].SearchMode)
	}
	if idx.reqs[0].Query != "Meal Ticket 2026" {
		t.Fatalf("expected text query to be preserved, got %q", idx.reqs[0].Query)
	}
}
