package stremio

import (
	"reflect"
	"testing"

	"streamnzb/pkg/auth"
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

func TestBuildMovieQueriesFromMetadataAddsGermanTransliterationVariant(t *testing.T) {
	metadata := &resolvedSearchMetadata{
		MovieDetails: &tmdb.MovieDetails{
			Title:            "The Lion King",
			OriginalTitle:    "The Lion King",
			OriginalLanguage: "en",
			ReleaseDate:      "1994-06-15",
		},
		MovieTranslations: &tmdb.MovieTranslationsResponse{
			Translations: []tmdb.MovieTranslationEntry{
				{
					ISO639_1:  "de",
					ISO3166_1: "DE",
					Data: tmdb.MovieTranslationData{
						Title: "König der Löwen",
					},
				},
			},
		},
	}

	got := buildMovieQueriesFromMetadata(metadata, "de-DE", false)
	want := []string{"Koenig der Loewen"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildMovieQueriesFromMetadata() = %#v, want %#v", got, want)
	}
}

func TestBuildSeriesQueriesFromMetadataAddsGermanTransliterationVariant(t *testing.T) {
	metadata := &resolvedSearchMetadata{
		TVDetails: &tmdb.TVDetails{
			Name:             "The Lion King",
			OriginalName:     "The Lion King",
			OriginalLanguage: "en",
			FirstAirDate:     "1994-09-10",
		},
		TVTranslations: &tmdb.TVTranslationsResponse{
			Translations: []tmdb.TVTranslationEntry{
				{
					ISO639_1:  "de",
					ISO3166_1: "DE",
					Data: tmdb.TVTranslationData{
						Name: "König der Löwen",
					},
				},
			},
		},
	}

	got := buildSeriesQueriesFromMetadata(metadata, "de-DE", false, "1", "2", false)
	want := []string{"Koenig der Loewen S01E02"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildSeriesQueriesFromMetadata() = %#v, want %#v", got, want)
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

	if params.Req.SearchMode != "id" {
		t.Fatalf("expected SearchMode to be id, got %q", params.Req.SearchMode)
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

	if params.Req.SearchMode != "text" {
		t.Fatalf("expected SearchMode to be text, got %q", params.Req.SearchMode)
	}
	if params.Req.Query != "" {
		t.Fatalf("expected query to stay empty before text query expansion, got %q", params.Req.Query)
	}
	if params.Req.Season != "1" || params.Req.Episode != "4" {
		t.Fatalf("expected base season/episode to remain available, got season=%q episode=%q", params.Req.Season, params.Req.Episode)
	}
}

func TestBuildSearchParamsBaseNumericIDMapsToTMDBID(t *testing.T) {
	srv := &Server{config: &config.Config{}}

	params, err := srv.buildSearchParamsBase("series", "123456:1:1", nil)
	if err != nil {
		t.Fatalf("buildSearchParamsBase() error = %v", err)
	}

	if params.Req.TMDBID != "123456" {
		t.Fatalf("expected numeric base ID to map to TMDB ID, got %q", params.Req.TMDBID)
	}
	if params.Req.IMDbID != "" {
		t.Fatalf("expected IMDb ID to stay empty for numeric base ID, got %q", params.Req.IMDbID)
	}
	if params.ContentIDs.Season != 1 || params.ContentIDs.Episode != 1 {
		t.Fatalf("expected content IDs to preserve season/episode, got season=%d episode=%d", params.ContentIDs.Season, params.ContentIDs.Episode)
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

func TestBuildRawSearchResultShortCircuitsWhenMetadataCannotBeResolved(t *testing.T) {
	srv := &Server{
		config: &config.Config{
			SeriesSearchQueries: []config.SearchQueryConfig{
				{Name: "TVQuery01", SearchMode: "id"},
			},
		},
	}
	stream := &auth.Stream{
		Username:            "Stream04",
		SeriesSearchQueries: []string{"TVQuery01"},
	}

	raw, err := srv.buildRawSearchResult(t.Context(), "tv", "stremevent_866", stream)
	if err != nil {
		t.Fatalf("buildRawSearchResult() error = %v", err)
	}
	if raw == nil {
		t.Fatal("expected zero-result raw search result, got nil")
	}
	if len(raw.IndexerReleases) != 0 {
		t.Fatalf("expected no releases after metadata short-circuit, got indexer=%d", len(raw.IndexerReleases))
	}
}

func boolPtr(v bool) *bool {
	return &v
}
