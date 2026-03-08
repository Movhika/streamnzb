package session

import (
	"context"
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"streamnzb/pkg/core/logger"
	"streamnzb/pkg/indexer"
	"streamnzb/pkg/media/loader"
	"streamnzb/pkg/media/nzb"
	"streamnzb/pkg/release"
)

type fakeIndexer struct {
	data     []byte
	err      error
	calls    int
	lastURL  string
	typeName string
}

func (*fakeIndexer) Search(indexer.SearchRequest) (*indexer.SearchResponse, error) { return nil, nil }
func (f *fakeIndexer) DownloadNZB(_ context.Context, rawURL string) ([]byte, error) {
	f.calls++
	f.lastURL = rawURL
	return f.data, f.err
}
func (*fakeIndexer) Ping() error             { return nil }
func (*fakeIndexer) Name() string            { return "fake" }
func (*fakeIndexer) GetUsage() indexer.Usage { return indexer.Usage{} }
func (f *fakeIndexer) Type() string {
	if f.typeName != "" {
		return f.typeName
	}
	return "newznab"
}

func TestSessionCloseClearsHeavyReferences(t *testing.T) {
	logger.Init("ERROR")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	file := loader.NewFile(ctx, &nzb.File{Subject: "test.mkv"}, nil, nil, nil, nil)
	s := &Session{
		ID:          "sess-1",
		NZB:         &nzb.NZB{Files: []nzb.File{{Subject: "video.mkv", Segments: []nzb.Segment{{ID: "<a>", Bytes: 10}}}}},
		Files:       []*loader.File{file},
		File:        file,
		Blueprint:   &struct{ Data []byte }{Data: make([]byte, 1024)},
		Release:     &release.Release{Title: "release"},
		ContentIDs:  &AvailReportMeta{ImdbID: "tt123"},
		Clients:     map[string]time.Time{"127.0.0.1": time.Now()},
		downloadURL: "https://example.invalid/nzb",
		indexer:     &fakeIndexer{},
		cancel:      cancel,
	}

	s.Close()

	if s.NZB != nil || s.Blueprint != nil || s.Release != nil || s.ContentIDs != nil {
		t.Fatalf("expected heavy session references to be cleared")
	}
	if s.Files != nil || s.File != nil || s.Clients != nil {
		t.Fatalf("expected loader and client references to be cleared")
	}
	if s.downloadURL != "" || s.indexer != nil || s.cancel != nil {
		t.Fatalf("expected deferred session state to be cleared")
	}
	select {
	case <-ctx.Done():
	default:
		t.Fatalf("expected Close to cancel session context")
	}
}

func TestCreateSessionAssignsFileOwners(t *testing.T) {
	logger.Init("ERROR")

	m := &Manager{
		sessions:  make(map[string]*Session),
		estimator: loader.NewSegmentSizeEstimator(),
	}
	nzbData := &nzb.NZB{Files: []nzb.File{{Subject: "video.mkv", Segments: []nzb.Segment{{ID: "<a>", Bytes: 10}}}}}

	s, err := m.CreateSession("sess-1", nzbData, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}
	if s.File == nil {
		t.Fatalf("expected session file to be created")
	}
	if got := s.File.OwnerSessionID(); got != "sess-1" {
		t.Fatalf("expected owner session id sess-1, got %q", got)
	}
	if got := m.sessionsWithFilesIDs(); len(got) != 1 || got[0] != "sess-1" {
		t.Fatalf("expected sessionsWithFilesIDs to return sess-1, got %v", got)
	}
	s.Close()
}

