package tmdb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"streamnzb/pkg/core/logger"
	"streamnzb/pkg/release"
	"strings"
	"time"
)

// Client for TheMovieDB API
type Client struct {
	apiKey string
	client *http.Client
}

// NewClient creates a new TMDB client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// FindResponse represents the response from /find/{id}
type FindResponse struct {
	MovieResults     []Result `json:"movie_results"`
	PersonResults    []Result `json:"person_results"`
	TVResults        []Result `json:"tv_results"`
	TVEpisodeResults []Result `json:"tv_episode_results"`
	TVSeasonResults  []Result `json:"tv_season_results"`
}

// Result represents a search result item
type Result struct {
	ID               int    `json:"id"`
	Name             string `json:"name"`               // TV
	Title            string `json:"title"`              // Movie
	OriginalName     string `json:"original_name"`      // TV
	OriginalTitle    string `json:"original_title"`     // Movie
	OriginalLanguage string `json:"original_language"` // Movie (ISO 639-1)
	MediaType        string `json:"media_type"`
	Overview         string `json:"overview"`
	ReleaseDate      string `json:"release_date"`       // Movie (from Find)
}

// SearchMultiResponse is the response from GET /search/multi
type SearchMultiResponse struct {
	Page         int                 `json:"page"`
	Results      []SearchMultiResult `json:"results"`
	TotalPages   int                 `json:"total_pages"`
	TotalResults int                 `json:"total_results"`
}

// SearchMultiResult is one item from multi search (movie or TV)
type SearchMultiResult struct {
	ID             int    `json:"id"`
	Title          string `json:"title"`           // Movie
	Name           string `json:"name"`            // TV
	MediaType      string `json:"media_type"`     // "movie" or "tv"
	ReleaseDate    string `json:"release_date"`   // Movie
	FirstAirDate   string `json:"first_air_date"` // TV
	OriginalTitle  string `json:"original_title"`
	OriginalName   string `json:"original_name"`
	PosterPath     string `json:"poster_path"`     // e.g. "/abc.jpg" for image.tmdb.org/t/p/w92/abc.jpg"
}

// ExternalIDsResponse represents the response from /{type}/{id}/external_ids
type ExternalIDsResponse struct {
	ID          int    `json:"id"`
	IMDbID      string `json:"imdb_id"`
	TVDBID      int    `json:"tvdb_id"`
	FreebaseID  string `json:"freebase_id"`
	WikidataID  string `json:"wikidata_id"`
	FacebookID  string `json:"facebook_id"`
	InstagramID string `json:"instagram_id"`
	TwitterID   string `json:"twitter_id"`
}

// MovieTranslationsResponse is the response from GET /movie/{id}/translations
// https://developer.themoviedb.org/reference/movie-translations
type MovieTranslationsResponse struct {
	ID           int                    `json:"id"`
	Translations []MovieTranslationEntry `json:"translations"`
}

// MovieTranslationEntry is one translation (language) for a movie
type MovieTranslationEntry struct {
	ISO639_1    string               `json:"iso_639_1"`
	ISO3166_1   string               `json:"iso_3166_1"`
	Name        string               `json:"name"`
	EnglishName string               `json:"english_name"`
	Data        MovieTranslationData `json:"data"`
}

// MovieTranslationData holds the translated title and overview
type MovieTranslationData struct {
	Title    string `json:"title"`
	Overview string `json:"overview"`
}

func (c *Client) doRequest(endpoint string, params url.Values) (*http.Response, error) {
	reqURL := fmt.Sprintf("%s?%s", endpoint, params.Encode())

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("accept", "application/json")

	return c.client.Do(req)
}

// Find searches for objects by external ID (IMDb ID)
// source: 'imdb_id', 'tvdb_id', etc.
func (c *Client) Find(externalID, source string) (*FindResponse, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("TMDB API key not configured")
	}

	endpoint := fmt.Sprintf("https://api.themoviedb.org/3/find/%s", externalID)
	params := url.Values{}
	params.Set("external_source", source)

	resp, err := c.doRequest(endpoint, params)
	if err != nil {
		return nil, fmt.Errorf("TMDB find request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB returned status: %d", resp.StatusCode)
	}

	var result FindResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode TMDB response: %w", err)
	}

	return &result, nil
}

