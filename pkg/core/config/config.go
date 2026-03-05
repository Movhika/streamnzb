package config

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"

	"streamnzb/pkg/core/env"
	"streamnzb/pkg/core/logger"
	"streamnzb/pkg/core/paths"
)

const defaultAdminPasswordHash = "8c6976e5b5410415bde908bd4dee15dfb167a9c873fc4bb8a81f6f2ab448a918"

type Provider struct {
	Name        string `json:"name"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	Connections int    `json:"connections"`
	UseSSL      bool   `json:"use_ssl"`
	Priority    *int   `json:"priority,omitempty"`
	Enabled     *bool  `json:"enabled,omitempty"`
}

func DefaultFilterConfig() FilterConfig {
	return FilterConfig{
		QualityExcluded: []string{"CAM", "TeleSync", "TeleCine", "SCR"},
		DubbedExcluded:  ptrBool(true),
	}
}

func DefaultSortConfig() SortConfig {
	return SortConfig{
		PreferredResolution: []string{"4k", "1080p", "720p", "sd"},
		PreferredQuality: []string{
			"BluRay REMUX", "REMUX", "BluRay", "BRRip", "BDRip", "UHDRip", "HDRip",
			"WEB-DL", "WEBRip", "WEB-DLRip", "WEB",
			"HDTV", "HDTVRip", "PDTV", "TVRip", "SATRip",
			"DVD", "DVDRip", "PPVRip", "R5", "XviD", "DivX",
		},
		PreferredAvailNZB: []string{"available"},
		SortCriteriaOrder: []string{
			"availnzb", "resolution", "quality", "codec", "visual_tag", "audio", "channels",
			"bit_depth", "container", "languages", "group", "edition", "network", "region",
			"three_d", "size", "keywords", "regex",
		},
		GrabWeight: 0.5,
		AgeWeight:  1.0,
	}
}

func ptrBool(b bool) *bool { return &b }

type IndexerSearchConfig struct {
	SearchResultLimit      int     `json:"search_result_limit,omitempty"`
	IncludeYearInSearch    *bool   `json:"include_year_in_search,omitempty"`
	SearchTitleLanguage    *string `json:"search_title_language,omitempty"`
	SearchTitleNormalize   *bool   `json:"search_title_normalize,omitempty"`
	MovieCategories        *string `json:"movie_categories,omitempty"`
	TVCategories           *string `json:"tv_categories,omitempty"`
	ExtraSearchTerms       *string `json:"extra_search_terms,omitempty"`
	UseSeasonEpisodeParams *bool   `json:"use_season_episode_params,omitempty"`
}

type IndexerConfig struct {
	Name         string `json:"name"`
	URL          string `json:"url"`
	APIKey       string `json:"api_key"`
	APIPath      string `json:"api_path"`
	Type         string `json:"type"`
	APIHitsDay   int    `json:"api_hits_day"`
	DownloadsDay int    `json:"downloads_day"`
	Enabled      *bool  `json:"enabled,omitempty"`

	Username string `json:"username"`
	Password string `json:"password"`

	MovieCategories        string `json:"movie_categories,omitempty"`
	TVCategories           string `json:"tv_categories,omitempty"`
	ExtraSearchTerms       string `json:"extra_search_terms,omitempty"`
	UseSeasonEpisodeParams *bool  `json:"use_season_episode_params,omitempty"`
	SearchResultLimit      int    `json:"search_result_limit,omitempty"`
	IncludeYearInSearch    *bool  `json:"include_year_in_search,omitempty"`
	SearchTitleLanguage    string `json:"search_title_language,omitempty"`
	SearchTitleNormalize   *bool  `json:"search_title_normalize,omitempty"`
}

type Config struct {
	Indexers []IndexerConfig `json:"indexers"`

	AddonPort    int    `json:"addon_port"`
	AddonBaseURL string `json:"addon_base_url"`
	LogLevel     string `json:"log_level"`

	AdminUsername           string `json:"admin_username"`
	AdminPasswordHash       string `json:"admin_password_hash"`
	AdminMustChangePassword bool   `json:"admin_must_change_password"`
	AdminToken              string `json:"admin_token"`

	Providers []Provider `json:"providers"`

	ProxyPort     int    `json:"proxy_port"`
	ProxyHost     string `json:"proxy_host"`
	ProxyAuthUser string `json:"proxy_auth_user"`
	ProxyAuthPass string `json:"proxy_auth_pass"`

	AvailNZBURL    string `json:"-"`
	AvailNZBAPIKey string `json:"-"`

	TMDBAPIKey string `json:"-"`

	TVDBAPIKey string `json:"-"`

	Devices map[string]*DeviceEntry `json:"devices,omitempty"`

	Streams []*StreamEntry `json:"streams,omitempty"`

	// MemoryLimitMB sets a soft limit on total Go heap (runtime/debug.SetMemoryLimit). 0 = no limit.
	// When set, segment cache is automatically 80% of this limit.
	// Use this to stop memory climbing; the runtime will GC more aggressively to stay under the limit.
	MemoryLimitMB int `json:"memory_limit_mb,omitempty"`

	// KeepLogFiles is how many log files to keep (current streamnzb.log + rotated streamnzb-*.log). Default 9.
	KeepLogFiles int `json:"keep_log_files,omitempty"`

	// AvailNZBMode controls how the AvailNZB integration behaves.
	// "" or "full"        – fetch availability status AND report playback results (default).
	// "status_only"       – fetch availability status but never report back (leeching).
	// "disabled"          – disable AvailNZB entirely (no GET, no POST).
	AvailNZBMode string `json:"availnzb_mode,omitempty"`

	LoadedPath string `json:"-"`
}

type DeviceEntry struct {
	Username         string                         `json:"username"`
	Token            string                         `json:"token"`
	IndexerOverrides map[string]IndexerSearchConfig `json:"indexer_overrides,omitempty"`
	StreamIDs        []string                       `json:"stream_ids,omitempty"`
}

type StreamEntry struct {
	ID                string                         `json:"id"`
	Name              string                         `json:"name"`
	Filters           FilterConfig                   `json:"filters"`
	Sorting           SortConfig                     `json:"sorting"`
	IndexerOverrides  map[string]IndexerSearchConfig `json:"indexer_overrides,omitempty"`
	ShowAllStream     bool                           `json:"show_all_stream"`
	PriorityGridAdded []string                       `json:"priority_grid_added,omitempty"`
}

func (c *Config) GetIncludeYearInSearch() bool { return true }

func (c *Config) GetSearchTitleLanguage() string { return "" }

func (c *Config) GetSearchTitleNormalize() bool { return false }

func MergeIndexerSearch(ic *IndexerConfig, override *IndexerSearchConfig, global *Config) *IndexerSearchConfig {
	out := &IndexerSearchConfig{}
	const defaultLimit = 1000
	out.SearchResultLimit = defaultLimit
	if ic != nil && ic.SearchResultLimit > 0 {
		out.SearchResultLimit = ic.SearchResultLimit
	}
	if override != nil && override.SearchResultLimit > 0 {
		out.SearchResultLimit = override.SearchResultLimit
	}
	val := true
	if ic != nil && ic.IncludeYearInSearch != nil {
		val = *ic.IncludeYearInSearch
	}
	if override != nil && override.IncludeYearInSearch != nil {
		val = *override.IncludeYearInSearch
	}
	out.IncludeYearInSearch = &val
	s := ""
	if ic != nil && ic.SearchTitleLanguage != "" {
		s = ic.SearchTitleLanguage
	}
	if override != nil && override.SearchTitleLanguage != nil {
		s = *override.SearchTitleLanguage
	}
	out.SearchTitleLanguage = &s
	n := false
	if ic != nil && ic.SearchTitleNormalize != nil {
		n = *ic.SearchTitleNormalize
	}
	if override != nil && override.SearchTitleNormalize != nil {
		n = *override.SearchTitleNormalize
	}
	out.SearchTitleNormalize = &n

	mc := ""
	if ic != nil {
		mc = ic.MovieCategories
	}
	if override != nil && override.MovieCategories != nil {
		mc = *override.MovieCategories
	}
	if mc != "" {
		out.MovieCategories = &mc
	}

	tc := ""
	if ic != nil {
		tc = ic.TVCategories
	}
	if override != nil && override.TVCategories != nil {
		tc = *override.TVCategories
	}
	if tc != "" {
		out.TVCategories = &tc
	}

	et := ""
	if ic != nil {
		et = ic.ExtraSearchTerms
	}
	if override != nil && override.ExtraSearchTerms != nil {
		et = *override.ExtraSearchTerms
	}
	if et != "" {
		out.ExtraSearchTerms = &et
	}

	useSE := true
	if ic != nil && ic.UseSeasonEpisodeParams != nil {
		useSE = *ic.UseSeasonEpisodeParams
	}
	if override != nil && override.UseSeasonEpisodeParams != nil {
		useSE = *override.UseSeasonEpisodeParams
	}
	out.UseSeasonEpisodeParams = &useSE
	return out
}

func (c *Config) GetAdminUsername() string {
	if c != nil && c.AdminUsername != "" {
		return c.AdminUsername
	}
	return "admin"
}

func Load() (*Config, error) {

	dataDir := paths.GetDataDir()
	configPath := filepath.Join(dataDir, "config.json")

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		logger.Warn("Failed to create data directory", "dir", dataDir, "err", err)
	}

	cfg := &Config{

		AddonPort:     7000,
		AddonBaseURL:  "http://localhost:7000",
		LogLevel:      "INFO",
		AdminUsername: "admin",
		ProxyPort:     119,
		ProxyHost:     "0.0.0.0",
		MemoryLimitMB: 512,
		KeepLogFiles:  9,
		LoadedPath:    configPath,
	}

	if err := cfg.LoadFile(configPath); err != nil {
		if os.IsNotExist(err) {
			logger.Info("No config found, creating new one", "path", configPath)
		} else {
			logger.Warn("Failed to load config, using defaults", "path", configPath, "err", err)
		}
	} else {
		logger.Info("Loaded configuration", "path", configPath)
	}
	if cfg.KeepLogFiles < 1 {
		cfg.KeepLogFiles = 9
	}

	overrides, keys := env.ReadConfigOverrides()
	ApplyEnvOverrides(cfg, overrides, keys)

	cfg.MigrateLegacyIndexers()

	needSave := cfg.ApplyProviderDefaults()

	if cfg.AdminToken == "" {
		bytes := make([]byte, 32)
		if _, err := rand.Read(bytes); err == nil {
			hash := sha256.Sum256(bytes)
			cfg.AdminToken = hex.EncodeToString(hash[:])
			needSave = true
		}
	}
	if cfg.AdminPasswordHash == "" {
		cfg.AdminPasswordHash = defaultAdminPasswordHash
		cfg.AdminMustChangePassword = true
		needSave = true
	}
	if needSave {
		logger.Info("Set default admin token/password in config")
	}

	if err := cfg.Save(); err != nil {
		logger.Warn("Failed to save config on startup", "err", err)
	} else {
		logger.Info("Saved merged configuration", "path", configPath)
	}

	if len(cfg.Providers) == 0 {
		logger.Warn("No NNTP providers configured. Add some via the web UI")
	}

	return cfg, nil
}

func (c *Config) LoadFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(c); err != nil {
		return err
	}
	return nil
}

func (c *Config) ApplyProviderDefaults() bool {
	changed := false
	for i := range c.Providers {
		p := &c.Providers[i]

		if p.Priority == nil {
			priority := i + 1
			p.Priority = &priority
			enabled := true
			p.Enabled = &enabled
			changed = true
		} else if p.Enabled == nil {

			enabled := true
			p.Enabled = &enabled
			changed = true
		}

	}
	return changed
}

func (c *Config) MigrateLegacyIndexers() {
	for i := range c.Indexers {
		if c.Indexers[i].Enabled == nil {
			enabled := true
			c.Indexers[i].Enabled = &enabled
		}
	}
}

func (c *Config) Save() error {
	path := c.LoadedPath
	if path == "" {
		path = "config.json"
	}
	return c.SaveFile(path)
}

func (c *Config) SaveFile(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(c)
}

func keySet(list []string, s string) bool {
	for _, k := range list {
		if k == s {
			return true
		}
	}
	return false
}

func ApplyEnvOverrides(cfg *Config, o env.ConfigOverrides, keys []string) {
	if keySet(keys, env.KeyAddonPort) {
		cfg.AddonPort = o.AddonPort
	}
	if keySet(keys, env.KeyAddonBaseURL) {
		cfg.AddonBaseURL = o.AddonBaseURL
	}
	if keySet(keys, env.KeyLogLevel) {
		cfg.LogLevel = o.LogLevel
	}
	if keySet(keys, env.KeyKeepLogFiles) {
		cfg.KeepLogFiles = o.KeepLogFiles
	}
	if keySet(keys, env.KeyProxyPort) {
		cfg.ProxyPort = o.ProxyPort
	}
	if keySet(keys, env.KeyProxyHost) {
		cfg.ProxyHost = o.ProxyHost
	}
	if keySet(keys, env.KeyProxyAuthUser) {
		cfg.ProxyAuthUser = o.ProxyAuthUser
	}
	if keySet(keys, env.KeyProxyAuthPass) {
		cfg.ProxyAuthPass = o.ProxyAuthPass
	}
	if keySet(keys, env.KeyAdminUsername) {
		cfg.AdminUsername = o.AdminUsername
	}
	if keySet(keys, env.KeyProviders) {
		cfg.Providers = make([]Provider, len(o.Providers))
		for i, p := range o.Providers {
			var priority *int
			var enabled *bool
			if p.Priority != nil {
				priority = p.Priority
			}
			if p.Enabled != nil {
				enabled = p.Enabled
			}
			cfg.Providers[i] = Provider{
				Name:        p.Name,
				Host:        p.Host,
				Port:        p.Port,
				Username:    p.Username,
				Password:    p.Password,
				Connections: p.Connections,
				UseSSL:      p.UseSSL,
				Priority:    priority,
				Enabled:     enabled,
			}
		}
	}
	if keySet(keys, env.KeyIndexers) {
		cfg.Indexers = make([]IndexerConfig, len(o.Indexers))
		for i, idx := range o.Indexers {
			enabled := true
			if idx.Enabled != nil {
				enabled = *idx.Enabled
			}
			cfg.Indexers[i] = IndexerConfig{
				Name:    idx.Name,
				URL:     idx.URL,
				APIKey:  idx.APIKey,
				Type:    "newznab",
				Enabled: &enabled,
			}
		}
	}
}

func GetEnvOverrideKeys() []string {
	return env.OverrideKeys()
}

func (c *Config) RedactForAPI() Config {
	out := *c
	out.AdminPasswordHash = ""
	out.AdminToken = ""
	return out
}

func CopyEnvOverridesFrom(src, dst *Config) {
	if src == nil || dst == nil {
		return
	}
	keys := env.OverrideKeys()
	for _, k := range keys {
		switch k {
		case env.KeyAddonPort:
			dst.AddonPort = src.AddonPort
		case env.KeyAddonBaseURL:
			dst.AddonBaseURL = src.AddonBaseURL
		case env.KeyLogLevel:
			dst.LogLevel = src.LogLevel
		case env.KeyKeepLogFiles:
			dst.KeepLogFiles = src.KeepLogFiles
		case env.KeyProxyPort:
			dst.ProxyPort = src.ProxyPort
		case env.KeyProxyHost:
			dst.ProxyHost = src.ProxyHost
		case env.KeyProxyAuthUser:
			dst.ProxyAuthUser = src.ProxyAuthUser
		case env.KeyProxyAuthPass:
			dst.ProxyAuthPass = src.ProxyAuthPass
		case env.KeyAdminUsername:
			dst.AdminUsername = src.AdminUsername
		case env.KeyProviders:
			dst.Providers = make([]Provider, len(src.Providers))
			for i, p := range src.Providers {
				var priority *int
				var enabled *bool
				if p.Priority != nil {
					priorityVal := *p.Priority
					priority = &priorityVal
				}
				if p.Enabled != nil {
					enabledVal := *p.Enabled
					enabled = &enabledVal
				}
				dst.Providers[i] = Provider{
					Name:        p.Name,
					Host:        p.Host,
					Port:        p.Port,
					Username:    p.Username,
					Password:    p.Password,
					Connections: p.Connections,
					UseSSL:      p.UseSSL,
					Priority:    priority,
					Enabled:     enabled,
				}
			}
		case env.KeyIndexers:
			dst.Indexers = make([]IndexerConfig, len(src.Indexers))
			copy(dst.Indexers, src.Indexers)
		}
	}
}
