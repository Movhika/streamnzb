package parser

import "testing"

func TestParseReleaseTitleRetainsEpisodeCollections(t *testing.T) {
	parsed := ParseReleaseTitle("The.Walking.Dead.S01E05E06.1080p.WEB-DL")
	if parsed == nil {
		t.Fatal("expected parsed release")
	}
	if parsed.Season != 1 {
		t.Fatalf("expected first season 1, got %d", parsed.Season)
	}
	if parsed.Episode != 5 {
		t.Fatalf("expected first episode 5, got %d", parsed.Episode)
	}
	if !parsed.HasSeason(1) {
		t.Fatal("expected parsed release to include season 1")
	}
	if !parsed.HasEpisode(5) || !parsed.HasEpisode(6) {
		t.Fatalf("expected parsed release to include episodes 5 and 6, got %v", parsed.Episodes)
	}
}

func TestParsedReleaseEpisodeMatchRank(t *testing.T) {
	tests := []struct {
		name   string
		parsed *ParsedRelease
		want   int
	}{
		{name: "exact episode", parsed: &ParsedRelease{Season: 1, Episode: 5, Seasons: []int{1}, Episodes: []int{5}}, want: 4},
		{name: "multi episode", parsed: &ParsedRelease{Season: 1, Episode: 5, Seasons: []int{1}, Episodes: []int{5, 6}}, want: 3},
		{name: "season pack", parsed: &ParsedRelease{Season: 1, Seasons: []int{1}, Complete: true}, want: 2},
		{name: "show pack", parsed: &ParsedRelease{Complete: true}, want: 1},
		{name: "wrong season", parsed: &ParsedRelease{Season: 2, Seasons: []int{2}, Complete: true}, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.parsed.EpisodeMatchRank(1, 5); got != tt.want {
				t.Fatalf("EpisodeMatchRank() = %d, want %d", got, tt.want)
			}
		})
	}
}