// SearchMulti searches for movies and TV shows by query string.
// Uses GET /search/multi. Limit results to the first page with page size up to 20.
func (c *Client) SearchMulti(query string) (*SearchMultiResponse, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("TMDB API key not configured")
	}
	if query == "" {
		return &SearchMultiResponse{Results: []SearchMultiResult{}}, nil
	}
	endpoint := "https://api.themoviedb.org/3/search/multi"
	params := url.Values{}
	params.Set("query", query)
	params.Set("page", "1")
	resp, err := c.doRequest(endpoint, params)
	if err != nil {
		return nil, fmt.Errorf("TMDB search request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB returned status: %d", resp.StatusCode)
	}
	var result SearchMultiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode TMDB search response: %w", err)
	}
	// Limit to 20 results
	if len(result.Results) > 20 {
		result.Results = result.Results[:20]
	}
	// Filter to only movie and tv
	filtered := result.Results[:0]
	for _, r := range result.Results {
		if r.MediaType == "movie" || r.MediaType == "tv" {
			filtered = append(filtered, r)
		}
	}
	result.Results = filtered
	return &result, nil
}

// GetExternalIDs retrieves external IDs for a specific TMDB object
// mediaType: 'movie' or 'tv'
func (c *Client) GetExternalIDs(tmdbID int, mediaType string) (*ExternalIDsResponse, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("TMDB API key not configured")
	}

	endpoint := fmt.Sprintf("https://api.themoviedb.org/3/%s/%d/external_ids", mediaType, tmdbID)
	params := url.Values{}

	resp, err := c.doRequest(endpoint, params)
	if err != nil {
		return nil, fmt.Errorf("TMDB external_ids request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB returned status: %d", resp.StatusCode)
	}

	var result ExternalIDsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode TMDB response: %w", err)
	}

	return &result, nil
}

// MovieDetails is the response from GET /movie/{id}
type MovieDetails struct {
	ID               int    `json:"id"`
	Title            string `json:"title"`
	ReleaseDate      string `json:"release_date"`
	OriginalTitle    string `json:"original_title"`
	OriginalLanguage string `json:"original_language"` // ISO 639-1, e.g. "en", "ja"
}

// TVDetails is the response from GET /tv/{id}
type TVDetails struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// GetMovieTitle returns the movie title for text-based search.
// Supports IMDb ID (tt123) or TMDB ID.
func (c *Client) GetMovieTitle(imdbID string, tmdbID string) (string, error) {
	title, _, err := c.GetMovieTitleAndYear(imdbID, tmdbID)
	return title, err
}

// GetMovieTitleAndYear returns the movie title and release year (e.g. "2026" from ReleaseDate).
// Year is empty when not available (e.g. when resolving by IMDb ID only via Find).
func (c *Client) GetMovieTitleAndYear(imdbID string, tmdbID string) (title string, year string, err error) {
	if tmdbID != "" {
		if id, parseErr := strconv.Atoi(tmdbID); parseErr == nil {
			d, getErr := c.GetMovieDetails(id)
			if getErr != nil {
				return "", "", getErr
			}
			year = ""
			if len(d.ReleaseDate) >= 4 {
				year = d.ReleaseDate[:4]
			}
			return d.Title, year, nil
		}
	}
	if imdbID != "" {
		find, findErr := c.Find(imdbID, "imdb_id")
		if findErr != nil {
			return "", "", findErr
		}
		if len(find.MovieResults) > 0 {
			return find.MovieResults[0].Title, "", nil
		}
	}
	return "", "", fmt.Errorf("could not resolve movie title")
}

