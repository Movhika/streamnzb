package stremio

import (
	"reflect"
	"testing"
)

func TestBuildSeriesEpisodeQueriesIncludesPackDiscoveryShapes(t *testing.T) {
	got := buildSeriesEpisodeQueries("Game of Thrones", "1", "4")
	want := []string{
		"Game of Thrones S01E04",
		"Game of Thrones S01",
		"Game of Thrones Complete",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildSeriesEpisodeQueries() = %#v, want %#v", got, want)
	}
}

func TestBuildSeriesEpisodeQueriesWithOptionsCanDisableExtraSearches(t *testing.T) {
	got := buildSeriesEpisodeQueriesWithOptions("Game of Thrones", "", "1", "4", false, false, false)
	want := []string{"Game of Thrones S01E04"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildSeriesEpisodeQueriesWithOptions() = %#v, want %#v", got, want)
	}
}

func TestBuildSeriesEpisodeQueriesWithOptionsCanEnableSeasonSearchOnly(t *testing.T) {
	got := buildSeriesEpisodeQueriesWithOptions("Game of Thrones", "", "1", "4", false, true, false)
	want := []string{
		"Game of Thrones S01E04",
		"Game of Thrones S01",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildSeriesEpisodeQueriesWithOptions() = %#v, want %#v", got, want)
	}
}

func TestBuildSeriesEpisodeQueriesWithOptionsCanEnableCompleteSearchOnly(t *testing.T) {
	got := buildSeriesEpisodeQueriesWithOptions("Game of Thrones", "", "1", "4", false, false, true)
	want := []string{
		"Game of Thrones S01E04",
		"Game of Thrones Complete",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildSeriesEpisodeQueriesWithOptions() = %#v, want %#v", got, want)
	}
}

func TestBuildSeriesEpisodeQueriesWithOptionsCanIncludeYear(t *testing.T) {
	got := buildSeriesEpisodeQueriesWithOptions("Game of Thrones", "2011", "1", "4", true, true, true)
	want := []string{
		"Game of Thrones 2011 S01E04",
		"Game of Thrones 2011 S01",
		"Game of Thrones 2011 Complete",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildSeriesEpisodeQueriesWithOptions() = %#v, want %#v", got, want)
	}
}
