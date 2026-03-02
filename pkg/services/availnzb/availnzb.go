package availnzb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"streamnzb/pkg/core/logger"
	"streamnzb/pkg/release"
)

const apiPath = "/api/v1"

type Client struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client

	backbonesMu sync.RWMutex
	backbones   map[string]string
}

type ReportRequest struct {
	URL             string `json:"url"`
	ReleaseName     string `json:"release_name"`
	Size            int64  `json:"size"`
	CompressionType string `json:"compression_type,omitempty"`
	ProviderURL     string `json:"provider_url"`
	Status          bool   `json:"status"`
	ImdbID          string `json:"imdb_id,omitempty"`
	TvdbID          string `json:"tvdb_id,omitempty"`
	Season          int    `json:"season,omitempty"`
	Episode         int    `json:"episode,omitempty"`
}

type BackboneStatus struct {
	Text        string    `json:"text"`
	LastUpdated time.Time `json:"last_updated"`
	Healthy     bool      `json:"healthy"`
}

type ProviderStatus = BackboneStatus

type StatusResponse struct {
	URL          string                    `json:"url"`
	Available    bool                      `json:"available"`
	ReleaseName  string                    `json:"release_name,omitempty"`
	DownloadLink string                    `json:"download_link,omitempty"`
	Size         int64                     `json:"size,omitempty"`
	Summary      map[string]BackboneStatus `json:"summary"`
}

type releaseItemJSON struct {
	URL             string                    `json:"url"`
	ReleaseName     string                    `json:"release_name,omitempty"`
	DownloadLink    string                    `json:"download_link,omitempty"`
	Size            int64                     `json:"size,omitempty"`
	CompressionType string                    `json:"compression_type,omitempty"`
	Indexer         string                    `json:"indexer"`
	Available       bool                      `json:"available"`
	Summary         map[string]BackboneStatus `json:"summary"`
}

type ReleaseWithStatus struct {
	*release.Release
	Available       bool
	CompressionType string
	Summary         map[string]BackboneStatus
}

type ReleasesResult struct {
	ImdbID   string
	Count    int
	Releases []*ReleaseWithStatus
}

type ReportMeta struct {
	ReleaseName     string
	Size            int64
	CompressionType string
	ImdbID          string
	TvdbID          string
	Season          int
	Episode         int
}