// GetTVShowName returns the TV show name for text-based search.
// Supports TMDB ID or IMDb ID (tt123).
func (c *Client) GetTVShowName(tmdbID string, imdbID string) (string, error) {
	if tmdbID != "" {
		if id, err := strconv.Atoi(tmdbID); err == nil {
			d, err := c.GetTVDetails(id)
			if err != nil {
				return "", err
			}
			return d.Name, nil
		}
	}
	if imdbID != "" {
		find, err := c.Find(imdbID, "imdb_id")
		if err != nil {
			return "", err
		}
		if len(find.TVResults) > 0 {
			return find.TVResults[0].Name, nil
		}
	}
	return "", fmt.Errorf("could not resolve TV show name")
}

// GetMovieDetails fetches movie title for text-based search.
func (c *Client) GetMovieDetails(tmdbID int) (*MovieDetails, error) {
	return c.GetMovieDetailsWithLanguage(tmdbID, "")
}

// GetMovieDetailsWithLanguage fetches movie details in the given language (e.g. "de-DE" for German).
// Pass "" for default language.
func (c *Client) GetMovieDetailsWithLanguage(tmdbID int, language string) (*MovieDetails, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("TMDB API key not configured")
	}
	endpoint := fmt.Sprintf("https://api.themoviedb.org/3/movie/%d", tmdbID)
	params := url.Values{}
	if language != "" {
		params.Set("language", language)
	}
	resp, err := c.doRequest(endpoint, params)
	if err != nil {
		return nil, fmt.Errorf("TMDB movie details: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB returned status: %d", resp.StatusCode)
	}
	var d MovieDetails
	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return nil, fmt.Errorf("TMDB movie decode: %w", err)
	}
	return &d, nil
}

// GetMovieTranslations fetches all translations for a movie (GET /movie/{id}/translations).
// https://developer.themoviedb.org/reference/movie-translations
func (c *Client) GetMovieTranslations(movieID int) (*MovieTranslationsResponse, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("TMDB API key not configured")
	}
	endpoint := fmt.Sprintf("https://api.themoviedb.org/3/movie/%d/translations", movieID)
	resp, err := c.doRequest(endpoint, url.Values{})
	if err != nil {
		return nil, fmt.Errorf("TMDB movie translations: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB returned status: %d", resp.StatusCode)
	}
	var out MovieTranslationsResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("TMDB translations decode: %w", err)
	}
	logger.Debug("TMDB movie translations fetched", "movie_id", movieID, "count", len(out.Translations))
	return &out, nil
}

// movieTitleFromTranslations returns the translated title for the given language.
// language is e.g. "de-DE" (iso_639_1 + iso_3166_1); we match exact or language-only (e.g. "de").
func movieTitleFromTranslations(translations *MovieTranslationsResponse, language string) string {
	if translations == nil || language == "" {
		return ""
	}
	langCode, countryCode := splitLanguageTag(language)
	for i := range translations.Translations {
		t := &translations.Translations[i]
		if t.Data.Title == "" {
			continue
		}
		// Prefer exact match (e.g. de-DE)
		if countryCode != "" {
			if strings.EqualFold(t.ISO639_1, langCode) && strings.EqualFold(t.ISO3166_1, countryCode) {
				logger.Debug("TMDB translation match", "requested", language, "iso_639_1", t.ISO639_1, "iso_3166_1", t.ISO3166_1, "title", t.Data.Title)
				return t.Data.Title
			}
		} else {
			if strings.EqualFold(t.ISO639_1, langCode) {
				logger.Debug("TMDB translation match (language only)", "requested", language, "iso_639_1", t.ISO639_1, "title", t.Data.Title)
				return t.Data.Title
			}
		}
	}
	// Fallback: match language only
	for i := range translations.Translations {
		t := &translations.Translations[i]
		if t.Data.Title != "" && strings.EqualFold(t.ISO639_1, langCode) {
			logger.Debug("TMDB translation match (fallback)", "requested", language, "iso_639_1", t.ISO639_1, "iso_3166_1", t.ISO3166_1, "title", t.Data.Title)
			return t.Data.Title
		}
	}
	logger.Debug("TMDB no translation for language", "requested", language, "lang_code", langCode, "country_code", countryCode, "available", len(translations.Translations))
	return ""
}

