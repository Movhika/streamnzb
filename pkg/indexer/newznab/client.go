package newznab

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"streamnzb/pkg/core/config"
	"streamnzb/pkg/core/env"
	"streamnzb/pkg/core/logger"
	"streamnzb/pkg/indexer"
	"strings"
	"sync"
	"time"
)

// Client represents a Newznab API client for a single indexer
type Client struct {
	baseURL string
	apiPath string // API path (e.g., "/api" or "/api/v1")
	apiKey  string
	name    string
	client  *http.Client
	cfg     config.IndexerConfig // full config for per-indexer search overrides
	caps    *indexer.Caps        // populated by GetCaps(); nil until fetched

	// Usage tracking
	apiLimit          int
	apiUsed           int
	apiRemaining      int
	downloadLimit     int
	downloadUsed      int
	downloadRemaining int
	usageManager      *indexer.UsageManager
	mu                sync.RWMutex
}

// Ensure Client implements indexer.Indexer and IndexerWithCaps at compile time.
var _ indexer.Indexer = (*Client)(nil)
var _ indexer.IndexerWithCaps = (*Client)(nil)

// APIError represents a Newznab API error response
type APIError struct {
	XMLName     xml.Name `xml:"error"`
	Code        int      `xml:"code,attr"`
	Description string   `xml:"description,attr"`
}

// Name returns the name of this indexer
func (c *Client) Name() string {
	if c.name != "" {
		return c.name
	}
	return "Newznab"
}

// Type returns the config type of this indexer ("newznab", "aggregator", "nzbhydra", "prowlarr", etc.).
func (c *Client) Type() string {
	if c.cfg.Type != "" {
		return c.cfg.Type
	}
	return "newznab"
}

// GetUsage returns the current usage stats
func (c *Client) GetUsage() indexer.Usage {
	c.mu.RLock()
	u := indexer.Usage{
		APIHitsLimit:       c.apiLimit,
		APIHitsUsed:        c.apiUsed,
		APIHitsRemaining:   c.apiRemaining,
		DownloadsLimit:     c.downloadLimit,
		DownloadsUsed:      c.downloadUsed,
		DownloadsRemaining: c.downloadRemaining,
	}
	c.mu.RUnlock()
	if c.usageManager != nil {
		ud := c.usageManager.GetIndexerUsage(c.name)
		u.AllTimeAPIHitsUsed = ud.AllTimeAPIHitsUsed
		u.AllTimeDownloadsUsed = ud.AllTimeDownloadsUsed
	}
	return u
}

// NewClient creates a new Newznab client
func NewClient(cfg config.IndexerConfig, um *indexer.UsageManager) *Client {
	// Create HTTP client with TLS skip verify for self-signed certs (common in local setups)
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		MaxConnsPerHost:     100,
		IdleConnTimeout:     90 * time.Second,
	}

	// Default API path to "/api" if not specified
	apiPath := cfg.APIPath
	if apiPath == "" {
		apiPath = "/api"
	}
	// Ensure it starts with "/"
	if !strings.HasPrefix(apiPath, "/") {
		apiPath = "/" + apiPath
	}

	c := &Client{
		name:    cfg.Name,
		baseURL: strings.TrimRight(cfg.URL, "/"),
		apiPath: apiPath,
		apiKey:  cfg.APIKey,
		cfg:     cfg,
		client: &http.Client{
			Timeout:   10 * time.Second,
			Transport: transport,
		},
		apiLimit:          cfg.APIHitsDay,
		apiUsed:           0,
		apiRemaining:      cfg.APIHitsDay,
		downloadLimit:     cfg.DownloadsDay,
		downloadUsed:      0,
		downloadRemaining: cfg.DownloadsDay,
		usageManager:      um,
	}

	// Load initial usage if manager is provided
	if um != nil {
		usage := um.GetIndexerUsage(cfg.Name)
		c.apiUsed = usage.APIHitsUsed
		c.downloadUsed = usage.DownloadsUsed

		c.apiRemaining = cfg.APIHitsDay - usage.APIHitsUsed
		c.downloadRemaining = cfg.DownloadsDay - usage.DownloadsUsed

		// Ensure remaining isn't negative if limits were lowered
		if c.apiRemaining < 0 && cfg.APIHitsDay > 0 {
			c.apiRemaining = 0
		}
		if c.downloadRemaining < 0 && cfg.DownloadsDay > 0 {
			c.downloadRemaining = 0
		}
	}

	return c
}

