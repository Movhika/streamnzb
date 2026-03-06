package unpack

import "testing"

func TestSelectEpisodeCandidatePrefersExactEpisode(t *testing.T) {
	target := EpisodeTarget{Season: 1, Episode: 5}
	best, ok := selectEpisodeCandidate([]namedEpisodeCandidate{
		{Name: "Show.S01.COMPLETE.mkv", Size: 900, Order: 0},
		{Name: "Show.S01E05.mkv", Size: 500, Order: 1},
		{Name: "Show.S01E05E06.mkv", Size: 700, Order: 2},
	}, target)
	if !ok {
		t.Fatal("expected episode candidate match")
	}
	if best.Name != "Show.S01E05.mkv" {
		t.Fatalf("expected exact episode match, got %q", best.Name)
	}
}

func TestSelectMainFilePrefersRequestedEpisodeOverLargest(t *testing.T) {
	target := EpisodeTarget{Season: 1, Episode: 5}
	best := selectMainFile([]filePart{
		{name: "Show.S01E06.mkv", packedSize: 900, isMedia: true},
		{name: "Show.S01E05.mkv", packedSize: 500, isMedia: true},
	}, target)
	if best != "Show.S01E05.mkv" {
		t.Fatalf("expected requested episode file, got %q", best)
	}
}
