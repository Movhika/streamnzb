package session

import (
	"context"
	"testing"
	"time"

	"streamnzb/pkg/core/logger"
	"streamnzb/pkg/indexer"
	"streamnzb/pkg/media/loader"
	"streamnzb/pkg/media/nzb"
	"streamnzb/pkg/release"
)

type fakeIndexer struct{}

func (fakeIndexer) Search(indexer.SearchRequest) (*indexer.SearchResponse, error) { return nil, nil }
func (fakeIndexer) DownloadNZB(context.Context, string) ([]byte, error)           { return nil, nil }
func (fakeIndexer) Ping() error                                                   { return nil }
func (fakeIndexer) Name() string                                                  { return "fake" }
func (fakeIndexer) GetUsage() indexer.Usage                                       { return indexer.Usage{} }

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
		indexer:     fakeIndexer{},
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