// checkAPILimit returns error if API limit is reached
func (c *Client) checkAPILimit() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.apiLimit > 0 && c.apiRemaining <= 0 {
		return fmt.Errorf("API limit reached for %s", c.Name())
	}
	return nil
}

// checkDownloadLimit returns error if download limit is reached
func (c *Client) checkDownloadLimit() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.downloadLimit > 0 && c.downloadRemaining <= 0 {
		return fmt.Errorf("download limit reached for %s", c.Name())
	}
	return nil
}

// updateUsageFromHeaders updates remaining counts from Newznab headers
func (c *Client) updateUsageFromHeaders(h http.Header) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Newznab Standard Headers
	if val := h.Get("X-RateLimit-Daily-Limit"); val != "" {
		if limit, err := strconv.Atoi(val); err == nil {
			c.apiLimit = limit
		}
	}
	if val := h.Get("X-RateLimit-Daily-Remaining"); val != "" {
		if remaining, err := strconv.Atoi(val); err == nil {
			c.apiRemaining = remaining
		}
	}

	// Grab limits (Downloads)
	if val := h.Get("X-DNZBLimit-Daily-Limit"); val != "" {
		if limit, err := strconv.Atoi(val); err == nil {
			c.downloadLimit = limit
		}
	}
	if val := h.Get("X-DNZBLimit-Daily-Remaining"); val != "" {
		if remaining, err := strconv.Atoi(val); err == nil {
			c.downloadRemaining = remaining
		}
	}

	// Some indexers use non-standard headers
	if val := h.Get("x-api-remaining"); val != "" && h.Get("X-RateLimit-Daily-Remaining") == "" {
		if remaining, err := strconv.Atoi(val); err == nil {
			c.apiRemaining = remaining
		}
	}
	if val := h.Get("x-grab-remaining"); val != "" && h.Get("X-DNZBLimit-Daily-Remaining") == "" {
		if remaining, err := strconv.Atoi(val); err == nil {
			c.downloadRemaining = remaining
		}
	}

	// Update persistent storage from header-derived absolute values.
	// Only call UpdateUsage when we have authoritative limits from headers;
	// unlimited accounts track incrementally via IncrementUsed in Search/DownloadNZB.
	if c.usageManager != nil && (c.apiLimit > 0 || c.downloadLimit > 0) {
		if c.apiLimit > 0 {
			c.apiUsed = c.apiLimit - c.apiRemaining
		}
		if c.downloadLimit > 0 {
			c.downloadUsed = c.downloadLimit - c.downloadRemaining
		}
		c.usageManager.UpdateUsage(c.name, c.apiUsed, c.downloadUsed)
	}
}

// Ping checks if the indexer is reachable
func (c *Client) Ping() error {
	apiURL := fmt.Sprintf("%s%s?t=caps&apikey=%s", c.baseURL, c.apiPath, c.apiKey)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", env.IndexerQueryHeader())
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s indexer returned error status: %d", c.Name(), resp.StatusCode)
	}
	return nil
}

// GetCaps fetches the indexer's capabilities (categories, search types, limits).
// Results are cached on the client for use in Search().
func (c *Client) GetCaps() (*indexer.Caps, error) {
	apiURL := fmt.Sprintf("%s%s?t=caps", c.baseURL, c.apiPath)
	if c.apiKey != "" {
		apiURL += "&apikey=" + url.QueryEscape(c.apiKey)
	}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create caps request: %w", err)
	}
	req.Header.Set("User-Agent", env.IndexerQueryHeader())

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch caps from %s: %w", c.Name(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s caps returned status %d", c.Name(), resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read caps from %s: %w", c.Name(), err)
	}

	if err := c.checkNewznabError(body); err != nil {
		return nil, err
	}

	caps, err := indexer.ParseCapsXML(body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse caps from %s: %w", c.Name(), err)
	}

	c.mu.Lock()
	c.caps = caps
	c.mu.Unlock()

	logger.Debug("Fetched capabilities", "indexer", c.Name(),
		"categories", len(caps.Categories),
		"movie_search", caps.Searching.MovieSearch,
		"tv_search", caps.Searching.TVSearch,
		"retention", caps.RetentionDays)

	return caps, nil
}

