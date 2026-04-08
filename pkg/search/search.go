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

// BuildFilterQuery resolves the expected title (and year for movies) from TMDB
// metadata so that FilterResults can reject releases whose title doesn't match.
// The returned string is empty when TMDB metadata is unavailable.
func BuildFilterQuery(tmdbClient TMDBResolver, req indexer.SearchRequest, contentType string, contentIDs *session.AvailReportMeta, imdbForText, tmdbForText string) string {
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
	} else if contentType == "series" && req.Season != "" && req.Episode != "" {
		if name, err := tmdbClient.GetTVShowName(tmdbForText, imdbForText); err == nil && name != "" {
			seasonNum, _ := strconv.Atoi(req.Season)
			epNum, _ := strconv.Atoi(req.Episode)
			if seasonNum > 0 || epNum > 0 {
				return fmt.Sprintf("%s S%02dE%02d", name, seasonNum, epNum)
			}
			return fmt.Sprintf("%s S%sE%s", name, req.Season, req.Episode)
		}
	}
	return ""
}

func RunIndexerSearches(idx indexer.Indexer, tmdbClient TMDBResolver, req indexer.SearchRequest, contentType string, contentIDs *session.AvailReportMeta, imdbForText, tmdbForText string, cfg SearchConfig) ([]*release.Release, error) {
	if idx == nil {
		return nil, nil
	}
	_ = cfg

	runIDSearch := strings.EqualFold(strings.TrimSpace(req.SearchMode), "id")
	searchReq := req

	filterQuery := req.FilterQuery
	if filterQuery == "" {
		filterQuery = BuildFilterQuery(tmdbClient, req, contentType, contentIDs, imdbForText, tmdbForText)
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

	if filterQuery != "" && !req.DisableResultFiltering {
		releases = FilterResults(releases, contentType, filterQuery, req.Season, req.Episode)
	}

	switch {
	case !runIDSearch:
		logger.Debug("Search request finished",
			"stream", req.StreamLabel,
			"request", req.RequestLabel,
			"mode", "text",
			"raw_results", len(rawReleases),
		)
	default:
		logger.Debug("Search request finished",
			"stream", req.StreamLabel,
			"request", req.RequestLabel,
			"mode", "id",
			"raw_results", len(rawReleases),
		)
	}
	return releases, nil
}
