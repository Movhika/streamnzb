package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"streamnzb/pkg/auth"
	"streamnzb/pkg/core/app"
	"streamnzb/pkg/core/config"
	"streamnzb/pkg/core/env"
	"streamnzb/pkg/core/logger"
	"streamnzb/pkg/core/persistence"
	"streamnzb/pkg/initialization"
	"streamnzb/pkg/server/api"
	"streamnzb/pkg/server/stremio"
	"streamnzb/pkg/server/web"
	"streamnzb/pkg/session"
	"streamnzb/pkg/stream"
	"streamnzb/pkg/usenet/nntp/proxy"

	"github.com/joho/godotenv"
)

var (
	// AvailNZB configuration set at build time via -ldflags
	AvailNZBURL    = ""
	AvailNZBAPIKey = ""

	// TMDB Key via ldflags
	TMDBKey = ""
	// TVDB Key via ldflags
	TVDBKey = ""

	// Version set at build time via -ldflags (from release-please tag, e.g. v1.0.0)
	Version = "dev"
)

func main() {
	// Load environment variables for logger and bootstrap
	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found, using environment variables")
	}

	// Default User-Agent for indexer requests (overridable via INDEXER_QUERY_HEADER / INDEXER_GRAB_HEADER)
	env.DefaultIndexerUserAgent = "StreamNZB/" + Version

	// Initialize Logger early so bootstrap can use it
	logger.Init(env.LogLevel())

	logger.Info("Starting StreamNZB", "version", Version)

	// Bootstrap application
	cfg, err := config.Load()
	if err != nil {
		initialization.WaitForInputAndExit(fmt.Errorf("configuration error: %w", err))
	}
	logger.SetLevel(cfg.LogLevel)

	availNZBUrl := os.Getenv(env.AvailNZBURL)
	if availNZBUrl == "" {
		availNZBUrl = AvailNZBURL
	}
	availNZBAPIKey := os.Getenv(env.AvailNZBAPIKey)
	if availNZBAPIKey == "" {
		availNZBAPIKey = AvailNZBAPIKey
	}
	tmdbKey := os.Getenv(env.TMDBAPIKey)
	if tmdbKey == "" {
		tmdbKey = TMDBKey
	}
	tvdbKey := os.Getenv(env.TVDBAPIKey)
	if tvdbKey == "" {
		tvdbKey = TVDBKey
	}

	dataDir := filepath.Dir(cfg.LoadedPath)
	if dataDir == "" || dataDir == "." {
		dataDir, _ = os.Getwd()
	}

	stateMgr, err := persistence.GetManager(dataDir)
	if err != nil {
		initialization.WaitForInputAndExit(fmt.Errorf("failed to get state manager: %v", err))
	}

	// Migrate admin from state.json to config.json (one-time)
	{
		var stateAdmin struct {
			PasswordHash       string `json:"password_hash"`
			MustChangePassword bool   `json:"must_change_password"`
		}
		if found, _ := stateMgr.Get("admin", &stateAdmin); found {
			cfg.AdminPasswordHash = stateAdmin.PasswordHash
			cfg.AdminMustChangePassword = stateAdmin.MustChangePassword
			if cfg.AdminToken == "" {
				if tok, err := auth.GenerateToken(); err == nil {
					cfg.AdminToken = tok
				}
			}
			if err := cfg.Save(); err != nil {
				logger.Warn("Failed to save config after admin migration", "err", err)
			} else {
				stateMgr.Delete("admin")
				stateMgr.Delete("admin_sessions")
				_ = stateMgr.Flush()
				logger.Info("Migrated admin credentials from state.json to config.json")
			}
		}
	}

	// Migrate devices and streams from state.json to config.json (one-time)
	{
		if len(cfg.Devices) == 0 {
			var stateDevices map[string]*auth.Device
			if found, _ := stateMgr.Get("devices", &stateDevices); found && len(stateDevices) > 0 {
				cfg.Devices = make(map[string]*config.DeviceEntry)
				for k, d := range stateDevices {
					if d == nil {
						continue
					}
					ov := d.IndexerOverrides
					if ov == nil {
						ov = make(map[string]config.IndexerSearchConfig)
					}
					cfg.Devices[k] = &config.DeviceEntry{
						Username:         d.Username,
						Token:            d.Token,
						IndexerOverrides: ov,
					}
				}
				if err := cfg.Save(); err != nil {
					logger.Warn("Failed to save config after devices migration", "err", err)
				} else {
					stateMgr.Delete("devices")
					stateMgr.Delete("users")
					_ = stateMgr.Flush()
					logger.Info("Migrated devices from state.json to config.json")
				}
			}
		}
		if len(cfg.Streams) == 0 {
			var stateStreams []*stream.Stream
			if found, _ := stateMgr.Get("streams", &stateStreams); found && len(stateStreams) > 0 {
				cfg.Streams = make([]*config.StreamEntry, 0, len(stateStreams))
				for _, s := range stateStreams {
					if s == nil || s.ID == "" {
						continue
					}
					ov := s.IndexerOverrides
					if ov == nil {
						ov = make(map[string]config.IndexerSearchConfig)
					}
					cfg.Streams = append(cfg.Streams, &config.StreamEntry{
						ID:               s.ID,
						Name:             s.Name,
						Filters:          s.Filters,
						Sorting:          s.Sorting,
						IndexerOverrides: ov,
						ShowAllStream:    s.ShowAllStream,
					})
				}
				if err := cfg.Save(); err != nil {
					logger.Warn("Failed to save config after streams migration", "err", err)
				} else {
					stateMgr.Delete("streams")
					_ = stateMgr.Flush()
					logger.Info("Migrated streams from state.json to config.json")
				}
			}
		}
	}

	// Centralized app container - builds all components
	application := app.New()
	comp, err := application.Build(cfg, app.BuildOpts{
		AvailNZBURL:    availNZBUrl,
		AvailNZBAPIKey: availNZBAPIKey,
		TMDBAPIKey:     tmdbKey,
		TVDBAPIKey:     tvdbKey,
		DataDir:        dataDir,
		SessionTTL:     30 * time.Minute,
	})
	if err != nil {
		initialization.WaitForInputAndExit(fmt.Errorf("failed to build components: %w", err))
	}

	sessionManager := session.NewManager(comp.StreamingPools, 30*time.Minute)
	logger.Info("Session manager initialized", "ttl", 30*time.Minute)

	saveConfig := func() error { return cfg.Save() }
	deviceManager, err := auth.NewDeviceManagerFromConfig(cfg, saveConfig)
	if err != nil {
		initialization.WaitForInputAndExit(fmt.Errorf("failed to initialize device manager: %v", err))
	}
	streamManager, err := stream.NewManagerFromConfig(cfg, saveConfig)
	if err != nil {
		initialization.WaitForInputAndExit(fmt.Errorf("failed to initialize stream manager: %v", err))
	}

	stremioServer, err := stremio.NewServer(&stremio.ServerOptions{
		Config:               comp.Config,
		BaseURL:              comp.Config.AddonBaseURL,
		Port:                 comp.Config.AddonPort,
		Indexer:              comp.Indexer,
		Validator:            comp.Validator,
		SessionManager:       sessionManager,
		TriageService:        comp.Triage,
		AvailClient:          comp.AvailClient,
		AvailNZBIndexerHosts: comp.AvailNZBIndexerHosts,
		TMDBClient:           comp.TMDBClient,
		TVDBClient:            comp.TVDBClient,
		DeviceManager:        deviceManager,
		StreamManager:        streamManager,
		Version:              Version,
	})
	if err != nil {
		initialization.WaitForInputAndExit(fmt.Errorf("failed to initialize Stremio server: %v", err))
	}

	apiServer := api.NewServerWithApp(comp.Config, comp.ProviderPools, sessionManager, stremioServer, comp.Indexer, deviceManager, streamManager, application, availNZBUrl, availNZBAPIKey, tmdbKey, tvdbKey)
	apiServer.SetIndexerCaps(comp.IndexerCaps)

	// Set embedded web handler
	stremioServer.SetWebHandler(web.Handler())
	stremioServer.SetAPIHandler(apiServer.Handler())

	// Setup HTTP routes
	mux := http.NewServeMux()
	stremioServer.SetupRoutes(mux)

	// Mount API routes (apiServer.Handler returns a mux with /api/...)
	// Since both are muxes, we need to merge or mount carefully.
	// StremioServer mounts "/" at the end.
	// We should mount /api/ before /.
	mux.Handle("/api/", apiServer.Handler())

	// Start NNTP proxy (always enabled)
	{
		proxyServer, err := proxy.NewServer(comp.Config.ProxyHost, comp.Config.ProxyPort, comp.StreamingPools, comp.Config.ProxyAuthUser, comp.Config.ProxyAuthPass)
		if err != nil {
			initialization.WaitForInputAndExit(fmt.Errorf("failed to initialize NNTP proxy: %v", err))
		}

		apiServer.SetProxyServer(proxyServer)

		go func() {
			logger.Info("Starting NNTP proxy", "host", comp.Config.ProxyHost, "port", comp.Config.ProxyPort)
			if err := proxyServer.Start(); err != nil {
				initialization.WaitForInputAndExit(fmt.Errorf("nntp proxy failed: %w", err))
			}
		}()
	}

	// Start Stremio server
	addr := fmt.Sprintf(":%d", comp.Config.AddonPort)

	logger.Info("Stremio addon server starting", "base_url", comp.Config.AddonBaseURL, "port", comp.Config.AddonPort)
	logger.Info("Note: Access requires device authentication tokens")

	if err := http.ListenAndServe(addr, mux); err != nil {
		initialization.WaitForInputAndExit(fmt.Errorf("server failed: %w", err))
	}
}
