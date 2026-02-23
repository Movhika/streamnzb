package search

import (
	"fmt"
	"strconv"
	"sync"

	"streamnzb/pkg/core/logger"
	"streamnzb/pkg/indexer"
	"streamnzb/pkg/release"
	"streamnzb/pkg/session"
)

// TMDBResolver resolves movie/TV titles for text search.
type TMDBResolver interface {
	GetMovieTitle(imdbID, tmdbID string) (string, error)
	GetMovieTitleAndYear(imdbID, tmdbID string) (title, year string, err error)
	GetMovieTitleForSearch(imdbID, tmdbID, language string, includeYear, normalize bool) (string, error)
	GetTVShowName(tmdbID, imdbID string) (string, error)
}

// SearchConfig provides global defaults for text search when PerIndexerQuery is not set.
type SearchConfig interface {
	GetIncludeYearInSearch() bool
	GetSearchTitleLanguage() string
	GetSearchTitleNormalize() bool
}

// RunIndexerSearches runs ID-based and text-based searches in parallel, merges and dedupes.
// When req.PerIndexerQuery is set, text search uses per-indexer queries and effective config; otherwise uses global cfg for title resolution.
func RunIndexerSearches(idx indexer.Indexer, tmdbClient TMDBResolver, req indexer.SearchRequest, contentType string, contentIDs *session.AvailReportMeta, imdbForText, tmdbForText string, cfg SearchConfig) ([]*release.Release, error) {
	idReq := req
	idReq.Query = ""
	idReq.PerIndexerQuery = nil // ID-only search: do not pass text query to indexers

	var textQuery string
	usePerIndexerQuery := len(req.PerIndexerQuery) > 0
	if !usePerIndexerQuery && tmdbClient != nil && cfg != nil {
		includeYear := cfg.GetIncludeYearInSearch()
		searchTitleLanguage := cfg.GetSearchTitleLanguage()
		searchTitleNormalize := cfg.GetSearchTitleNormalize()
		if contentType == "movie" {
			if searchTitleLanguage != "" || searchTitleNormalize {
				if q, err := tmdbClient.GetMovieTitleForSearch(contentIDs.ImdbID, req.TMDBID, searchTitleLanguage, includeYear, searchTitleNormalize); err == nil {
					textQuery = q
				}
			} else if includeYear {
				if t, y, err := tmdbClient.GetMovieTitleAndYear(contentIDs.ImdbID, req.TMDBID); err == nil {
					if y != "" {
						textQuery = t + " " + y
					} else {
						textQuery = t
					}
				}
			} else {
				if t, err := tmdbClient.GetMovieTitle(contentIDs.ImdbID, req.TMDBID); err == nil {
					textQuery = t
				}
			}
		} else if req.Season != "" && req.Episode != "" {
			if name, err := tmdbClient.GetTVShowName(tmdbForText, imdbForText); err == nil {
				seasonNum, _ := strconv.Atoi(req.Season)
				epNum, _ := strconv.Atoi(req.Episode)
				if seasonNum > 0 || epNum > 0 {
					textQuery = fmt.Sprintf("%s S%02dE%02d", name, seasonNum, epNum)
				} else {
					textQuery = fmt.Sprintf("%s S%sE%s", name, req.Season, req.Episode)
				}
			}
		}
	}

	var idResp *indexer.SearchResponse
	var idErr error
	var textReleases []*release.Release
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		idResp, idErr = idx.Search(idReq)
	}()
	if usePerIndexerQuery {
		wg.Add(1)
		textReq := indexer.SearchRequest{
			Cat:                req.Cat,
			Limit:              req.Limit,
			Season:             req.Season,
			Episode:            req.Episode,
			EffectiveByIndexer: req.EffectiveByIndexer,
			PerIndexerQuery:    req.PerIndexerQuery,
		}
		go func() {
			defer wg.Done()
			if resp, err := idx.Search(textReq); err == nil {
				indexer.NormalizeSearchResponse(resp)
				// Filter by content; per-indexer queries can differ (e.g. movie with/without year), so filter by each and dedupe
				var filtered [][]*release.Release
				for _, q := range req.PerIndexerQuery {
					filtered = append(filtered, FilterTextResultsByContent(resp.Releases, contentType, q, req.Season, req.Episode))
				}
				if len(filtered) > 0 {
					for _, list := range filtered {
						textReleases = append(textReleases, list...)
					}
					textReleases = MergeAndDedupeSearchResults(textReleases)
				}
			}
		}()
	} else if textQuery != "" {
		wg.Add(1)
		textReq := indexer.SearchRequest{Query: textQuery, Cat: req.Cat, Limit: req.Limit, Season: req.Season, Episode: req.Episode}
		go func() {
			defer wg.Done()
			if resp, err := idx.Search(textReq); err == nil {
				indexer.NormalizeSearchResponse(resp)
				textReleases = FilterTextResultsByContent(resp.Releases, contentType, textQuery, req.Season, req.Episode)
			}
		}()
	}
	wg.Wait()

	if idErr != nil {
		return nil, fmt.Errorf("indexer search failed: %w", idErr)
	}
	indexer.NormalizeSearchResponse(idResp)
	idReleases := make([]*release.Release, 0, len(idResp.Releases)+len(textReleases))
	for _, rel := range idResp.Releases {
		if rel != nil {
			rel.QuerySource = "id"
			idReleases = append(idReleases, rel)
		}
	}
	for _, rel := range textReleases {
		if rel != nil {
			rel.QuerySource = "text"
			idReleases = append(idReleases, rel)
		}
	}
	if len(textReleases) > 0 {
		logger.Debug("Indexer dual search", "id", len(idResp.Releases), "text", len(textReleases))
	}
	return MergeAndDedupeSearchResults(idReleases), nil
}