func NewClient(baseURL, apiKey string) *Client {
	baseURL = strings.TrimSuffix(baseURL, "/")
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTP: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) ReportAvailability(releaseURL string, providerURL string, status bool, meta ReportMeta) error {
	if c.BaseURL == "" {
		logger.Debug("AvailNZB report skipped", "reason", "no base URL configured")
		return nil
	}
	if c.APIKey == "" {
		logger.Debug("AvailNZB report skipped", "reason", "no API key configured")
		return nil
	}
	if meta.ReleaseName == "" {
		logger.Debug("AvailNZB report skipped", "reason", "no release_name in meta", "url", releaseURL)
		return nil
	}

	body := ReportRequest{
		URL:             releaseURL,
		ReleaseName:     meta.ReleaseName,
		Size:            meta.Size,
		CompressionType: meta.CompressionType,
		ProviderURL:     providerURL,
		Status:          status,
	}

	if meta.TvdbID != "" && (meta.Season > 0 || meta.Episode > 0) {
		body.TvdbID = meta.TvdbID
		body.Season = meta.Season
		body.Episode = meta.Episode
	} else if meta.ImdbID != "" {
		body.ImdbID = meta.ImdbID
	}
	if body.ImdbID == "" && body.TvdbID == "" {
		logger.Debug("AvailNZB report skipped", "reason", "no imdb_id or tvdb_id in meta", "url", releaseURL)
		return nil
	}

	logger.Debug("AvailNZB report", "url", releaseURL, "release_name", body.ReleaseName, "provider", providerURL, "status", status, "imdb_id", body.ImdbID, "tvdb_id", body.TvdbID, "season", body.Season, "episode", body.Episode)

	reqBody, err := json.Marshal(body)
	if err != nil {
		logger.Error("AvailNZB report marshal failed", "err", err)
		return err
	}

	req, err := http.NewRequest("POST", c.BaseURL+apiPath+"/report", bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}

	req.Header.Set("X-API-Key", c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		logger.Error("AvailNZB report unexpected status", "status", resp.StatusCode, "url", releaseURL)
		return fmt.Errorf("availnzb report: unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

type backbonesResponse struct {
	Backbones         []string            `json:"backbones"`
	ProviderHostnames map[string][]string `json:"provider_hostnames"`
}

func (c *Client) RefreshBackbones() error {
	if c.BaseURL == "" {
		return nil
	}
	reqURL := c.BaseURL + apiPath + "/backbones"
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return err
	}
	if c.APIKey != "" {
		req.Header.Set("X-API-Key", c.APIKey)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		logger.Error("AvailNZB RefreshBackbones request failed", "err", err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		logger.Error("AvailNZB RefreshBackbones unexpected status", "status", resp.StatusCode)
		return fmt.Errorf("availnzb backbones: status %d", resp.StatusCode)
	}
	var wrapped backbonesResponse
	if err := json.NewDecoder(resp.Body).Decode(&wrapped); err != nil {
		logger.Error("AvailNZB RefreshBackbones decode failed", "err", err)
		return err
	}
	m := make(map[string]string)
	for backbone, hostnames := range wrapped.ProviderHostnames {
		backbone = strings.TrimSpace(backbone)
		if backbone == "" {
			continue
		}
		for _, h := range hostnames {
			h = strings.ToLower(strings.TrimSpace(h))
			if h != "" {
				m[h] = backbone
			}
		}
	}
	c.backbonesMu.Lock()
	c.backbones = m
	c.backbonesMu.Unlock()
	logger.Debug("AvailNZB RefreshBackbones", "entries", len(m))
	return nil
}

func (c *Client) GetBackbones() (map[string]string, error) {
	c.backbonesMu.RLock()
	defer c.backbonesMu.RUnlock()
	if c.backbones == nil {
		return nil, nil
	}
	out := make(map[string]string, len(c.backbones))
	for k, v := range c.backbones {
		out[k] = v
	}
	return out, nil
}

func (c *Client) GetStatus(releaseURL string) (*StatusResponse, error) {
	if c.BaseURL == "" {
		logger.Trace("AvailNZB GetStatus skipped", "reason", "no base URL")
		return nil, nil
	}

	params := url.Values{}
	params.Set("url", releaseURL)
	reqURL := c.BaseURL + apiPath + "/status/url?" + params.Encode()

	logger.Debug("AvailNZB GetStatus", "url", releaseURL)

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	if c.APIKey != "" {
		req.Header.Set("X-API-Key", c.APIKey)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		logger.Error("AvailNZB GetStatus request failed", "err", err, "url", releaseURL)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		logger.Debug("AvailNZB GetStatus", "result", "not_found", "url", releaseURL)
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		logger.Error("AvailNZB GetStatus unexpected status", "status", resp.StatusCode, "url", releaseURL)
		return nil, fmt.Errorf("availnzb status: unexpected status code: %d", resp.StatusCode)
	}

	var status StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		logger.Error("AvailNZB GetStatus decode failed", "err", err)
		return nil, err
	}

	logger.Debug("AvailNZB GetStatus", "url", releaseURL, "available", status.Available, "backbones", len(status.Summary))
	return &status, nil
}

type releasesResponseJSON struct {
	ImdbID   string            `json:"imdb_id,omitempty"`
	Count    int               `json:"count"`
	Releases []releaseItemJSON `json:"releases"`
}

func (c *Client) GetReleases(imdbID string, tvdbID string, season, episode int, indexers []string, providers []string) (*ReleasesResult, error) {
	if c.BaseURL == "" {
		logger.Trace("AvailNZB GetReleases skipped", "reason", "no base URL")
		return nil, nil
	}

	var path string
	if tvdbID != "" && (season > 0 || episode > 0) {
		path = fmt.Sprintf("%s/status/tvdb/%s/%d/%d", apiPath, url.PathEscape(tvdbID), season, episode)
	} else if imdbID != "" {
		path = apiPath + "/status/imdb/" + url.PathEscape(imdbID)
	} else {
		return nil, fmt.Errorf("availnzb releases: need imdb_id or tvdb_id+season+episode")
	}
	params := url.Values{}
	if len(indexers) > 0 {
		params.Set("indexers", strings.Join(indexers, ","))
	}
	if len(providers) > 0 {
		params.Set("providers", strings.Join(providers, ","))
	}
	reqURL := c.BaseURL + path
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}

	logger.Debug("AvailNZB GetReleases", "imdb_id", imdbID, "tvdb_id", tvdbID, "season", season, "episode", episode, "indexers", len(indexers), "providers", len(providers))

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	if c.APIKey != "" {
		req.Header.Set("X-API-Key", c.APIKey)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		logger.Error("AvailNZB GetReleases request failed", "err", err, "imdb_id", imdbID, "tvdb_id", tvdbID)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("AvailNZB GetReleases unexpected status", "status", resp.StatusCode, "imdb_id", imdbID, "tvdb_id", tvdbID)
		return nil, fmt.Errorf("availnzb releases: unexpected status code: %d", resp.StatusCode)
	}

	var raw releasesResponseJSON
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		logger.Error("AvailNZB GetReleases decode failed", "err", err)
		return nil, err
	}

	releases := make([]*ReleaseWithStatus, 0, len(raw.Releases))
	availableCount := 0
	for i := range raw.Releases {
		r := &raw.Releases[i]
		idx := r.Indexer
		if idx == "" {
			idx = "AvailNZB"
		}
		rel := &release.Release{
			Title:      r.ReleaseName,
			Link:       r.DownloadLink,
			DetailsURL: r.URL,
			Size:       r.Size,
			Indexer:    idx,
		}
		releases = append(releases, &ReleaseWithStatus{
			Release:         rel,
			Available:       r.Available,
			CompressionType: r.CompressionType,
			Summary:         r.Summary,
		})
		if r.Available {
			availableCount++
		}
	}
	logger.Debug("AvailNZB GetReleases", "count", raw.Count, "available", availableCount, "imdb_id", imdbID, "tvdb_id", tvdbID)
	return &ReleasesResult{ImdbID: raw.ImdbID, Count: raw.Count, Releases: releases}, nil
}

