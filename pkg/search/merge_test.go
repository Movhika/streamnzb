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
		{"Batman", "The Batman", false},
		{"Batman", "Batman Beyond", false},
		{"Batman", "Batman Forever", false},
		{"The Hunger Games: Mockingjay - Part 1", "The Hunger Games Mockingjay Part 2", false},
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
		{Title: "The.Walking.Dead.Season.01.1080p.WEB-DL"},
		{Title: "The.Walking.Dead.Season.1.1080p.WEB-DL"},
		{Title: "The.Walking.Dead.S01.1080p.WEB-DL"},
		{Title: "The.Walking.Dead.Complete.Series.1080p.WEB-DL"},
		{Title: "The.Walking.Dead.COMPLETE.1080p.WEB-DL"},
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
		"The.Walking.Dead.Season.01.1080p.WEB-DL",
		"The.Walking.Dead.Season.1.1080p.WEB-DL",
		"The.Walking.Dead.S01.1080p.WEB-DL",
		"The.Walking.Dead.Complete.Series.1080p.WEB-DL",
		"The.Walking.Dead.COMPLETE.1080p.WEB-DL",
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

func TestFilterResultsSeriesEpisodeRequestKeepsSTitlesIntact(t *testing.T) {
	logger.Init("ERROR")

	releases := []*release.Release{
		{Title: "Star.Trek.Strange.New.Worlds.S01E01.WEBRip.x265-ION265"},
		{Title: "Star.Trek.Starfleet.Academy.S01E01.WEBRip.x265-ION265"},
	}

	filtered := FilterResults(releases, "series", "Star Trek: Strange New Worlds S01E01", "1", "1")
	if len(filtered) != 1 {
		t.Fatalf("expected 1 match, got %d: %+v", len(filtered), filtered)
	}
	if filtered[0].Title != releases[0].Title {
		t.Fatalf("expected %q, got %q", releases[0].Title, filtered[0].Title)
	}
}

func TestFilterResultsSeriesEpisodeRequestRejectsSingleWordTitleVariants(t *testing.T) {
	logger.Init("ERROR")

	releases := []*release.Release{
		{Title: "Batman.S01E02.1080p.WEB-DL"},
		{Title: "The.Batman.S01E02.1080p.WEB-DL"},
		{Title: "Batman.Beyond.S01E02.1080p.WEB-DL"},
	}

	filtered := FilterResults(releases, "series", "Batman S01E02", "1", "2")
	if len(filtered) != 1 {
		t.Fatalf("expected 1 match, got %d: %+v", len(filtered), filtered)
	}
	if filtered[0].Title != releases[0].Title {
		t.Fatalf("expected %q, got %q", releases[0].Title, filtered[0].Title)
	}
}

func TestFilterResultsMovieRejectsNumberedTitleVariants(t *testing.T) {
	logger.Init("ERROR")

	releases := []*release.Release{
		{Title: "The.Hunger.Games.Mockingjay.Part.1.2014.2160p.UHD.BluRay.x265-TERMiNAL"},
		{Title: "The.Hunger.Games.Mockingjay.Part.2.2015.2160p.UHD.BluRay.x265-TERMiNAL"},
	}

	filtered := FilterResults(releases, "movie", "The Hunger Games: Mockingjay - Part 1 2014", "", "")
	if len(filtered) != 1 {
		t.Fatalf("expected 1 match, got %d: %+v", len(filtered), filtered)
	}
	if filtered[0].Title != releases[0].Title {
		t.Fatalf("expected %q, got %q", releases[0].Title, filtered[0].Title)
	}
}

func TestFilterResultsMovieYearRange(t *testing.T) {
	logger.Init("ERROR")

	releases := []*release.Release{
		{Title: "Batman.1080p.BluRay"},
		{Title: "Batman.1993.1080p.BluRay"},
		{Title: "Batman.1994.1080p.BluRay"},
		{Title: "Batman.1995.1080p.BluRay"},
		{Title: "Batman.2026.1080p.BluRay"},
		{Title: "Other.Movie.1994.1080p.BluRay"},
	}

	filtered := FilterResults(releases, "movie", "Batman 1994", "", "")
	got := make([]string, 0, len(filtered))
	for _, rel := range filtered {
		if rel != nil {
			got = append(got, rel.Title)
		}
	}

	want := []string{
		"Batman.1080p.BluRay",
		"Batman.1993.1080p.BluRay",
		"Batman.1994.1080p.BluRay",
		"Batman.1995.1080p.BluRay",
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
