package search

import (
	"fmt"
	"strings"

	"streamnzb/pkg/core/config"
	"streamnzb/pkg/core/logger"
	"streamnzb/pkg/indexer"
	"streamnzb/pkg/release"
)

func compactValidationQueryForLog(query string) string {
	query = strings.Join(strings.Fields(strings.TrimSpace(query)), " ")
	if len(query) <= 96 {
		return query
	}
	return strings.TrimSpace(query[:93]) + "..."
}

func RunIndexerSearches(idx indexer.Indexer, req indexer.SearchRequest, contentType string) ([]*release.Release, error) {
	if idx == nil {
		return nil, nil
	}

	runIDSearch := strings.EqualFold(strings.TrimSpace(req.SearchMode), "id")
	searchReq := req

	validationQuery := req.ValidationQuery

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
	releases, validationStats = ValidateSearchResultsWithStats(releases, contentType, validationQuery, req.Season, req.Episode, true, req.EnableYearValidation)
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
		"raw_results", validationStats.RawResults,
		"final_results", validationStats.FinalResults,
		"rejected_results", validationStats.RejectedResults,
		"dropped_title", validationStats.DroppedTitle,
		"dropped_year", validationStats.DroppedYear,
	}
	if contentType == "series" {
		validationAttrs = append(validationAttrs,
			"scope", config.NormalizeSeriesSearchScope(req.SeriesSearchScope),
			"expected_season", validationStats.ExpectedSeason,
			"expected_episode", validationStats.ExpectedEpisode,
			"dropped_episode_request", validationStats.DroppedEpisodeRequest,
			"dropped_season", validationStats.DroppedSeason,
			"accepted_exact_episode", validationStats.AcceptedExactEpisode,
			"accepted_multi_episode", validationStats.AcceptedMultiEpisode,
			"accepted_season_pack", validationStats.AcceptedSeasonPack,
			"accepted_complete_pack", validationStats.AcceptedCompletePack,
			"accepted_season_match", validationStats.AcceptedSeasonMatch,
		)
	}
	logger.Debug("Search request validation", validationAttrs...)

	switch {
	case !runIDSearch:
		logger.Debug("Search request finished",
			"stream", req.StreamLabel,
			"request", req.RequestLabel,
			"mode", "text",
			"raw_results", len(rawReleases),
			"final_results", len(releases),
		)
	default:
		logger.Debug("Search request finished",
			"stream", req.StreamLabel,
			"request", req.RequestLabel,
			"mode", "id",
			"raw_results", len(rawReleases),
			"final_results", len(releases),
		)
	}
	return releases, nil
}
