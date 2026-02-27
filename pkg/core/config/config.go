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

// defaultAdminPasswordHash is SHA256("admin") - used when no admin password is set
const defaultAdminPasswordHash = "8c6976e5b5410415bde908bd4dee15dfb167a9c873fc4bb8a81f6f2ab448a918"

// Provider represents a Usenet provider configuration
type Provider struct {
	Name        string `json:"name"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	Connections int    `json:"connections"`
	UseSSL      bool   `json:"use_ssl"`
	Priority    *int   `json:"priority,omitempty"` // Lower number = higher priority (1 = first, 2 = backup, etc.). nil = not set (old config)
	Enabled     *bool  `json:"enabled,omitempty"`  // Whether this provider is enabled. nil = not set (old config)
}

// FilterConfig and SortConfig are defined in config_filter_sort.go (Include/Avoid and Order lists with legacy JSON migration).

// DefaultFilterConfig returns built-in filter defaults (Avoid CAM/TeleSync).
func DefaultFilterConfig() FilterConfig {
	return FilterConfig{
		QualityAvoid: []string{"CAM", "TeleSync"},
		DubbedAvoid:  ptrBool(true),
	}
}

func DefaultSortConfig() SortConfig {
	return SortConfig{
		ResolutionOrder:   []string{"4k", "1080p", "720p", "sd"},
		QualityOrder:      []string{"BluRay", "Blu-ray", "WEB-DL", "WEBRip", "HDTV"},
		SortCriteriaOrder: []string{"resolution", "quality", "codec", "visual_tag", "audio", "channels"},
		GrabWeight:        0.5,
		AgeWeight:         1.0,
	}
}

func ptrBool(b bool) *bool { return &b }

// IndexerSearchConfig holds indexer/search overrides (per-device per-indexer). Pointers = nil means use indexer's value. Used in Device.IndexerOverrides and API payloads.
type IndexerSearchConfig struct {
	SearchResultLimit      int     `json:"search_result_limit,omitempty"` // 0 = use indexer/global
	IncludeYearInSearch    *bool   `json:"include_year_in_search,omitempty"`
	SearchTitleLanguage    *string `json:"search_title_language,omitempty"`
	SearchTitleNormalize   *bool   `json:"search_title_normalize,omitempty"`
	MovieCategories        *string `json:"movie_categories,omitempty"`
	TVCategories           *string `json:"tv_categories,omitempty"`
	ExtraSearchTerms       *string `json:"extra_search_terms,omitempty"`
	UseSeasonEpisodeParams *bool   `json:"use_season_episode_params,omitempty"`
}

// IndexerConfig represents an internal Newznab indexer configuration
type IndexerConfig struct {
	Name         string `json:"name"`
	URL          string `json:"url"`
	APIKey       string `json:"api_key"`
	APIPath      string `json:"api_path"` // API path (default: "/api"), e.g., "/api" or "/api/v1"
	Type         string `json:"type"`     // "newznab", "easynews"
	APIHitsDay   int    `json:"api_hits_day"`
	DownloadsDay int    `json:"downloads_day"`
	Enabled      *bool  `json:"enabled,omitempty"` // Whether this indexer is enabled. nil = not set (old config)
	// Easynews-specific fields
	Username string `json:"username"` // Easynews username
	Password string `json:"password"` // Easynews password
	// Per-indexer search settings (all 8; empty/nil = use global default where applicable)
	MovieCategories        string `json:"movie_categories,omitempty"`
	TVCategories           string `json:"tv_categories,omitempty"`
	ExtraSearchTerms       string `json:"extra_search_terms,omitempty"`
	UseSeasonEpisodeParams *bool  `json:"use_season_episode_params,omitempty"`
	SearchResultLimit      int    `json:"search_result_limit,omitempty"` // 0 = use global default
	IncludeYearInSearch    *bool  `json:"include_year_in_search,omitempty"`
	SearchTitleLanguage    string `json:"search_title_language,omitempty"`
	SearchTitleNormalize   *bool  `json:"search_title_normalize,omitempty"`
}

// Config holds application configuration
type Config struct {
	// Internal Indexers
	Indexers []IndexerConfig `json:"indexers"`

	// Addon settings
	AddonPort    int    `json:"addon_port"`
	AddonBaseURL string `json:"addon_base_url"`
	LogLevel     string `json:"log_level"`

	// Dashboard admin: stored in config.json (never send hash/token to frontend)
	AdminUsername           string `json:"admin_username"`
	AdminPasswordHash       string `json:"admin_password_hash"` // SHA256 hash; do not send to API clients
	AdminMustChangePassword bool   `json:"admin_must_change_password"`
	AdminToken              string `json:"admin_token"` // Single token for dashboard + streaming; do not send to API clients

	// NNTP Providers
	Providers []Provider `json:"providers"`

	// NNTP Proxy (always enabled when configured)
	ProxyPort     int    `json:"proxy_port"`
	ProxyHost     string `json:"proxy_host"`
	ProxyAuthUser string `json:"proxy_auth_user"`
	ProxyAuthPass string `json:"proxy_auth_pass"`

	// AvailNZB (Internal/Community)
	AvailNZBURL    string `json:"-"`
	AvailNZBAPIKey string `json:"-"`

	// TMDB Settings
	TMDBAPIKey string `json:"-"`

	// TVDB Settings
	TVDBAPIKey string `json:"-"`

	// Devices (dashboard/Stremio tokens; persisted in config.json)
	Devices map[string]*DeviceEntry `json:"devices,omitempty"`

	// Streams (playback configs; persisted in config.json)
	Streams []*StreamEntry `json:"streams,omitempty"`

	// Internal - where was this config loaded from?
	LoadedPath string `json:"-"`
}

// DeviceEntry is the persisted shape of a device (auth.Device) in config.json.
type DeviceEntry struct {
	Username         string                         `json:"username"`
	Token            string                         `json:"token"`
	IndexerOverrides map[string]IndexerSearchConfig `json:"indexer_overrides,omitempty"`
}

// StreamEntry is the persisted shape of a stream in config.json (mirrors stream.Stream).
type StreamEntry struct {
	ID                string                         `json:"id"`
	Name              string                         `json:"name"`
	Filters           FilterConfig                   `json:"filters"`
	Sorting           SortConfig                     `json:"sorting"`
	IndexerOverrides  map[string]IndexerSearchConfig `json:"indexer_overrides,omitempty"`
	ShowAllStream     bool                           `json:"show_all_stream"`
	PriorityGridAdded []string                       `json:"priority_grid_added,omitempty"` // optional category keys added to the grid (e.g. "quality", "codec")
}

// GetIncludeYearInSearch returns the default for including year in movie search (used when no per-indexer config).
func (c *Config) GetIncludeYearInSearch() bool { return true }

// GetSearchTitleLanguage returns the default TMDB language for movie title.
func (c *Config) GetSearchTitleLanguage() string { return "" }

// GetSearchTitleNormalize returns the default for normalizing movie title.
func (c *Config) GetSearchTitleNormalize() bool { return false }

// MergeIndexerSearch merges per-indexer config and per-device override; uses built-in defaults for limit/year/language/normalize.
// Returns a fully populated IndexerSearchConfig for use when building the search request per indexer.
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
	// MovieCategories
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
	// TVCategories
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
	// ExtraSearchTerms
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
	// UseSeasonEpisodeParams (nil/true = send)
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

// GetAdminUsername returns the dashboard admin login username (default "admin").
func (c *Config) GetAdminUsername() string {
	if c != nil && c.AdminUsername != "" {
		return c.AdminUsername
	}
	return "admin"
}

// Load is intended for startup only. It loads configuration from config.json,
// applies environment variable overrides once, then saves the merged config.
// Environment variables are not read again after startup; subsequent reloads
// use only the saved config.
//
// Priority: Environment variables (if set) > config.json > defaults.
// When saving from the UI, values for keys that have an env override are preserved
// from the current effective config (so the file is not overwritten with form values
// that would be overridden by env on next restart). See CopyEnvOverridesFrom.
func Load() (*Config, error) {
	// 1. Determine config path using common data directory function
	dataDir := paths.GetDataDir()
	configPath := filepath.Join(dataDir, "config.json")
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		logger.Warn("Failed to create data directory", "dir", dataDir, "err", err)
	}

	// 2. Load config.json (or create with defaults if it doesn't exist)
	cfg := &Config{
		// Set defaults
		AddonPort:     7000,
		AddonBaseURL:  "http://localhost:7000",
		LogLevel:      "INFO",
		AdminUsername: "admin",
		ProxyPort:     119,
		ProxyHost:     "0.0.0.0",
		LoadedPath:    configPath,
	}

	// Try to load existing config
	if err := cfg.LoadFile(configPath); err != nil {
		if os.IsNotExist(err) {
			logger.Info("No config found, creating new one", "path", configPath)
		} else {
			logger.Warn("Failed to load config, using defaults", "path", configPath, "err", err)
		}
	} else {
		logger.Info("Loaded configuration", "path", configPath)
	}

	// 3. Override with environment variables (single source: pkg/env)
	overrides, keys := env.ReadConfigOverrides()
	ApplyEnvOverrides(cfg, overrides, keys)

	// 4. Migrate legacy indexers
	cfg.MigrateLegacyIndexers()

	// 4.3. Apply provider defaults (priority and enabled)
	needSave := cfg.ApplyProviderDefaults()

	// 4.5. Ensure admin token and password hash defaults (do not overwrite if already set)
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

	// 5. Save the merged configuration
	if err := cfg.Save(); err != nil {
		logger.Warn("Failed to save config on startup", "err", err)
	} else {
		logger.Info("Saved merged configuration", "path", configPath)
	}

	// Warn if no providers configured
	if len(cfg.Providers) == 0 {
		logger.Warn("No NNTP providers configured. Add some via the web UI")
	}

	return cfg, nil
}

// LoadFile overrides config with values from a JSON file
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

// ApplyProviderDefaults migrates old provider configs to new format (priority and enabled).
// This is a one-time migration: if priority is nil (not set), treat as old config and set defaults.
// Returns true if any changes were made (indicating config should be saved).
func (c *Config) ApplyProviderDefaults() bool {
	changed := false
	for i := range c.Providers {
		p := &c.Providers[i]
		// Migration: if priority is nil (not set in JSON), it's an old config - migrate to new format
		if p.Priority == nil {
			priority := i + 1
			p.Priority = &priority
			enabled := true
			p.Enabled = &enabled
			changed = true
		} else if p.Enabled == nil {
			// Priority was set but enabled is nil - also migrate enabled
			enabled := true
			p.Enabled = &enabled
			changed = true
		}
		// If both priority and enabled are set, it's already migrated - respect values as-is
	}
	return changed
}

// MigrateLegacyIndexers applies any one-off config migrations for indexers.
func (c *Config) MigrateLegacyIndexers() {
	for i := range c.Indexers {
		if c.Indexers[i].Enabled == nil {
			enabled := true
			c.Indexers[i].Enabled = &enabled
		}
	}
}

// Save saves the current configuration to the file it was loaded from
func (c *Config) Save() error {
	path := c.LoadedPath
	if path == "" {
		path = "config.json"
	}
	return c.SaveFile(path)
}

// SaveFile saves the current configuration to a JSON file
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

// keySet returns true if s is in list.
func keySet(list []string, s string) bool {
	for _, k := range list {
		if k == s {
			return true
		}
	}
	return false
}

// ApplyEnvOverrides applies environment-derived overrides to cfg (used at startup only).
// Only fields present in keys are applied, so env vars override file values per setting.
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

// GetEnvOverrideKeys returns config JSON keys that have environment variable overrides set.
// Used by the UI to show "overwritten on restart" warnings and when saving to preserve
// those values so the file is not overwritten with form data that env would override.
func GetEnvOverrideKeys() []string {
	return env.OverrideKeys()
}

// RedactForAPI returns a copy of the config with AdminPasswordHash and AdminToken cleared.
// Use when sending config to the frontend so sensitive values are never exposed.
func (c *Config) RedactForAPI() Config {
	out := *c
	out.AdminPasswordHash = ""
	out.AdminToken = ""
	return out
}

// CopyEnvOverridesFrom copies into dst the effective values for any key that has an
// environment override (from GetEnvOverrideKeys). Call before saving config from the UI
// so that env/ldflag-derived values are not overwritten by the form payload; the file
// then keeps the current effective values for those keys and env still wins on restart.
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
