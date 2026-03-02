package app

import (
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"streamnzb/pkg/core/config"
	"streamnzb/pkg/core/logger"
	"streamnzb/pkg/indexer"
	"streamnzb/pkg/initialization"
	"streamnzb/pkg/search/triage"
	"streamnzb/pkg/services/availnzb"
	"streamnzb/pkg/services/metadata/tmdb"
	"streamnzb/pkg/services/metadata/tvdb"
	"streamnzb/pkg/usenet/nntp"
	"streamnzb/pkg/usenet/pool"
	"streamnzb/pkg/usenet/validation"
)

type BuildOpts struct {
	AvailNZBURL    string
	AvailNZBAPIKey string
	TMDBAPIKey     string
	TVDBAPIKey     string
	DataDir        string
	SessionTTL     time.Duration
}

type Components struct {
	Config               *config.Config
	Indexer              indexer.Indexer
	ProviderPools        map[string]*nntp.ClientPool
	ProviderOrder        []string
	StreamingPools       []*nntp.ClientPool
	UsenetPool           *pool.Pool
	AvailNZBIndexerHosts []string
	IndexerCaps          map[string]*indexer.Caps
	Validator            *validation.Checker
	Triage               *triage.Service
	AvailClient          *availnzb.Client
	TMDBClient           *tmdb.Client
	TVDBClient           *tvdb.Client
}

type App struct {
	mu         sync.RWMutex
	components *Components
	opts       BuildOpts
}

func New() *App {
	return &App{}
}

func (a *App) Build(cfg *config.Config, opts BuildOpts) (*Components, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.opts = opts

	comp, err := a.buildFull(cfg, opts)
	if err != nil {
		return nil, err
	}
	a.components = comp
	return comp, nil
}

func (a *App) buildFull(cfg *config.Config, opts BuildOpts) (*Components, error) {
	base, err := initialization.BuildComponents(cfg)
	if err != nil {
		return nil, err
	}

	const validationSampleSize = 5
	validator := validation.NewChecker(base.UsenetPool, validationSampleSize, 6)
	defaultFilters := config.DefaultFilterConfig()
	defaultSorting := config.DefaultSortConfig()
	triageSvc := triage.NewService(&defaultFilters, defaultSorting)
	availClient := availnzb.NewClient(opts.AvailNZBURL, opts.AvailNZBAPIKey)
	if err := availClient.RefreshBackbones(); err != nil {
		logger.Debug("AvailNZB backbones refresh on start", "err", err)
	}
	dataDir := opts.DataDir
	if dataDir == "" {
		dataDir = filepath.Dir(cfg.LoadedPath)
	}
	if dataDir == "" || dataDir == "." {
		dataDir, _ = filepath.Abs(".")
	}
	tmdbClient := tmdb.NewClient(opts.TMDBAPIKey)
	tvdbClient := tvdb.NewClient(opts.TVDBAPIKey, dataDir)

	return &Components{
		Config:               base.Config,
		Indexer:              base.Indexer,
		ProviderPools:        base.ProviderPools,
		ProviderOrder:        base.ProviderOrder,
		StreamingPools:       base.StreamingPools,
		UsenetPool:           base.UsenetPool,
		AvailNZBIndexerHosts: base.AvailNZBIndexerHosts,
		IndexerCaps:          base.IndexerCaps,
		Validator:            validator,
		Triage:               triageSvc,
		AvailClient:          availClient,
		TMDBClient:           tmdbClient,
		TVDBClient:           tvdbClient,
	}, nil
}

type ReloadScope int

const (
	ReloadConfigOnly ReloadScope = iota
	ReloadIndexers
	ReloadProviders
	ReloadProxy
	ReloadFull
)

func ConfigChanged(old, new_ *config.Config) ReloadScope {
	if old == nil || new_ == nil {
		return ReloadFull
	}

	indexersChanged := !reflect.DeepEqual(old.Indexers, new_.Indexers)
	providersChanged := !reflect.DeepEqual(old.Providers, new_.Providers)
	proxyChanged := old.ProxyHost != new_.ProxyHost ||
		old.ProxyPort != new_.ProxyPort ||
		old.ProxyAuthUser != new_.ProxyAuthUser ||
		old.ProxyAuthPass != new_.ProxyAuthPass

	if providersChanged || indexersChanged {
		return ReloadFull
	}
	if proxyChanged {
		return ReloadFull
	}
	return ReloadConfigOnly
}

func (a *App) Reload(newCfg *config.Config) (*Components, bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	old := a.components
	scope := ConfigChanged(old.Config, newCfg)

	switch scope {
	case ReloadConfigOnly:

		logger.Info("Reload: config-only - no NNTP/indexer restart")
		defaultFilters := config.DefaultFilterConfig()
		defaultSorting := config.DefaultSortConfig()
		triageSvc := triage.NewService(&defaultFilters, defaultSorting)
		comp := *old
		comp.Config = newCfg
		comp.Triage = triageSvc
		a.components = &comp
		return &comp, false, nil

	case ReloadFull:
		logger.Info("Reload: full rebuild (indexers or providers changed)")
		comp, err := a.buildFull(newCfg, a.opts)
		if err != nil {
			return nil, true, err
		}
		a.components = comp
		return comp, true, nil

	case ReloadProxy:
		logger.Info("Reload: proxy config changed")
		comp, err := a.buildFull(newCfg, a.opts)
		if err != nil {
			return nil, true, err
		}
		a.components = comp
		return comp, true, nil

	default:
		logger.Info("Reload: indexers changed - full rebuild")
		comp, err := a.buildFull(newCfg, a.opts)
		if err != nil {
			return nil, true, err
		}
		a.components = comp
		return comp, true, nil
	}
}

func (a *App) Components() *Components {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.components
}