// checkNewznabError checks for Newznab error responses and returns appropriate errors
func (c *Client) checkNewznabError(bodyBytes []byte) error {
	var apiErr APIError
	if err := xml.Unmarshal(bodyBytes, &apiErr); err == nil && apiErr.Description != "" {
		// Parse error code to determine error type
		switch {
		case apiErr.Code >= 100 && apiErr.Code <= 199:
			return fmt.Errorf("%s authentication error (code %d): %s", c.Name(), apiErr.Code, apiErr.Description)
		case apiErr.Code == 201:
			return fmt.Errorf("%s request limit reached (code %d): %s", c.Name(), apiErr.Code, apiErr.Description)
		case apiErr.Code >= 200 && apiErr.Code <= 299:
			return fmt.Errorf("%s request error (code %d): %s", c.Name(), apiErr.Code, apiErr.Description)
		case apiErr.Code >= 300 && apiErr.Code <= 399:
			return fmt.Errorf("%s server error (code %d): %s", c.Name(), apiErr.Code, apiErr.Description)
		default:
			return fmt.Errorf("%s API error (code %d): %s", c.Name(), apiErr.Code, apiErr.Description)
		}
	}
	return nil
}

// Search queries the Newznab indexer
// The indexer handles pagination internally, we just request what we need
func (c *Client) Search(req indexer.SearchRequest) (*indexer.SearchResponse, error) {
	if err := c.checkAPILimit(); err != nil {
		return nil, err
	}

	limit := req.Limit
	if o := req.OptionalOverrides; o != nil && o.SearchResultLimit > 0 {
		limit = o.SearchResultLimit
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	params := url.Values{}
	params.Set("apikey", c.apiKey)
	params.Set("o", "xml")
	params.Set("limit", fmt.Sprintf("%d", limit))
	params.Set("offset", "0") // Start from beginning

	// Determine search type from caps (with fallback)
	c.mu.RLock()
	caps := c.caps
	c.mu.RUnlock()

	isMovieSearch := strings.HasPrefix(req.Cat, "2")
	isTVSearch := strings.HasPrefix(req.Cat, "5")

	// For TV: use t=search for string queries (per Newznab base search); t=tvsearch for ID/structured (tvdbid, season/ep).
	useTVSearchParams := false
	if isMovieSearch && (caps == nil || caps.Searching.MovieSearch) {
		// ID-only movie search uses t=movie with imdbid only; text search uses t=search
		if req.Query == "" && req.IMDbID != "" {
			params.Set("t", "movie")
		} else {
			params.Set("t", "search")
		}
	} else if isTVSearch && (caps == nil || caps.Searching.TVSearch) {
		if req.Query != "" && req.TVDBID == "" && req.IMDbID == "" {
			params.Set("t", "search")
		} else {
			params.Set("t", "tvsearch")
			useTVSearchParams = true
		}
	} else {
		params.Set("t", "search")
	}

	query := req.Query
	extraTerms := c.cfg.ExtraSearchTerms
	if o := req.OptionalOverrides; o != nil && o.ExtraSearchTerms != nil {
		extraTerms = *o.ExtraSearchTerms
	}
	if extraTerms != "" {
		if query != "" {
			query = query + " " + extraTerms
		} else {
			query = extraTerms
		}
	}
	idOnlyMovie := isMovieSearch && req.Query == "" && req.IMDbID != ""
	if query != "" && !idOnlyMovie {
		params.Set("q", query)
	}

	// Only send IDs that Newznab uses: imdbid for movie; tvdbid+season+ep for tvsearch (no imdbid on tvsearch)
	if isMovieSearch && req.IMDbID != "" {
		imdbID := strings.TrimPrefix(req.IMDbID, "tt")
		params.Set("imdbid", imdbID)
	}
	if req.TVDBID != "" {
		params.Set("tvdbid", req.TVDBID)
	}

	cat := req.Cat
	if isMovieSearch && c.cfg.MovieCategories != "" {
		cat = c.cfg.MovieCategories
	} else if isTVSearch && c.cfg.TVCategories != "" {
		cat = c.cfg.TVCategories
	}
	if o := req.OptionalOverrides; o != nil {
		if isMovieSearch && o.MovieCategories != nil && *o.MovieCategories != "" {
			cat = *o.MovieCategories
		} else if isTVSearch && o.TVCategories != nil && *o.TVCategories != "" {
			cat = *o.TVCategories
		}
	}
	if cat != "" {
		params.Set("cat", cat)
	}

	useSeasonEp := c.cfg.UseSeasonEpisodeParams
	if o := req.OptionalOverrides; o != nil && o.UseSeasonEpisodeParams != nil {
		useSeasonEp = o.UseSeasonEpisodeParams
	}
	// season/ep are tvsearch params; omit for t=search (string query) to avoid strict matching on indexers
	if useTVSearchParams && (useSeasonEp == nil || *useSeasonEp) {
		if req.Season != "" {
			params.Set("season", req.Season)
		}
		if req.Episode != "" {
			params.Set("ep", req.Episode)
		}
	}

	apiURL := fmt.Sprintf("%s%s?%s", c.baseURL, c.apiPath, params.Encode())
	logger.Debug("Newznab search request", "indexer", c.Name(), "url", apiURL, "limit", limit)

	httpReq, err := http.NewRequestWithContext(context.Background(), "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("User-Agent", env.IndexerQueryHeader())
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to query %s: %w", c.Name(), err)
	}
	defer resp.Body.Close()

	c.mu.Lock()
	c.apiUsed++
	if c.apiRemaining > 0 {
		c.apiRemaining--
	}
	c.mu.Unlock()

	c.updateUsageFromHeaders(resp.Header)

	// For unlimited accounts (no header-derived limits), persist incrementally
	if c.usageManager != nil && c.apiLimit == 0 {
		c.usageManager.IncrementUsed(c.name, 1, 0)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s response: %w", c.Name(), err)
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		// Try to parse Newznab error response
		if err := c.checkNewznabError(bodyBytes); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("%s returned status %d: %s", c.Name(), resp.StatusCode, string(bodyBytes))
	}

	// Check for Newznab API errors in successful HTTP responses
	if err := c.checkNewznabError(bodyBytes); err != nil {
		return nil, err
	}

	var result indexer.SearchResponse
	if err := xml.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to parse %s response: %w", c.Name(), err)
	}

	// Populate SourceIndexer and fix metadata for each item
	for i := range result.Channel.Items {
		item := &result.Channel.Items[i]
		item.SourceIndexer = c

		// Fallback size extraction
		if item.Size <= 0 {
			if item.Enclosure.Length > 0 {
				item.Size = item.Enclosure.Length
			} else if sizeAttr := item.GetAttribute("size"); sizeAttr != "" {
				fmt.Sscanf(sizeAttr, "%d", &item.Size)
			}
		}
	}

	// Truncate to requested limit if indexer returned more
	if len(result.Channel.Items) > limit {
		result.Channel.Items = result.Channel.Items[:limit]
	}

	return &result, nil
}

func (c *Client) DownloadNZB(ctx context.Context, nzbURL string) ([]byte, error) {
	if err := c.checkDownloadLimit(); err != nil {
		logger.Warn("Download limit reached for indexer", "indexer", c.Name())
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", nzbURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", env.IndexerGrabHeader())
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download NZB from %s: %w", c.Name(), err)
	}
	defer resp.Body.Close()

	c.mu.Lock()
	c.apiUsed++
	c.downloadUsed++
	if c.apiRemaining > 0 {
		c.apiRemaining--
	}
	if c.downloadRemaining > 0 {
		c.downloadRemaining--
	}
	c.mu.Unlock()

	c.updateUsageFromHeaders(resp.Header)

	// For unlimited accounts (no header-derived limits), persist incrementally
	if c.usageManager != nil && c.apiLimit == 0 && c.downloadLimit == 0 {
		c.usageManager.IncrementUsed(c.name, 1, 1)
	} else if c.usageManager != nil && c.apiLimit == 0 {
		c.usageManager.IncrementUsed(c.name, 1, 0)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s NZB download returned status %d", c.Name(), resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read NZB data from %s: %w", c.Name(), err)
	}

	return data, nil
}
