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
	idReq := req
	idReq.Query = ""
	idReq.PerIndexerQuery = nil

	var textReq *indexer.SearchRequest
	filterQuery := BuildFilterQuery(tmdbClient, req, contentType, contentIDs, imdbForText, tmdbForText)

	usePerIndexerQuery := len(req.PerIndexerQuery) > 0

	if usePerIndexerQuery {
		textReq = &indexer.SearchRequest{
			Cat:                req.Cat,
			Limit:              req.Limit,
			IMDbID:             req.IMDbID,
			TMDBID:             req.TMDBID,
			EffectiveByIndexer: req.EffectiveByIndexer,
			PerIndexerQuery:    req.PerIndexerQuery,
		}
	} else if tmdbClient != nil && cfg != nil {
		var textQuery string
		includeYear := true
		searchTitleLanguage := cfg.GetSearchTitleLanguage()
		if contentType == "movie" {
			if searchTitleLanguage != "" {
				if q, err := tmdbClient.GetMovieTitleForSearch(contentIDs.ImdbID, req.TMDBID, searchTitleLanguage, includeYear, false); err == nil {
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
			// String search for episodes disabled: use ID-only search and rely on FilterResults.
			// Uncomment below to re-enable dual search (ID + "Show Name S00E00" query).
			// if name, err := tmdbClient.GetTVShowName(tmdbForText, imdbForText); err == nil {
			// 	seasonNum, _ := strconv.Atoi(req.Season)
			// 	epNum, _ := strconv.Atoi(req.Episode)
			// 	if seasonNum > 0 || epNum > 0 {
			// 		textQuery = fmt.Sprintf("%s S%02dE%02d", name, seasonNum, epNum)
			// 	} else {
			// 		textQuery = fmt.Sprintf("%s S%sE%s", name, req.Season, req.Episode)
			// 	}
			// }
		}
		if textQuery != "" {
			textReq = &indexer.SearchRequest{Query: textQuery, Cat: req.Cat, Limit: req.Limit, Season: req.Season, Episode: req.Episode}
		}
	}

	// NOTE: Per-indexer DisableIdSearch / DisableStringSearch flags are enforced
	// inside the Aggregator.Search method, which skips individual indexers based
	// on whether the request is ID-based or text-based. No filtering is needed here
	// because RunIndexerSearches dispatches two separate Search calls (one ID, one text)
	// and the aggregator handles the per-indexer opt-out.

	var idResp *indexer.SearchResponse
	var idErr error
	var textReleases []*release.Release
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		idResp, idErr = idx.Search(idReq)
	}()

	if textReq != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if resp, err := idx.Search(*textReq); err == nil {
				indexer.NormalizeSearchResponse(resp)
				textReleases = resp.Releases
			}
		}()
	}

	wg.Wait()

	if idErr != nil {
		return nil, fmt.Errorf("indexer search failed: %w", idErr)
	}
	if idResp != nil {
		indexer.NormalizeSearchResponse(idResp)
	}

	var idReleases []*release.Release
	if idResp != nil {
		idReleases = idResp.Releases
	}

	combined := make([]*release.Release, 0, len(idReleases)+len(textReleases))
	for _, rel := range idReleases {
		if rel != nil {
			rel.QuerySource = "id"
			combined = append(combined, rel)
		}
	}
	for _, rel := range textReleases {
		if rel != nil {
			rel.QuerySource = "text"
			combined = append(combined, rel)
		}
	}

	merged := MergeAndDedupeSearchResults(combined)
	if len(textReleases) > 0 {
		logger.Debug("Indexer dual search", "id", len(idReleases), "text", len(textReleases))
	}

	if filterQuery != "" {
		merged = FilterResults(merged, contentType, filterQuery, req.Season, req.Episode)
	}
	return merged, nil
}
