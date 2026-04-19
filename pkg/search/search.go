package search

import (
	"fmt"
	"strconv"
	"strings"

	"streamnzb/pkg/core/config"
	"streamnzb/pkg/core/logger"
	"streamnzb/pkg/indexer"
	"streamnzb/pkg/release"
	"streamnzb/pkg/session"
)

type TMDBResolver interface {
	GetMovieTitle(imdbID, tmdbID string) (string, error)
	GetMovieTitleAndYear(imdbID, tmdbID string) (title, year string, err error)
	GetMovieTitleForSearch(imdbID, tmdbID, language string, includeYear, normalize bool) (string, error)
	GetTVShowName(tmdbID, imdbID string) (string, error)
}

type SearchConfig interface {
	GetSearchTitleLanguage() string
}

// BuildValidationQuery is a fallback that derives the expected title (and year
// for movies) from TMDB metadata when the request did not already provide a
// prepared ValidationQuery. The returned string is empty when TMDB metadata is
// unavailable.
func BuildValidationQuery(tmdbClient TMDBResolver, req indexer.SearchRequest, contentType string, contentIDs *session.AvailReportMeta, imdbForText, tmdbForText string) string {
	if tmdbClient == nil {
		return ""
	}
	if contentType == "movie" && contentIDs != nil {
		if t, y, err := tmdbClient.GetMovieTitleAndYear(contentIDs.ImdbID, req.TMDBID); err == nil && t != "" {
			if y != "" {
				return t + " " + y
			}
			return t
		}
	} else if contentType == "series" {
		if name, err := tmdbClient.GetTVShowName(tmdbForText, imdbForText); err == nil && name != "" {
			seasonNum, seasonErr := strconv.Atoi(req.Season)
			epNum, episodeErr := strconv.Atoi(req.Episode)
			switch {
			case req.Season != "" && req.Episode != "":
				if seasonErr == nil && episodeErr == nil {
					return fmt.Sprintf("%s S%02dE%02d", name, seasonNum, epNum)
				}
				return fmt.Sprintf("%s S%sE%s", name, req.Season, req.Episode)
			case req.Season != "":
				if seasonNum > 0 {
					return fmt.Sprintf("%s S%02d", name, seasonNum)
				}
				return fmt.Sprintf("%s S%s", name, req.Season)
			}
			return name
		}
	}
	return ""
}

func compactValidationQueryForLog(query string) string {
	query = strings.Join(strings.Fields(strings.TrimSpace(query)), " ")
	if len(query) <= 96 {
		return query
	}
	return strings.TrimSpace(query[:93]) + "..."
}

