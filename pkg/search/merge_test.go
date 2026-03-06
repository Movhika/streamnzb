package search

import (
	"testing"

	"streamnzb/pkg/core/logger"
	"streamnzb/pkg/release"
)

func TestNormalizedTitleMatches(t *testing.T) {
	logger.Init("ERROR")

	tests := []struct {
		expect   string
		gotTitle string
		want     bool
	}{
		{"Law & Order", "Law and Order", true},
		{"Law & Order", "Law and Order SVU", true},
		{"Star Trek: Starfleet Academy", "Star.Trek.Starfleet.Academy.S01E01", true},
		{"Star Trek: Starfleet Academy", "Starfleet Academy S01E01", false},
		{"The Walking Dead", "The Walking Dead S06E07", true},
		{"Some Show", "Other Show", false},
		{"Law and Order", "Law & Order", true},
		{"Show 2024", "Show 2024 1080p", true},
		{"Interstellar", "The.Science.of.Interstellar", false},
		{"Interstellar", "Interstellar.2014.2160p.BluRay", true},
	}
	for _, tt := range tests {
		got := normalizedTitleMatches(tt.expect, tt.gotTitle)
		if got != tt.want {
			t.Errorf("normalizedTitleMatches(%q, %q) = %v, want %v", tt.expect, tt.gotTitle, got, tt.want)
		}
	}
}

func TestFilterResultsSeriesEpisodeRequestAcceptsPacks(t *testing.T) {
	logger.Init("ERROR")

	releases := []*release.Release{
		{Title: "The.Walking.Dead.S01E05.1080p.WEB-DL"},
		{Title: "The.Walking.Dead.S01E05E06.1080p.WEB-DL"},
		{Title: "The.Walking.Dead.S01.COMPLETE.1080p.WEB-DL"},
		{Title: "The.Walking.Dead.Complete.Series.1080p.WEB-DL"},
		{Title: "The.Walking.Dead.S02.COMPLETE.1080p.WEB-DL"},
		{Title: "Other.Show.S01E05.1080p.WEB-DL"},
	}

	filtered := FilterResults(releases, "series", "The Walking Dead S01E05", "1", "5")
	got := make([]string, 0, len(filtered))
	for _, rel := range filtered {
		if rel != nil {
			got = append(got, rel.Title)
		}
	}

	want := []string{
		"The.Walking.Dead.S01E05.1080p.WEB-DL",
		"The.Walking.Dead.S01E05E06.1080p.WEB-DL",
		"The.Walking.Dead.S01.COMPLETE.1080p.WEB-DL",
		"The.Walking.Dead.Complete.Series.1080p.WEB-DL",
	}

	if len(got) != len(want) {
		t.Fatalf("FilterResults() returned %d releases, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("FilterResults()[%d] = %q, want %q (all: %v)", i, got[i], want[i], got)
		}
	}
}

func TestFilterResultsSeriesEpisodeRequestRejectsWrongEpisodePacks(t *testing.T) {
	logger.Init("ERROR")

	releases := []*release.Release{
		{Title: "The.Walking.Dead.S01E06.1080p.WEB-DL"},
		{Title: "The.Walking.Dead.S01E06E07.1080p.WEB-DL"},
		{Title: "The.Walking.Dead.S02.COMPLETE.1080p.WEB-DL"},
	}

	filtered := FilterResults(releases, "series", "The Walking Dead S01E05", "1", "5")
	if len(filtered) != 0 {
		t.Fatalf("expected no matches, got %d: %+v", len(filtered), filtered)
	}
}