func splitLanguageTag(tag string) (lang, country string) {
	tag = strings.TrimSpace(tag)
	if i := strings.Index(tag, "-"); i >= 0 {
		return tag[:i], tag[i+1:]
	}
	return tag, ""
}

// GetMovieTitleForSearch returns the movie title (and optional year) for indexer search,
// optionally in the given language and normalized for filename matching (e.g. ü→ue).
// language: e.g. "de-DE" for German; empty = default. When set, we use the Translations API for the title.
// includeYear: append release year. normalize: apply umlaut→ascii.
func (c *Client) GetMovieTitleForSearch(imdbID, tmdbID, language string, includeYear, normalize bool) (string, error) {
	var movieID int
	var title, year string

	// Resolve movie ID and default title/year
	if tmdbID != "" {
		if id, err := strconv.Atoi(tmdbID); err == nil {
			movieID = id
			d, err := c.GetMovieDetails(id)
			if err == nil {
				title = d.Title
				if includeYear && len(d.ReleaseDate) >= 4 {
					year = d.ReleaseDate[:4]
				}
				logger.Debug("TMDB movie title from details", "tmdb_id", movieID, "title", title, "language", language)
			}
		}
	}
	if movieID == 0 && imdbID != "" {
		find, err := c.Find(imdbID, "imdb_id")
		if err != nil {
			return "", err
		}
		if len(find.MovieResults) > 0 {
			movieID = find.MovieResults[0].ID
			title = find.MovieResults[0].Title
			if includeYear && find.MovieResults[0].ReleaseDate != "" && len(find.MovieResults[0].ReleaseDate) >= 4 {
				year = find.MovieResults[0].ReleaseDate[:4]
			}
			logger.Debug("TMDB movie resolved from IMDb Find", "imdb_id", imdbID, "tmdb_id", movieID, "default_title", title)
		}
	}

	// When a language is requested, use the Translations API for the title
	if language != "" && movieID != 0 {
		tr, err := c.GetMovieTranslations(movieID)
		if err != nil {
			logger.Debug("TMDB translations not used, falling back to default title", "movie_id", movieID, "language", language, "err", err)
		} else if t := movieTitleFromTranslations(tr, language); t != "" {
			title = t
			logger.Debug("TMDB movie title for search (translated)", "language", language, "title", title)
		}
	}

	if title == "" {
		return "", fmt.Errorf("could not resolve movie title")
	}
	out := strings.TrimSpace(title)
	if year != "" {
		out = out + " " + year
	}
	if normalize {
		out = release.NormalizeTitleForFilename(out)
	}
	return out, nil
}

