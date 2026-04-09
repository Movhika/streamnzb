package search

import (
	"context"
	"fmt"
	"strings"
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

type errIndexer struct {
	name string
	err  error
}

func (e *errIndexer) Search(req indexer.SearchRequest) (*indexer.SearchResponse, error) {
	return nil, e.err
}

func (e *errIndexer) DownloadNZB(context.Context, string) ([]byte, error) { return nil, nil }
func (e *errIndexer) Ping() error                                         { return nil }
func (e *errIndexer) Name() string                                        { return e.name }
func (e *errIndexer) GetUsage() indexer.Usage                             { return indexer.Usage{} }

func TestRunIndexerSearchesTextRequestCarriesSeasonEpisodeWhenEnabled(t *testing.T) {
	idx := &recordingIndexer{name: "TestIndexer"}
	req := indexer.SearchRequest{
		Cat:                    "5000",
		Limit:                  100,
		IMDbID:                 "tt1234567",
		Season:                 "1",
		Episode:                "5",
		UseSeasonEpisodeParams: true,
		Query:                  "The Walking Dead",
	}

	if _, err := RunIndexerSearches(idx, nil, req, "series", nil, "", "", nil); err != nil {
		t.Fatalf("RunIndexerSearches() error = %v", err)
	}

	if len(idx.reqs) != 1 {
		t.Fatalf("expected 1 Search call, got %d", len(idx.reqs))
	}

	textReq := &idx.reqs[0]
	if textReq.Season != "1" || textReq.Episode != "5" {
		t.Fatalf("expected text request to keep season/episode when enabled, got season=%q episode=%q", textReq.Season, textReq.Episode)
	}
	if textReq.Query != "The Walking Dead" {
		t.Fatalf("expected text query to be preserved, got %q", textReq.Query)
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
		Cat:                    "2100",
		IMDbID:                 "tt0187393",
		TMDBID:                 "2024",
		Query:                  "Der Patriot 2000",
		FilterQuery:            "The Patriot 2000",
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

func TestRunIndexerSearchesIDModePreservesPreparedQuery(t *testing.T) {
	idx := &recordingIndexer{name: "TestIndexer"}
	req := indexer.SearchRequest{
		Cat:        "2000",
		Limit:      100,
		SearchMode: "id",
		IMDbID:     "tt1655441",
		TMDBID:     "1655441",
		Query:      "The Age of Adaline",
	}

	if _, err := RunIndexerSearches(idx, nil, req, "movie", nil, "", "", nil); err != nil {
		t.Fatalf("RunIndexerSearches() error = %v", err)
	}

	if len(idx.reqs) != 1 {
		t.Fatalf("expected exactly 1 Search call, got %d", len(idx.reqs))
	}
	if idx.reqs[0].SearchMode != "id" {
		t.Fatalf("expected id mode to stay id-only, got mode %q", idx.reqs[0].SearchMode)
	}
	if idx.reqs[0].Query != "The Age of Adaline" {
		t.Fatalf("expected id request to preserve prepared query, got %q", idx.reqs[0].Query)
	}
}

func TestRunIndexerSearchesReturnsTextSearchErrors(t *testing.T) {
	idx := &errIndexer{name: "BrokenIndexer", err: fmt.Errorf("backend unavailable")}
	req := indexer.SearchRequest{
		SearchMode:   "text",
		Query:        "The King Who Never Was",
		StreamLabel:  "TestStream",
		RequestLabel: "Text Request",
	}

	_, err := RunIndexerSearches(idx, nil, req, "series", nil, "", "", nil)
	if err == nil {
		t.Fatalf("expected text search error, got nil")
	}
	if got := err.Error(); got == "" || !strings.Contains(got, "text search failed") {
		t.Fatalf("expected wrapped text search error, got %q", got)
	}
}