func (c *Client) OurBackbones(providerHosts []string) (map[string]bool, error) {
	m, err := c.GetBackbones()
	if err != nil || m == nil {
		return nil, err
	}
	out := make(map[string]bool)
	for _, h := range providerHosts {
		if b := m[strings.ToLower(strings.TrimSpace(h))]; b != "" {
			out[b] = true
		}
	}
	return out, nil
}

func (c *Client) CheckPreDownload(releaseURL string, validProviderHosts []string) (available bool, lastUpdated time.Time, capableProvider string, err error) {
	logger.Debug("AvailNZB CheckPreDownload", "url", releaseURL, "our_providers", len(validProviderHosts))
	if c.BaseURL == "" || releaseURL == "" {
		logger.Trace("AvailNZB CheckPreDownload skipped", "reason", "no base URL or empty release URL")
		return false, time.Time{}, "", nil
	}

	status, err := c.GetStatus(releaseURL)
	if err != nil {
		logger.Debug("AvailNZB CheckPreDownload GetStatus failed", "url", releaseURL, "err", err)
		return false, time.Time{}, "", err
	}
	if status == nil {
		logger.Debug("AvailNZB CheckPreDownload", "result", "not_found", "url", releaseURL)
		return false, time.Time{}, "", nil
	}

	hostToBackbone, err := c.GetBackbones()
	if err != nil || len(hostToBackbone) == 0 {
		logger.Trace("AvailNZB CheckPreDownload", "result", "no_backbone_mapping", "err", err)
		if status.Available && len(status.Summary) > 0 {
			for _, report := range status.Summary {
				if report.LastUpdated.After(lastUpdated) {
					lastUpdated = report.LastUpdated
				}
			}
			return true, lastUpdated, "", nil
		}
		return false, time.Time{}, "", nil
	}
	ourBackbones := make(map[string]bool)
	for _, h := range validProviderHosts {
		if b := hostToBackbone[strings.ToLower(strings.TrimSpace(h))]; b != "" {
			ourBackbones[b] = true
		}
	}
	if len(ourBackbones) == 0 {
		if status.Available && len(status.Summary) > 0 {
			for _, report := range status.Summary {
				if report.LastUpdated.After(lastUpdated) {
					lastUpdated = report.LastUpdated
				}
			}
			return true, lastUpdated, "", nil
		}
		return false, time.Time{}, "", nil
	}

	for backbone, report := range status.Summary {
		if ourBackbones[backbone] && report.Healthy {
			if report.LastUpdated.After(lastUpdated) {
				lastUpdated = report.LastUpdated
			}
			available = true
			for _, h := range validProviderHosts {
				if hostToBackbone[strings.ToLower(strings.TrimSpace(h))] == backbone {
					capableProvider = h
					break
				}
			}
			if capableProvider == "" {
				capableProvider = backbone
			}
			break
		}
	}
	if status.Available && !available && len(status.Summary) > 0 {
		for _, report := range status.Summary {
			if report.LastUpdated.After(lastUpdated) {
				lastUpdated = report.LastUpdated
			}
		}
		available = status.Available
	}

	logger.Debug("AvailNZB CheckPreDownload", "result", "found", "available", available, "capable_provider", capableProvider, "url", releaseURL)
	return available, lastUpdated, capableProvider, nil
}