func TestCreateSessionSelectsRequestedEpisode(t *testing.T) {
	logger.Init("ERROR")

	m := &Manager{
		sessions:  make(map[string]*Session),
		estimator: loader.NewSegmentSizeEstimator(),
	}
	nzbData := &nzb.NZB{Files: []nzb.File{
		{Subject: "Show.S01E06.1080p.mkv", Segments: []nzb.Segment{{ID: "<a>", Bytes: 60}}},
		{Subject: "Show.S01E05.1080p.mkv", Segments: []nzb.Segment{{ID: "<b>", Bytes: 50}}},
	}}

	s, err := m.CreateSession("sess-target", nzbData, nil, &AvailReportMeta{Season: 1, Episode: 5})
	if err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}
	if got := s.File.Name(); !strings.Contains(got, "S01E05") {
		t.Fatalf("expected target episode file, got %q", got)
	}
	if len(s.Files) != 1 {
		t.Fatalf("expected one selected file, got %d", len(s.Files))
	}
	s.Close()
}

func TestGetOrDownloadNZBSelectsRequestedEpisode(t *testing.T) {
	logger.Init("ERROR")

	m := &Manager{
		sessions:  make(map[string]*Session),
		estimator: loader.NewSegmentSizeEstimator(),
	}
	data := marshalTestNZB(t, &nzb.NZB{Files: []nzb.File{
		{Subject: "Show.S01E06.1080p.mkv", Segments: []nzb.Segment{{ID: "<a>", Bytes: 60}}},
		{Subject: "Show.S01E05.1080p.mkv", Segments: []nzb.Segment{{ID: "<b>", Bytes: 50}}},
	}})
	idx := &fakeIndexer{data: data}
	s, err := m.CreateDeferredSession("sess-lazy", "https://example.invalid/get?nzb=1&apikey=test", nil, idx, &AvailReportMeta{Season: 1, Episode: 5}, "series", "tmdb:1:1:5")
	if err != nil {
		t.Fatalf("CreateDeferredSession returned error: %v", err)
	}
	if _, err := s.GetOrDownloadNZB(m); err != nil {
		t.Fatalf("GetOrDownloadNZB returned error: %v", err)
	}
	if idx.calls != 1 {
		t.Fatalf("expected DownloadNZB to be called once, got %d", idx.calls)
	}
	if s.File == nil {
		t.Fatal("expected session file after lazy load")
	}
	if got := s.File.Name(); !strings.Contains(got, "S01E05") {
		t.Fatalf("expected target episode file after lazy load, got %q", got)
	}
	if len(s.Files) != 1 {
		t.Fatalf("expected one selected file after lazy load, got %d", len(s.Files))
	}
	s.Close()
}

func TestGetOrDownloadNZBDownloadsKeylessURL(t *testing.T) {
	logger.Init("ERROR")

	m := &Manager{
		sessions:  make(map[string]*Session),
		estimator: loader.NewSegmentSizeEstimator(),
	}
	data := marshalTestNZB(t, &nzb.NZB{Files: []nzb.File{{
		Subject:  "Movie.2024.1080p.mkv",
		Segments: []nzb.Segment{{ID: "<a>", Bytes: 60}},
	}}})
	idx := &fakeIndexer{data: data, typeName: "newznab"}
	s, err := m.CreateDeferredSession("sess-keyless", "https://nzbfinder.ws/api?t=get&id=abc123", nil, idx, nil, "movie", "tt123")
	if err != nil {
		t.Fatalf("CreateDeferredSession returned error: %v", err)
	}
	if _, err := s.GetOrDownloadNZB(m); err != nil {
		t.Fatalf("GetOrDownloadNZB returned error: %v", err)
	}
	if idx.calls != 1 {
		t.Fatalf("expected DownloadNZB to be called once for keyless URL, got %d", idx.calls)
	}
	if idx.lastURL != "https://nzbfinder.ws/api?t=get&id=abc123" {
		t.Fatalf("DownloadNZB called with URL %q", idx.lastURL)
	}
	s.Close()
}

func marshalTestNZB(t *testing.T, doc *nzb.NZB) []byte {
	t.Helper()
	data, err := xml.Marshal(doc)
	if err != nil {
		t.Fatalf("xml.Marshal: %v", err)
	}
	return data
}
