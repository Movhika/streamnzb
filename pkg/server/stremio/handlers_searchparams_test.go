package stremio

import (
	"reflect"
	"testing"

	"streamnzb/pkg/core/config"
	"streamnzb/pkg/indexer"
	"streamnzb/pkg/services/metadata/tmdb"
)

func TestBuildSeriesQueriesReturnsGenericShowName(t *testing.T) {
	got := buildSeriesQueries("Game of Thrones")
	want := []string{"Game of Thrones"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildSeriesQueries() = %#v, want %#v", got, want)
	}
}

func TestBuildSeriesQueriesWithOptionsCanIncludeYear(t *testing.T) {
	got := buildSeriesQueriesWithOptions("Game of Thrones", "2011", true)
	want := []string{"Game of Thrones 2011"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildSeriesQueriesWithOptions() = %#v, want %#v", got, want)
	}
}

func TestBuildSeriesQueriesWithOptionsCanOmitYear(t *testing.T) {
	got := buildSeriesQueriesWithOptions("Game of Thrones", "2011", false)
	want := []string{"Game of Thrones"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildSeriesQueriesWithOptions() = %#v, want %#v", got, want)
	}
}

func TestBuildSearchParamsFromBaseSeriesIDQueryModeMovesSeasonEpisodeIntoQuery(t *testing.T) {
	srv := &Server{config: &config.Config{}}
	base := &SearchParams{
		ContentType: "series",
		ID:          "tt1234567:1:4",
		Req: indexer.SearchRequest{
			Season:  "1",
			Episode: "4",
			IMDbID:  "tt1234567",
			Cat:     "5000",
			Limit:   1000,
		},
	}

	params, err := srv.buildSearchParamsFromBase(base, &config.SearchQueryConfig{
		SearchMode:             "id",
		UseSeasonEpisodeParams: boolPtr(false),
	})
	if err != nil {
		t.Fatalf("buildSearchParamsFromBase() error = %v", err)
	}

	if !params.Req.ForceIDSearch {
		t.Fatal("expected ForceIDSearch to be enabled")
	}
	if params.Req.Query != "S01E04" {
		t.Fatalf("expected S/E query suffix, got %q", params.Req.Query)
	}
	if params.Req.Season != "" || params.Req.Episode != "" {
		t.Fatalf("expected season/episode params to be cleared, got season=%q episode=%q", params.Req.Season, params.Req.Episode)
	}
}

func TestBuildSearchParamsFromBaseSeriesTextQueryModeKeepsSeasonEpisodeForLaterDispatchDecision(t *testing.T) {
	srv := &Server{config: &config.Config{}}
	base := &SearchParams{
		ContentType: "series",
		ID:          "tt1234567:1:4",
		Req: indexer.SearchRequest{
			Season:  "1",
			Episode: "4",
			IMDbID:  "tt1234567",
			Cat:     "5000",
			Limit:   1000,
		},
	}

	params, err := srv.buildSearchParamsFromBase(base, &config.SearchQueryConfig{
		SearchMode:             "text",
		UseSeasonEpisodeParams: boolPtr(false),
	})
	if err != nil {
		t.Fatalf("buildSearchParamsFromBase() error = %v", err)
	}

	if params.Req.ForceIDSearch {
		t.Fatal("expected text mode to keep ForceIDSearch disabled")
	}
	if params.Req.Query != "" {
		t.Fatalf("expected query to stay empty before text query expansion, got %q", params.Req.Query)
	}
	if params.Req.Season != "1" || params.Req.Episode != "4" {
		t.Fatalf("expected base season/episode to remain available, got season=%q episode=%q", params.Req.Season, params.Req.Episode)
	}
}

func TestHasResolvedIdentifiers(t *testing.T) {
	if hasResolvedIdentifiers(indexer.SearchRequest{}) {
		t.Fatal("expected empty request to report no resolved identifiers")
	}
	if !hasResolvedIdentifiers(indexer.SearchRequest{IMDbID: "tt1234567"}) {
		t.Fatal("expected IMDb ID to count as resolved identifier")
	}
	if !hasResolvedIdentifiers(indexer.SearchRequest{TMDBID: "123"}) {
		t.Fatal("expected TMDB ID to count as resolved identifier")
	}
	if !hasResolvedIdentifiers(indexer.SearchRequest{TVDBID: "456"}) {
		t.Fatal("expected TVDB ID to count as resolved identifier")
	}
}

func TestHasPreparedTextQueries(t *testing.T) {
	if hasPreparedTextQueries(indexer.SearchRequest{}) {
		t.Fatal("expected empty request to report no prepared text queries")
	}
	if !hasPreparedTextQueries(indexer.SearchRequest{Query: "Invincible"}) {
		t.Fatal("expected explicit query to count as prepared text query")
	}
	if !hasPreparedTextQueries(indexer.SearchRequest{FilterQuery: "Invincible S01E04"}) {
		t.Fatal("expected filter query to count as prepared text query")
	}
	if !hasPreparedTextQueries(indexer.SearchRequest{PerIndexerQuery: map[string][]string{"NzbPlanet": {"Invincible 2021"}}}) {
		t.Fatal("expected per-indexer query to count as prepared text query")
	}
	if hasPreparedTextQueries(indexer.SearchRequest{PerIndexerQuery: map[string][]string{"NzbPlanet": {"", "   "}}}) {
		t.Fatal("expected blank per-indexer queries not to count as prepared text query")
	}
}

func TestHasUsableResolvedMetadata(t *testing.T) {
	if hasUsableResolvedMetadata(nil, "series") {
		t.Fatal("expected nil params not to have usable resolved metadata")
	}
	if hasUsableResolvedMetadata(&SearchParams{}, "series") {
		t.Fatal("expected empty params not to have usable resolved metadata")
	}
	if !hasUsableResolvedMetadata(&SearchParams{Req: indexer.SearchRequest{IMDbID: "tt1234567"}}, "series") {
		t.Fatal("expected resolved identifier to count as usable metadata")
	}
	if !hasUsableResolvedMetadata(&SearchParams{
		Metadata: &resolvedSearchMetadata{
			TVDetails: &tmdb.TVDetails{Name: "Invincible"},
		},
	}, "series") {
		t.Fatal("expected resolved title to count as usable metadata")
	}
}

func boolPtr(v bool) *bool {
	return &v
}