// GetMovieTitlesForSearch returns primary and optional original-language title for indexer search.
// When the movie's original_language is not "en", original is the formatted original title (with year/normalize)
// so callers can run a second text query and merge results. Uses one fetch (details or find).
func (c *Client) GetMovieTitlesForSearch(imdbID, tmdbID, language string, includeYear, normalize bool) (primary, original string, err error) {
	var movieID int
	var title, year string
	var origTitle, origYear string
	var origLang string

	if tmdbID != "" {
		if id, parseErr := strconv.Atoi(tmdbID); parseErr == nil {
			d, getErr := c.GetMovieDetails(id)
			if getErr != nil {
				return "", "", getErr
			}
			movieID = d.ID
			title = d.Title
			if includeYear && len(d.ReleaseDate) >= 4 {
				year = d.ReleaseDate[:4]
			}
			origTitle = strings.TrimSpace(d.OriginalTitle)
			origLang = d.OriginalLanguage
			if includeYear && len(d.ReleaseDate) >= 4 {
				origYear = d.ReleaseDate[:4]
			}
			logger.Debug("TMDB movie title from details", "tmdb_id", movieID, "title", title, "language", language)
		}
	}
	if movieID == 0 && imdbID != "" {
		find, findErr := c.Find(imdbID, "imdb_id")
		if findErr != nil {
			return "", "", findErr
		}
		if len(find.MovieResults) == 0 {
			return "", "", fmt.Errorf("could not resolve movie title")
		}
		r := find.MovieResults[0]
		movieID = r.ID
		title = r.Title
		if includeYear && r.ReleaseDate != "" && len(r.ReleaseDate) >= 4 {
			year = r.ReleaseDate[:4]
		}
		origTitle = strings.TrimSpace(r.OriginalTitle)
		origLang = r.OriginalLanguage
		if includeYear && r.ReleaseDate != "" && len(r.ReleaseDate) >= 4 {
			origYear = r.ReleaseDate[:4]
		}
		logger.Debug("TMDB movie resolved from IMDb Find", "imdb_id", imdbID, "tmdb_id", movieID, "default_title", title)
	}

	if language != "" && movieID != 0 {
		tr, trErr := c.GetMovieTranslations(movieID)
		if trErr != nil {
			logger.Debug("TMDB translations not used, falling back to default title", "movie_id", movieID, "language", language, "err", trErr)
		} else if t := movieTitleFromTranslations(tr, language); t != "" {
			title = t
			logger.Debug("TMDB movie title for search (translated)", "language", language, "title", title)
		}
	}

	if title == "" {
		return "", "", fmt.Errorf("could not resolve movie title")
	}
	primary = strings.TrimSpace(title)
	if year != "" {
		primary = primary + " " + year
	}
	if normalize {
		primary = release.NormalizeTitleForFilename(primary)
	}
	primary = strings.TrimSpace(primary)

	if origTitle == "" || origLang == "" || strings.ToLower(origLang) == "en" {
		return primary, "", nil
	}
	if release.NormalizeTitle(origTitle) == release.NormalizeTitle(title) {
		return primary, "", nil
	}
	out := origTitle
	if origYear != "" {
		out = out + " " + origYear
	}
	if normalize {
		out = release.NormalizeTitleForFilename(out)
	}
	out = strings.TrimSpace(out)
	if out == primary {
		return primary, "", nil
	}
	logger.Debug("TMDB movie original title for search", "original_language", origLang, "query", out)
	return primary, out, nil
}

// GetTVDetails fetches TV show name for text-based search.
func (c *Client) GetTVDetails(tmdbID int) (*TVDetails, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("TMDB API key not configured")
	}
	endpoint := fmt.Sprintf("https://api.themoviedb.org/3/tv/%d", tmdbID)
	resp, err := c.doRequest(endpoint, url.Values{})
	if err != nil {
		return nil, fmt.Errorf("TMDB TV details: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB returned status: %d", resp.StatusCode)
	}
	var d TVDetails
	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return nil, fmt.Errorf("TMDB TV decode: %w", err)
	}
	return &d, nil
}

// ResolveTVDBID tries to find the TVDB ID for a given IMDb string (e.g. tt123456)
func (c *Client) ResolveTVDBID(imdbID string) (string, error) {
	// 1. Find the TMDB ID from IMDb ID
	findResp, err := c.Find(imdbID, "imdb_id")
	if err != nil {
		return "", err
	}

	// Check if we found a TV show
	if len(findResp.TVResults) == 0 {
		return "", fmt.Errorf("no TV show found for IMDb ID: %s", imdbID)
	}

	tmdbID := findResp.TVResults[0].ID
	logger.Debug("Resolved TMDB ID from IMDb", "imdb", imdbID, "tmdb", tmdbID)

	// 2. Get External IDs using TMDB ID
	extIDs, err := c.GetExternalIDs(tmdbID, "tv")
	if err != nil {
		return "", err
	}

	if extIDs.TVDBID == 0 {
		return "", fmt.Errorf("no TVDB ID found for TMDB ID: %d", tmdbID)
	}

	logger.Debug("Resolved TVDB ID", "tvdb", extIDs.TVDBID)
	return strconv.Itoa(extIDs.TVDBID), nil
}