func RunIndexerSearches(idx indexer.Indexer, tmdbClient TMDBResolver, req indexer.SearchRequest, contentType string, contentIDs *session.AvailReportMeta, imdbForText, tmdbForText string, cfg SearchConfig) ([]*release.Release, error) {
	if idx == nil {
		return nil, nil
	}
	_ = cfg

	runIDSearch := strings.EqualFold(strings.TrimSpace(req.SearchMode), "id")
	searchReq := req

	validationQuery := req.ValidationQuery
	if req.EnableResultValidation && (req.EnableTitleValidation || req.EnableYearValidation) && validationQuery == "" {
		validationQuery = BuildValidationQuery(tmdbClient, req, contentType, contentIDs, imdbForText, tmdbForText)
	}

	if !runIDSearch && strings.TrimSpace(req.Query) != "" {
		searchReq.SearchMode = "text"
	}

	filterAggregator := func(base indexer.Indexer, request indexer.SearchRequest, textMode bool) indexer.Indexer {
		agg, ok := base.(*indexer.Aggregator)
		if !ok {
			return base
		}
		filtered := make([]indexer.Indexer, 0, len(agg.GetIndexers()))
		for _, idxr := range agg.GetIndexers() {
			var overrides *config.IndexerSearchConfig
			if request.EffectiveByIndexer != nil {
				overrides = request.EffectiveByIndexer[idxr.Name()]
			}
			reqCopy := request
			reqCopy.EffectiveByIndexer = nil
			reqCopy.OptionalOverrides = overrides
			if textMode && reqCopy.Query != "" {
				reqCopy.SearchMode = "text"
			}
			if indexer.ShouldSkipIndexerForRequest(reqCopy, overrides) {
				continue
			}
			filtered = append(filtered, idxr)
		}
		if len(filtered) == 0 {
			return nil
		}
		return indexer.NewAggregator(filtered...)
	}

	// NOTE: Per-indexer DisableIdSearch / DisableStringSearch flags are enforced
	// inside the Aggregator.Search method as a fallback, but we prefilter here so only
	// relevant indexers participate in each request path.

	idxForMode := filterAggregator(idx, searchReq, !runIDSearch)
	if idxForMode == nil {
		return nil, nil
	}

	resp, err := idxForMode.Search(searchReq)
	if err != nil {
		mode := "text"
		if runIDSearch {
			mode = "id"
		}
		logger.Warn("Stream search failed",
			"stream", req.StreamLabel,
			"request", req.RequestLabel,
			"mode", mode,
			"err", err,
		)
		return nil, fmt.Errorf("%s search failed for stream=%s request=%s: %w", mode, req.StreamLabel, req.RequestLabel, err)
	}
	indexer.NormalizeSearchResponse(resp)

	rawReleases := resp.Releases
	var releases []*release.Release
	for _, rel := range rawReleases {
		if rel != nil {
			if runIDSearch {
				rel.QuerySource = "id"
			} else {
				rel.QuerySource = "text"
			}
			releases = append(releases, rel)
		}
	}

	var validationStats ValidationStats
	if req.EnableResultValidation {
		releases, validationStats = ValidateSearchResultsWithStats(releases, contentType, validationQuery, req.Season, req.Episode, req.EnableTitleValidation, req.EnableYearValidation)
		validationAttrs := []any{
			"stream", req.StreamLabel,
			"request", req.RequestLabel,
			"mode", func() string {
				if runIDSearch {
					return "id"
				}
				return "text"
			}(),
			"type", contentType,
			"validation_query", compactValidationQueryForLog(validationQuery),
			"validation_query_length", len(strings.TrimSpace(validationQuery)),
			"title_validation", validationStats.TitleValidationApplied,
			"year_validation", validationStats.YearValidationApplied,
			"expected_title", validationStats.ExpectedTitle,
			"expected_year", validationStats.ExpectedYear,
			"raw_results", validationStats.RawResults,
			"final_results", validationStats.FinalResults,
			"rejected_results", validationStats.RejectedResults,
			"dropped_title", validationStats.DroppedTitle,
			"dropped_episode_request", validationStats.DroppedEpisodeRequest,
			"dropped_season", validationStats.DroppedSeason,
			"dropped_year", validationStats.DroppedYear,
			"accepted_exact_episode", validationStats.AcceptedExactEpisode,
			"accepted_multi_episode", validationStats.AcceptedMultiEpisode,
			"accepted_season_pack", validationStats.AcceptedSeasonPack,
			"accepted_complete_pack", validationStats.AcceptedCompletePack,
			"accepted_season_match", validationStats.AcceptedSeasonMatch,
		}
		if contentType == "series" {
			validationAttrs = append(validationAttrs,
				"scope", config.NormalizeSeriesSearchScope(req.SeriesSearchScope, nil),
				"expected_season", validationStats.ExpectedSeason,
				"expected_episode", validationStats.ExpectedEpisode,
			)
		}
		logger.Debug("Search request validation", validationAttrs...)
	}

	switch {
	case !runIDSearch:
		logger.Debug("Search request finished",
			"stream", req.StreamLabel,
			"request", req.RequestLabel,
			"mode", "text",
			"raw_results", len(rawReleases),
			"final_results", len(releases),
			"validation_enabled", req.EnableResultValidation,
		)
	default:
		logger.Debug("Search request finished",
			"stream", req.StreamLabel,
			"request", req.RequestLabel,
			"mode", "id",
			"raw_results", len(rawReleases),
			"final_results", len(releases),
			"validation_enabled", req.EnableResultValidation,
		)
	}
	return releases, nil
}
