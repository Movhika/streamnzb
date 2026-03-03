package stremio

import (
	"bufio"
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"streamnzb/pkg/auth"
	"streamnzb/pkg/core/config"
	"streamnzb/pkg/core/logger"
	"streamnzb/pkg/indexer"
	"streamnzb/pkg/media/loader"
	"streamnzb/pkg/media/nzb"
	"streamnzb/pkg/media/seek"
	"streamnzb/pkg/media/unpack"
	"streamnzb/pkg/release"
	"streamnzb/pkg/search"
	"streamnzb/pkg/search/parser"
	"streamnzb/pkg/search/triage"
	"streamnzb/pkg/services/availnzb"
	"streamnzb/pkg/services/metadata/tmdb"
	"streamnzb/pkg/services/metadata/tvdb"
	"streamnzb/pkg/session"
	"streamnzb/pkg/stream"
	"streamnzb/pkg/usenet/validation"
)

var (
	availTrue  = true
	availFalse = false
)

type Server struct {
	mu                   sync.RWMutex
	manifest             *Manifest
	version              string
	baseURL              string
	config               *config.Config
	indexer              indexer.Indexer
	validator            *validation.Checker
	sessionManager       *session.Manager
	triageService        *triage.Service
	availClient          *availnzb.Client
	availReporter        *availnzb.Reporter
	availNZBIndexerHosts []string
	tmdbClient           *tmdb.Client
	tvdbClient           *tvdb.Client
	deviceManager        *auth.DeviceManager
	streamManager        *stream.Manager
	recentFailures       sync.Map
	playListCache        sync.Map
	rawSearchCache       sync.Map
	nextReleaseIndex     sync.Map
	webHandler           http.Handler
	apiHandler           http.Handler
}

const FailoverOrderPath = "/failover_order"

type ServerOptions struct {
	Config               *config.Config
	BaseURL              string
	Port                 int
	Indexer              indexer.Indexer
	Validator            *validation.Checker
	SessionManager       *session.Manager
	TriageService        *triage.Service
	AvailClient          *availnzb.Client
	AvailNZBIndexerHosts []string
	TMDBClient           *tmdb.Client
	TVDBClient           *tvdb.Client
	DeviceManager        *auth.DeviceManager
	StreamManager        *stream.Manager
	Version              string
}

func NewServer(opts *ServerOptions) (*Server, error) {
	if opts == nil {
		return nil, fmt.Errorf("ServerOptions is required")
	}
	version := opts.Version
	if version == "" {
		version = "dev"
	}
	var availReporter *availnzb.Reporter
	if opts.AvailClient != nil {
		availReporter = availnzb.NewReporter(opts.AvailClient, opts.Validator)
	}
	s := &Server{
		manifest:             NewManifest(version),
		version:              version,
		baseURL:              opts.BaseURL,
		config:               opts.Config,
		indexer:              opts.Indexer,
		validator:            opts.Validator,
		sessionManager:       opts.SessionManager,
		triageService:        opts.TriageService,
		availClient:          opts.AvailClient,
		availReporter:        availReporter,
		availNZBIndexerHosts: opts.AvailNZBIndexerHosts,
		tmdbClient:           opts.TMDBClient,
		tvdbClient:           opts.TVDBClient,
		deviceManager:        opts.DeviceManager,
		streamManager:        opts.StreamManager,
	}

	if err := s.CheckPort(opts.Port); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Server) CheckPort(port int) error {
	address := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("addon port %d is already in use", port)
	}
	ln.Close()
	return nil
}

func (s *Server) SetWebHandler(h http.Handler) {
	s.webHandler = h
}

func (s *Server) SetAPIHandler(h http.Handler) {
	s.apiHandler = h
}

func (s *Server) Version() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.version != "" {
		return s.version
	}
	return "dev"
}

func (s *Server) SetupRoutes(mux *http.ServeMux) {

	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.RLock()
		deviceManager := s.deviceManager
		webHandler := s.webHandler
		apiHandler := s.apiHandler
		s.mu.RUnlock()

		path := r.URL.Path
		var authenticatedDevice *auth.Device

		if path == "/error/failure.mp4" && webHandler != nil {
			webHandler.ServeHTTP(w, r)
			return
		}

		isStremioRoute := path == "/manifest.json" || path == FailoverOrderPath || strings.HasPrefix(path, "/stream/") || strings.HasPrefix(path, "/play/") || strings.HasPrefix(path, "/next/") || strings.HasPrefix(path, "/debug/play")

		trimmedPath := strings.TrimPrefix(path, "/")
		parts := strings.SplitN(trimmedPath, "/", 2)

		if len(parts) >= 1 && parts[0] != "" {
			token := parts[0]

			if deviceManager != nil {
				device, err := deviceManager.AuthenticateToken(token, s.config.GetAdminUsername(), s.config.AdminToken)
				if err == nil && device != nil {
					authenticatedDevice = device

					if len(parts) > 1 {
						path = "/" + parts[1]
					} else {
						path = "/"
					}
					r.URL.Path = path

					r = r.WithContext(auth.ContextWithDevice(r.Context(), device))
				} else if isStremioRoute {

					logger.Error("Unauthorized request - invalid device token", "path", path, "remote", r.RemoteAddr)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}

			}
		} else if isStremioRoute {

			logger.Error("Unauthorized request - Stremio route requires device token", "path", path, "remote", r.RemoteAddr)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if path == "/manifest.json" {
			s.handleManifest(w, r)
		} else if strings.HasPrefix(path, "/stream/") {
			s.handleStream(w, r, authenticatedDevice)
		} else if strings.HasPrefix(path, "/play/") {
			s.handlePlay(w, r, authenticatedDevice)
		} else if strings.HasPrefix(path, "/next/") {
			s.handleNextRelease(w, r, authenticatedDevice)
		} else if path == FailoverOrderPath {
			s.handleFailoverOrder(w, r, authenticatedDevice)
		} else if strings.HasPrefix(path, "/debug/play") {
			s.handleDebugPlay(w, r, authenticatedDevice)
		} else if path == "/health" {
			s.handleHealth(w, r)
		} else if strings.HasPrefix(path, "/api/") {
			if apiHandler != nil {

				apiHandler.ServeHTTP(w, r)
			} else {
				http.NotFound(w, r)
			}
		} else {
			if webHandler != nil {
				webHandler.ServeHTTP(w, r)
			} else {
				http.NotFound(w, r)
			}
		}
	})

	mux.Handle("/", finalHandler)
}

func (s *Server) handleManifest(w http.ResponseWriter, r *http.Request) {
	logger.Debug("Manifest request", "remote", r.RemoteAddr)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	s.mu.RLock()
	manifest := s.manifest
	s.mu.RUnlock()

	device, _ := auth.DeviceFromContext(r)
	isAdmin := device != nil && device.Username == s.config.GetAdminUsername()

	data, err := manifest.ToJSONForDevice(isAdmin)
	if err != nil {
		http.Error(w, "Failed to generate manifest", http.StatusInternalServerError)
		return
	}

	w.Write(data)
}

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request, device *auth.Device) {

	path := strings.TrimPrefix(r.URL.Path, "/stream/")
	path = strings.TrimSuffix(path, ".json")

	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		http.Error(w, "Invalid stream URL", http.StatusBadRequest)
		return
	}

	contentType := parts[0]
	id := parts[1]

	logger.Info("Stream request", "type", contentType, "id", id, "device", func() string {
		if device != nil {
			return device.Username
		}
		return "legacy"
	}())

	const streamRequestTimeout = 30 * time.Second
	ctx, cancel := context.WithTimeout(r.Context(), streamRequestTimeout)
	defer cancel()

	logger.Trace("stream request start", "type", contentType, "id", id)
	isAIOStreams := strings.Contains(r.Header.Get("User-Agent"), "AIOStreams")
	var streams []Stream
	streamsList := s.streamConfigsForStreamRequest()
	for _, str := range streamsList {
		key := StreamSlotKey{StreamID: str.ID, ContentType: contentType, ID: id}
		list, err := s.buildOrderedPlayList(ctx, key, isAIOStreams)
		if err != nil {
			logger.Error("Error building play list", "streamId", str.ID, "err", err)
			continue
		}
		if list == nil {
			continue
		}
		if len(list.Candidates) == 0 {
			continue
		}
		s.clearNextReleaseBound(device, key)
		baseURL := s.baseURLWithToken(device)
		streams = append(streams, buildStreamsFromPlayList(list, key, str.Name, baseURL, str.ShowAllStream || isAIOStreams)...)
		logger.Debug("Stream rows", "streamId", str.ID, "name", str.Name, "candidates", len(list.Candidates), "showAllStream", str.ShowAllStream)
	}
	if streams == nil {
		streams = []Stream{}
	}

	response := StreamResponse{
		Streams: streams,
	}

	responseJSON, _ := json.MarshalIndent(response, "", "  ")
	logger.Trace("Sending stream response", "json", string(responseJSON))

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	json.NewEncoder(w).Encode(response)
}

type failoverOrderRequest struct {
	Streams []struct {
		FailoverID string `json:"failoverId"`
	} `json:"streams"`
}

func (s *Server) handleFailoverOrder(w http.ResponseWriter, r *http.Request, device *auth.Device) {
	logger.Debug("Failover order request", "device", device.Username, "method", r.Method, "url", r.URL.Path)
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	var req failoverOrderRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if len(req.Streams) == 0 {
		logger.Debug("Failover order: empty streams array, not storing")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	s.mu.RLock()
	sm := s.streamManager
	s.mu.RUnlock()
	var order []string
	for _, entry := range req.Streams {
		raw := strings.TrimSpace(entry.FailoverID)
		if raw == "" {
			continue
		}
		slotOrID := raw
		if after, ok := strings.CutPrefix(raw, "streamnzb-"); ok {
			slotOrID = after
		}

		if !strings.HasPrefix(slotOrID, streamSlotPrefix) {
			if sm != nil {
				if str, _ := sm.Get(slotOrID); str == nil {
					continue
				}
			}
		}
		order = append(order, slotOrID)
	}
	if len(order) == 0 {
		var firstNonEmptyRaw string
		nonEmptyCount := 0
		for _, e := range req.Streams {
			trimmed := strings.TrimSpace(e.FailoverID)
			if trimmed != "" {
				nonEmptyCount++
				if firstNonEmptyRaw == "" {
					firstNonEmptyRaw = trimmed
					if len(firstNonEmptyRaw) > 80 {
						firstNonEmptyRaw = firstNonEmptyRaw[:80] + "..."
					}
				}
			}
		}
		devKey := ""
		if device != nil {
			devKey = device.Token
		}
		logger.Info("Failover order: no entries stored (all skipped)", "device", devKey, "requested", len(req.Streams), "nonEmptyFailoverIds", nonEmptyCount, "firstFailoverIdSample", firstNonEmptyRaw)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	key := ""
	if device != nil {
		key = device.Token
	}
	s.sessionManager.SetDeviceFailoverOrder(key, order)
	sample := ""
	if len(order) > 0 {
		sample = order[0]
		if len(order) > 1 {
			sample += " ... " + order[len(order)-1]
		}
	}
	usingPaths := len(order) > 0 && strings.HasPrefix(order[0], streamSlotPrefix)
	logger.Info("Failover order stored", "device", key, "slots", len(order), "usingSlotPaths", usingPaths, "sample", sample)
	w.WriteHeader(http.StatusNoContent)
}

func bingeGroupLabelFromMeta(meta *parser.ParsedRelease) string {
	if meta == nil {
		return ""
	}
	if g := meta.ResolutionGroup(); g != "" && g != "sd" {
		return g
	}
	if meta.Resolution != "" {
		return meta.Resolution
	}
	return meta.Quality
}

func streamBehaviorHints(streamName, streamID string, rel *release.Release, cached *bool, bingeGroupLabel string) *BehaviorHints {
	bingeGroup := "streamnzb-" + streamID
	if bingeGroupLabel != "" {
		bingeGroup = "streamnzb-" + bingeGroupLabel
	}
	h := &BehaviorHints{
		NotWebReady: true,
		BingeGroup:  bingeGroup,
	}
	if rel != nil {
		h.Filename = rel.Title
		if rel.Size > 0 {
			h.VideoSize = rel.Size
		}
	}
	if cached != nil {
		h.Cached = cached
	}
	return h
}

func buildStreamsFromPlayList(list *orderedPlayListResult, key StreamSlotKey, streamName, baseURL string, showAll bool) []Stream {
	nameLeft := streamName
	if nameLeft == "" {
		nameLeft = key.StreamID
	}
	var streams []Stream
	if showAll {
		for i, cand := range list.Candidates {
			relTitle := ""
			if cand.Release != nil && cand.Release.Title != "" {
				relTitle = cand.Release.Title
			} else {
				relTitle = fmt.Sprintf("Release %d", i+1)
			}
			isAvail := list.CachedAvailable != nil && cand.Release != nil && cand.Release.DetailsURL != "" && list.CachedAvailable[cand.Release.DetailsURL]
			sName := nameLeft
			if isAvail {
				sName = "⚡ " + nameLeft
			}
			desc := "StreamNZB\n" + relTitle
			playPath := key.SlotPath(i)
			streamURL := baseURL + "/play/" + playPath
			failoverId := "streamnzb-" + playPath
			bingeLabel := bingeGroupLabelFromMeta(cand.Metadata)
			if bingeLabel == "" {
				bingeLabel = nameLeft
			}
			hints := streamBehaviorHints(nameLeft, key.StreamID, cand.Release, &isAvail, bingeLabel)
			streams = append(streams, Stream{
				FailoverID:    failoverId,
				Name:          sName,
				URL:           streamURL,
				Description:   desc,
				BehaviorHints: hints,
			})
		}
	} else {
		branding := "StreamNZB"
		if list.FirstIsAvailGood {
			branding = "StreamNZB [availNZB]"
		}
		var line2 string
		var firstRel *release.Release
		var firstMeta *parser.ParsedRelease
		if len(list.Candidates) > 0 {
			if list.Candidates[0].Release != nil {
				firstRel = list.Candidates[0].Release
				if list.FirstIsAvailGood && firstRel.Title != "" {
					line2 = firstRel.Title
				} else {
					line2 = fmt.Sprintf("%d possible releases", len(list.Candidates))
				}
			} else {
				line2 = fmt.Sprintf("%d possible releases", len(list.Candidates))
			}
			firstMeta = list.Candidates[0].Metadata
		}
		description := branding
		if line2 != "" {
			description = branding + "\n" + line2
		}
		playPath := key.SlotPath(0)
		streamURL := baseURL + "/play/" + playPath
		firstAvail := list.FirstIsAvailGood
		failoverId := "streamnzb-" + playPath
		bingeLabel := bingeGroupLabelFromMeta(firstMeta)
		if bingeLabel == "" {
			bingeLabel = nameLeft
		}
		hints := streamBehaviorHints(nameLeft, key.StreamID, firstRel, &firstAvail, bingeLabel)
		streams = append(streams, Stream{
			FailoverID:    failoverId,
			Name:          nameLeft,
			URL:           streamURL,
			Description:   description,
			BehaviorHints: hints,
		})
		if len(list.Candidates) >= 2 {
			nextPath := key.SlotPath(0)
			nextURL := baseURL + "/next/" + nextPath
			nextName := nameLeft + " (next release)"
			nextDesc := "StreamNZB\nTry next release in list"
			nextFailoverId := "streamnzb-" + nextPath
			nextHints := streamBehaviorHints(nameLeft, key.StreamID, nil, nil, nameLeft)
			streams = append(streams, Stream{
				FailoverID:    nextFailoverId,
				Name:          nextName,
				URL:           nextURL,
				Description:   nextDesc,
				BehaviorHints: nextHints,
			})
		}
	}
	return streams
}

func (s *Server) GetStreams(ctx context.Context, contentType, id string, device *auth.Device) ([]Stream, error) {
	const streamRequestTimeout = 30 * time.Second
	ctx, cancel := context.WithTimeout(ctx, streamRequestTimeout)
	defer cancel()
	var streams []Stream
	streamsList := s.streamConfigsForStreamRequest()
	for _, str := range streamsList {
		key := StreamSlotKey{StreamID: str.ID, ContentType: contentType, ID: id}
		list, err := s.buildOrderedPlayList(ctx, key, false)
		if err != nil || list == nil {
			continue
		}
		if len(list.Candidates) == 0 {
			continue
		}
		s.clearNextReleaseBound(device, key)
		baseURL := s.baseURLWithToken(device)
		streams = append(streams, buildStreamsFromPlayList(list, key, str.Name, baseURL, str.ShowAllStream)...)
	}
	if sink := getStreamSinkFromContext(ctx); sink != nil {
		for _, st := range streams {
			if !sink(st) {
				break
			}
		}
	}
	return streams, nil
}

func addAPIKeyToDownloadURL(downloadURL string, indexers []config.IndexerConfig) string {
	if downloadURL == "" || len(indexers) == 0 {
		return downloadURL
	}
	u, err := url.Parse(downloadURL)
	if err != nil {
		return downloadURL
	}
	q := u.Query()
	if q.Get("t") == "get" && q.Get("id") == "" && q.Get("guid") != "" {
		q.Set("id", q.Get("guid"))
		u.RawQuery = q.Encode()
	}
	downloadHost := strings.ToLower(u.Hostname())
	for _, idx := range indexers {
		idxU, err := url.Parse(idx.URL)
		if err != nil || idx.APIKey == "" {
			continue
		}
		idxHost := strings.ToLower(idxU.Hostname())
		if idxHost == downloadHost ||
			strings.TrimPrefix(idxHost, "api.") == downloadHost ||
			strings.TrimPrefix(downloadHost, "api.") == idxHost {
			q := u.Query()
			q.Set("apikey", idx.APIKey)
			u.RawQuery = q.Encode()
			return u.String()
		}
	}
	return downloadURL
}

func (s *Server) getGlobalStream() *stream.Stream {
	s.mu.RLock()
	sm := s.streamManager
	s.mu.RUnlock()
	if sm == nil {
		return nil
	}
	return sm.GetGlobal()
}

func (s *Server) getDefaultStreamID() string {
	if str := s.getGlobalStream(); str != nil && str.ID != "" {
		return str.ID
	}
	return stream.GlobalStreamID
}

func (s *Server) streamConfigsForStreamRequest() []*stream.Stream {
	s.mu.RLock()
	sm := s.streamManager
	s.mu.RUnlock()
	if sm == nil {
		if g := s.getGlobalStream(); g != nil {
			return []*stream.Stream{g}
		}
		return nil
	}
	list := sm.List()
	if len(list) == 0 {
		if g := s.getGlobalStream(); g != nil {
			return []*stream.Stream{g}
		}
	}

	sort.Slice(list, func(i, j int) bool {
		a, b := list[i], list[j]
		if a == nil || b == nil {
			return a != nil
		}
		ai, bj := a.ID == stream.GlobalStreamID, b.ID == stream.GlobalStreamID
		if ai != bj {
			return ai
		}
		if ai {
			return true
		}
		return a.ID < b.ID
	})
	return list
}

func (s *Server) triageCandidates(str *stream.Stream, releases []*release.Release) []triage.Candidate {
	if str == nil {
		return s.triageService.Filter(releases)
	}
	ts := triage.NewService(&str.Filters, str.Sorting)
	return ts.Filter(releases)
}

type streamSinkKeyType struct{}

var streamSinkKey = streamSinkKeyType{}

const streamSlotPrefix = "stream:"

type StreamSlotKey struct {
	StreamID    string
	ContentType string
	ID          string
}

func (k StreamSlotKey) SlotPath(index int) string {
	return formatStreamSlotPath(k.StreamID, k.ContentType, k.ID, index)
}

func (k StreamSlotKey) CacheKey() string {
	return k.StreamID + ":" + k.ContentType + ":" + k.ID
}

func (k StreamSlotKey) RawCacheKey() string {
	return k.ContentType + ":" + k.ID
}

func (s *Server) baseURLWithToken(device *auth.Device) string {
	base := strings.TrimSuffix(s.baseURL, "/")
	if device != nil && device.Token != "" {
		base += "/" + device.Token
	}
	return base
}

func formatStreamSlotPath(streamID, contentType, id string, index int) string {
	return streamSlotPrefix + streamID + ":" + contentType + ":" + id + ":" + strconv.Itoa(index)
}

type orderedPlayListResult struct {
	Candidates       []triage.Candidate
	FirstIsAvailGood bool
	Params           *SearchParams

	CachedAvailable map[string]bool
}

type rawSearchResult struct {
	Params          *SearchParams
	AvailReleases   []*release.Release
	IndexerReleases []*release.Release
	CachedAvailable map[string]bool
	AvailResult     *availnzb.ReleasesResult
}

const playListCacheTTL = 10 * time.Minute

type playListCacheEntry struct {
	result *orderedPlayListResult
	until  time.Time
}

type rawSearchCacheEntry struct {
	raw   *rawSearchResult
	until time.Time
}

func (s *Server) buildOrderedPlayList(ctx context.Context, key StreamSlotKey, skipFilterSort bool) (*orderedPlayListResult, error) {
	if key.StreamID == "" {
		key.StreamID = s.getDefaultStreamID()
	}
	list, err := s.buildOrderedPlayListUncached(ctx, key, skipFilterSort)
	if err != nil || list == nil {
		return list, err
	}
	cacheKey := key.CacheKey()
	if skipFilterSort {
		cacheKey += "|noFilter"
	}
	s.playListCache.Store(cacheKey, &playListCacheEntry{result: list, until: time.Now().Add(playListCacheTTL)})
	return list, nil
}

func (s *Server) buildOrderedPlayListUncached(ctx context.Context, key StreamSlotKey, skipFilterSort bool) (*orderedPlayListResult, error) {
	raw, err := s.getOrBuildRawSearchResult(ctx, key.ContentType, key.ID)
	if err != nil || raw == nil {
		return nil, err
	}
	streamId := key.StreamID
	if streamId == "" {
		streamId = s.getDefaultStreamID()
	}
	var str *stream.Stream
	if s.streamManager != nil {
		str, _ = s.streamManager.Get(streamId)
	}
	if str == nil {
		str = s.getGlobalStream()
	}
	return s.buildOrderedPlayListFromRaw(raw, str, skipFilterSort)
}

func (s *Server) getOrBuildRawSearchResult(ctx context.Context, contentType, id string) (*rawSearchResult, error) {
	rawKey := contentType + ":" + id
	if v, ok := s.rawSearchCache.Load(rawKey); ok {
		if ent, _ := v.(*rawSearchCacheEntry); ent != nil && time.Now().Before(ent.until) {
			logger.Debug("Raw search cache hit", "key", rawKey)
			return ent.raw, nil
		}
	}
	raw, err := s.buildRawSearchResult(ctx, contentType, id)
	if err != nil || raw == nil {
		return nil, err
	}
	s.rawSearchCache.Store(rawKey, &rawSearchCacheEntry{raw: raw, until: time.Now().Add(playListCacheTTL)})
	return raw, nil
}

func (s *Server) buildRawSearchResult(ctx context.Context, contentType, id string) (*rawSearchResult, error) {
	str := s.getGlobalStream()
	params, err := s.buildSearchParams(contentType, id, str)
	if err != nil {
		return nil, err
	}
	req := &params.Req
	contentIDs := params.ContentIDs
	imdbForText := params.ImdbForText
	tmdbForText := params.TmdbForText

	var availReleases []*release.Release
	cachedAvailable := make(map[string]bool)
	var availResult *availnzb.ReleasesResult
	if s.availClient != nil && s.availClient.BaseURL != "" && (contentIDs.ImdbID != "" || contentIDs.TvdbID != "") {
		availResult, _ = s.availClient.GetReleases(contentIDs.ImdbID, contentIDs.TvdbID, contentIDs.Season, contentIDs.Episode, params.AvailIndexers, s.validator.GetProviderHosts())
	}

	indexerReleases, err := search.RunIndexerSearches(s.indexer, s.tmdbClient, *req, contentType, contentIDs, imdbForText, tmdbForText, s.config)
	if err != nil {
		return nil, err
	}

	if availResult != nil && len(params.AvailIndexers) == 0 {
		indexerDetailsURLs := make(map[string]bool)
		for _, r := range indexerReleases {
			if r != nil && r.DetailsURL != "" {
				indexerDetailsURLs[r.DetailsURL] = true
			}
		}
		if len(indexerDetailsURLs) > 0 {
			filtered := availResult.Releases[:0]
			for _, rws := range availResult.Releases {
				if rws == nil || rws.Release == nil {
					continue
				}
				if !indexerDetailsURLs[rws.Release.DetailsURL] {
					continue
				}
				filtered = append(filtered, rws)
			}
			availResult = &availnzb.ReleasesResult{ImdbID: availResult.ImdbID, Count: availResult.Count, Releases: filtered}
		}
	}

	if availResult != nil {
		for _, rws := range availResult.Releases {
			if rws == nil || rws.Release == nil || !rws.Available || rws.Release.Link == "" {
				continue
			}

			rws.Release.QuerySource = "id"
			availReleases = append(availReleases, rws.Release)
			cachedAvailable[rws.Release.DetailsURL] = true
		}
	}

	return &rawSearchResult{
		Params:          params,
		AvailReleases:   availReleases,
		IndexerReleases: indexerReleases,
		CachedAvailable: cachedAvailable,
		AvailResult:     availResult,
	}, nil
}

func (s *Server) GetSearchReleases(ctx context.Context, contentType, id string) (*SearchReleasesResponse, error) {
	raw, err := s.getOrBuildRawSearchResult(ctx, contentType, id)
	if err != nil || raw == nil {
		return nil, err
	}
	populateAvailable(raw)

	type releaseWithAvail struct {
		rel   *release.Release
		avail string
	}
	seenKey := make(map[string]bool)
	var unified []releaseWithAvail

	addRelease := func(r *release.Release) bool {
		key := release.Key(r)
		if key == "" || seenKey[key] {
			return true
		}
		seenKey[key] = true
		return false
	}

	setAvail := func(rel *release.Release, avail string) {
		if rel == nil {
			return
		}
		switch avail {
		case "Available":
			rel.Available = &availTrue
		case "Unavailable":
			rel.Available = &availFalse
		case "Unknown":
			if len(raw.CachedAvailable) > 0 && rel.DetailsURL != "" && raw.CachedAvailable[rel.DetailsURL] {
				rel.Available = &availTrue
			}

		}
	}

	if raw.AvailResult != nil {
		for _, rws := range raw.AvailResult.Releases {
			if rws == nil || rws.Release == nil {
				continue
			}
			r := rws.Release
			avail := "Unavailable"
			if rws.Available {
				avail = "Available"
			}
			if addRelease(r) {
				continue
			}
			setAvail(r, avail)
			unified = append(unified, releaseWithAvail{rel: r, avail: avail})
		}
	}
	for _, r := range raw.IndexerReleases {
		if r == nil {
			continue
		}
		if addRelease(r) {
			continue
		}
		setAvail(r, "Unknown")
		unified = append(unified, releaseWithAvail{rel: r, avail: "Unknown"})
	}

	streams := s.streamConfigsForStreamRequest()
	releaseScores := make(map[string]map[string]struct {
		Fits  bool
		Score int
	})
	for _, str := range streams {
		if str == nil {
			continue
		}
		releasesOnly := make([]*release.Release, 0, len(unified))
		for _, u := range unified {
			releasesOnly = append(releasesOnly, u.rel)
		}
		candidates := s.triageCandidates(str, releasesOnly)
		for _, c := range candidates {
			if c.Release == nil {
				continue
			}
			key := release.Key(c.Release)
			if releaseScores[key] == nil {
				releaseScores[key] = make(map[string]struct {
					Fits  bool
					Score int
				})
			}
			releaseScores[key][str.ID] = struct {
				Fits  bool
				Score int
			}{Fits: true, Score: c.Score}
		}

		for _, u := range unified {
			key := release.Key(u.rel)
			if releaseScores[key] == nil {
				releaseScores[key] = make(map[string]struct {
					Fits  bool
					Score int
				})
			}
			if _, ok := releaseScores[key][str.ID]; !ok {
				releaseScores[key][str.ID] = struct {
					Fits  bool
					Score int
				}{Fits: false, Score: 0}
			}
		}
	}

	streamInfos := make([]SearchStreamInfo, 0, len(streams))
	for _, str := range streams {
		if str != nil {
			streamInfos = append(streamInfos, SearchStreamInfo{ID: str.ID, Name: str.Name})
		}
	}

	releasesOut := make([]SearchReleaseTag, 0, len(unified))
	for _, u := range unified {
		r := u.rel
		key := release.Key(r)
		tags := make([]SearchStreamTag, 0, len(streams))
		for _, str := range streams {
			if str == nil {
				continue
			}
			ts := releaseScores[key][str.ID]
			tags = append(tags, SearchStreamTag{
				StreamID:   str.ID,
				StreamName: str.Name,
				Fits:       ts.Fits,
				Score:      ts.Score,
			})
		}
		idxName := r.Indexer
		if idxName == "" && r.SourceIndexer != nil {
			if idx, ok := r.SourceIndexer.(indexer.Indexer); ok {
				idxName = idx.Name()
			}
		}
		if idxName == "" {
			idxName = "Indexer"
		}
		releasesOut = append(releasesOut, SearchReleaseTag{
			Title:        r.Title,
			Link:         r.Link,
			DetailsURL:   r.DetailsURL,
			Size:         r.Size,
			Indexer:      idxName,
			Availability: u.avail,
			StreamTags:   tags,
		})
	}

	if len(streamInfos) > 0 && len(releasesOut) > 0 {
		firstStreamID := streamInfos[0].ID
		sort.Slice(releasesOut, func(i, j int) bool {
			si := 0
			sj := 0
			for _, t := range releasesOut[i].StreamTags {
				if t.StreamID == firstStreamID {
					si = t.Score
					break
				}
			}
			for _, t := range releasesOut[j].StreamTags {
				if t.StreamID == firstStreamID {
					sj = t.Score
					break
				}
			}
			if si != sj {
				return si > sj
			}

			availOrder := map[string]int{"Available": 2, "Unknown": 1, "Unavailable": 0}
			return (availOrder[releasesOut[i].Availability] - availOrder[releasesOut[j].Availability]) > 0
		})
	}

	return &SearchReleasesResponse{Streams: streamInfos, Releases: releasesOut}, nil
}

func populateAvailable(raw *rawSearchResult) {
	if raw.AvailResult != nil {
		for _, rws := range raw.AvailResult.Releases {
			if rws == nil || rws.Release == nil {
				continue
			}
			if rws.Available {
				rws.Release.Available = &availTrue
			} else {
				rws.Release.Available = &availFalse
			}
		}
	}
	if len(raw.CachedAvailable) > 0 {
		for _, rel := range raw.IndexerReleases {
			if rel != nil && rel.DetailsURL != "" && raw.CachedAvailable[rel.DetailsURL] {
				rel.Available = &availTrue
			}
		}
	}
}

func buildAllReleasesFromRaw(raw *rawSearchResult) []*release.Release {
	seenURL := make(map[string]bool)
	var out []*release.Release
	for _, rel := range raw.AvailReleases {
		if rel == nil || rel.DetailsURL == "" {
			continue
		}
		if seenURL[rel.DetailsURL] {
			continue
		}
		seenURL[rel.DetailsURL] = true
		out = append(out, rel)
	}
	for _, rel := range raw.IndexerReleases {
		if rel == nil || rel.DetailsURL == "" {
			continue
		}
		if seenURL[rel.DetailsURL] {
			continue
		}
		seenURL[rel.DetailsURL] = true
		out = append(out, rel)
	}
	return out
}

func releasesToCandidates(releases []*release.Release) []triage.Candidate {
	var out []triage.Candidate
	for _, rel := range releases {
		if rel == nil {
			continue
		}
		out = append(out, triage.Candidate{Release: rel, Score: 0})
	}
	return out
}

func (s *Server) buildOrderedPlayListFromRaw(raw *rawSearchResult, str *stream.Stream, skipFilterSort bool) (*orderedPlayListResult, error) {
	populateAvailable(raw)

	unavailableDetailsURLs := make(map[string]bool)
	if raw.AvailResult != nil {
		for _, rws := range raw.AvailResult.Releases {
			if rws == nil || rws.Release == nil || rws.Available {
				continue
			}
			if rws.Release.DetailsURL != "" {
				unavailableDetailsURLs[rws.Release.DetailsURL] = true
			}
		}
	}

	allReleases := buildAllReleasesFromRaw(raw)
	var merged []triage.Candidate
	if skipFilterSort {
		merged = releasesToCandidates(allReleases)
	} else {
		merged = s.triageCandidates(str, allReleases)
	}

	seenURL := make(map[string]bool)
	var seenTitle map[string]bool
	if skipFilterSort {
		seenTitle = make(map[string]bool)
	}
	filtered := merged[:0]
	for _, c := range merged {
		if c.Release == nil || c.Release.DetailsURL == "" {
			continue
		}
		if unavailableDetailsURLs[c.Release.DetailsURL] {
			continue
		}
		if seenURL[c.Release.DetailsURL] {
			continue
		}
		if seenTitle != nil && c.Release.Title != "" {
			titleKey := release.NormalizeTitleForDedup(c.Release.Title)
			if titleKey != "" {
				if seenTitle[titleKey] {
					continue
				}
				seenTitle[titleKey] = true
			}
		}
		seenURL[c.Release.DetailsURL] = true
		filtered = append(filtered, c)
	}
	merged = filtered

	{
		const recentFailureTTL = 5 * time.Minute
		now := time.Now()
		filtered := merged[:0]
		for _, c := range merged {
			if c.Release == nil || c.Release.Title == "" {
				continue
			}
			key := release.NormalizeTitle(c.Release.Title)
			if v, ok := s.recentFailures.Load(key); ok {
				if failedAt, ok := v.(time.Time); ok && now.Sub(failedAt) < recentFailureTTL {
					continue
				}
				s.recentFailures.Delete(key)
			}
			filtered = append(filtered, c)
		}
		merged = filtered
	}

	if raw.AvailResult != nil && s.availClient != nil {
		ourBackbones, _ := s.availClient.OurBackbones(s.validator.GetProviderHosts())
		cachedUnhealthyForUs := make(map[string]bool)
		for _, rws := range raw.AvailResult.Releases {
			if rws == nil || rws.Release == nil {
				continue
			}
			if rws.Available {
				continue
			}
			if len(ourBackbones) > 0 && len(rws.Summary) > 0 {
				ourReported, ourHealthy := 0, 0
				for backbone, status := range rws.Summary {
					if ourBackbones[backbone] {
						ourReported++
						if status.Healthy {
							ourHealthy++
						}
					}
				}
				if ourReported > 0 && ourHealthy == 0 {
					cachedUnhealthyForUs[rws.Release.DetailsURL] = true
				}
			}
		}
		if len(cachedUnhealthyForUs) > 0 {
			filtered := merged[:0]
			for _, c := range merged {
				if c.Release == nil || !cachedUnhealthyForUs[c.Release.DetailsURL] {
					filtered = append(filtered, c)
				}
			}
			merged = filtered
		}
	}
	if !skipFilterSort {

		sort.Slice(merged, func(i, j int) bool {
			return streamScoreFromCandidate(merged[i]) > streamScoreFromCandidate(merged[j])
		})
	}

	firstIsAvailGood := false
	if len(merged) > 0 && merged[0].Release != nil && merged[0].Release.DetailsURL != "" {
		firstIsAvailGood = raw.CachedAvailable[merged[0].Release.DetailsURL]
	}
	return &orderedPlayListResult{
		Candidates:       merged,
		FirstIsAvailGood: firstIsAvailGood,
		Params:           raw.Params,
		CachedAvailable:  raw.CachedAvailable,
	}, nil
}

func streamScoreFromCandidate(c triage.Candidate) int {
	return c.Score
}

type StreamSink func(Stream) bool

func WithStreamSink(ctx context.Context, sink StreamSink) context.Context {
	if sink == nil {
		return ctx
	}
	return context.WithValue(ctx, streamSinkKey, sink)
}

func getStreamSinkFromContext(ctx context.Context) StreamSink {
	if v := ctx.Value(streamSinkKey); v != nil {
		if f, ok := v.(StreamSink); ok {
			return f
		}
	}
	return nil
}

type SearchParams struct {
	ContentType   string
	ID            string
	Req           indexer.SearchRequest
	ContentIDs    *session.AvailReportMeta
	ImdbForText   string
	TmdbForText   string
	AvailIndexers []string
}

func (s *Server) buildSearchParams(contentType, id string, str *stream.Stream) (*SearchParams, error) {
	const searchLimit = 1000
	params := &SearchParams{ContentType: contentType, ID: id}
	req := indexer.SearchRequest{Limit: searchLimit}

	searchID := id
	if contentType == "series" && strings.Contains(id, ":") {
		parts := strings.Split(id, ":")
		if parts[0] == "tmdb" && len(parts) >= 4 {
			searchID = parts[1]
			req.Season, req.Episode = parts[2], parts[3]
		} else if len(parts) >= 3 {
			searchID = parts[0]
			req.Season, req.Episode = parts[1], parts[2]
		} else if len(parts) > 0 {
			searchID = parts[0]
		}
	} else if strings.HasPrefix(id, "tmdb:") {
		searchID = strings.TrimPrefix(id, "tmdb:")
	}
	if strings.HasPrefix(searchID, "tt") {
		req.IMDbID = searchID
	} else {
		req.TMDBID = searchID
	}
	imdbForText := req.IMDbID
	tmdbForText := req.TMDBID
	if contentType == "series" && strings.Contains(id, ":") {
		parts := strings.Split(id, ":")
		if parts[0] == "tmdb" && len(parts) >= 2 {
			tmdbForText = parts[1]
		}
	}
	if contentType == "movie" {
		req.Cat = "2000"
	} else {
		req.Cat = "5000"
	}

	if contentType == "series" {
		if req.IMDbID != "" && req.TVDBID == "" {
			if s.tvdbClient != nil {
				if tvdbID, err := s.tvdbClient.ResolveTVDBID(req.IMDbID); err == nil && tvdbID != "" {
					req.TVDBID = tvdbID
				}
			}
			if req.TVDBID == "" && s.tmdbClient != nil {
				if tvdbID, err := s.tmdbClient.ResolveTVDBID(req.IMDbID); err == nil && tvdbID != "" {
					req.TVDBID = tvdbID
				}
			}
		}
		if req.TVDBID == "" && req.TMDBID != "" && s.tmdbClient != nil {
			if tmdbIDNum, err := strconv.Atoi(req.TMDBID); err == nil {
				if extIDs, err := s.tmdbClient.GetExternalIDs(tmdbIDNum, "tv"); err == nil {
					if extIDs.TVDBID != 0 {
						req.TVDBID = strconv.Itoa(extIDs.TVDBID)
					}
					if extIDs.IMDbID != "" && req.IMDbID == "" {
						req.IMDbID = extIDs.IMDbID
						imdbForText = extIDs.IMDbID
					}
				}
			}
		}
	}
	seasonNum, _ := strconv.Atoi(req.Season)
	episodeNum, _ := strconv.Atoi(req.Episode)
	contentIDs := &session.AvailReportMeta{ImdbID: req.IMDbID, TvdbID: req.TVDBID, Season: seasonNum, Episode: episodeNum}
	if contentType == "movie" && contentIDs.ImdbID == "" && req.TMDBID != "" && s.tmdbClient != nil {
		if tmdbIDNum, err := strconv.Atoi(req.TMDBID); err == nil {
			if extIDs, err := s.tmdbClient.GetExternalIDs(tmdbIDNum, "movie"); err == nil && extIDs.IMDbID != "" {
				contentIDs.ImdbID = extIDs.IMDbID
				req.IMDbID = contentIDs.ImdbID
				imdbForText = contentIDs.ImdbID
			}
		}
	}
	if len(s.config.Indexers) > 0 {
		req.EffectiveByIndexer = make(map[string]*config.IndexerSearchConfig)
		for i := range s.config.Indexers {
			ic := &s.config.Indexers[i]
			var override *config.IndexerSearchConfig
			if str != nil && str.IndexerOverrides != nil {
				if o, ok := str.IndexerOverrides[ic.Name]; ok {
					override = &o
				}
				if override == nil {
					if o, ok := str.IndexerOverrides[""]; ok {
						override = &o
					}
				}
			}
			req.EffectiveByIndexer[ic.Name] = config.MergeIndexerSearch(ic, override, s.config)
		}
		req.PerIndexerQuery = make(map[string][]string)
		if s.tmdbClient != nil {
			if contentType == "movie" {

				type queryKey struct {
					lang        string
					includeYear bool
					norm        bool
				}
				resolved := make(map[queryKey][]string)
				for name, eff := range req.EffectiveByIndexer {
					includeYear := eff.IncludeYearInSearch != nil && *eff.IncludeYearInSearch
					lang := ""
					if eff.SearchTitleLanguage != nil {
						lang = *eff.SearchTitleLanguage
					}
					norm := eff.SearchTitleNormalize != nil && *eff.SearchTitleNormalize
					k := queryKey{lang: lang, includeYear: includeYear, norm: norm}
					if queries, ok := resolved[k]; ok {
						req.PerIndexerQuery[name] = queries
						continue
					}
					primary, orig, err := s.tmdbClient.GetMovieTitlesForSearch(contentIDs.ImdbID, req.TMDBID, lang, includeYear, norm)
					if err != nil {
						logger.Debug("Per-indexer movie query failed", "indexer", name, "language", lang, "err", err)
						continue
					}
					queries := []string{primary}
					if orig != "" {
						queries = append(queries, orig)
						logger.Debug("Per-indexer movie query", "indexer", name, "language", lang, "primary", primary, "original", orig)
					} else {
						logger.Debug("Per-indexer movie query", "indexer", name, "language", lang, "query", primary)
					}
					resolved[k] = queries
					req.PerIndexerQuery[name] = queries
				}
			} else if req.Season != "" && req.Episode != "" {
				showName, err := s.tmdbClient.GetTVShowName(tmdbForText, imdbForText)
				if err == nil {
					seasonNum, _ := strconv.Atoi(req.Season)
					epNum, _ := strconv.Atoi(req.Episode)
					var q string
					if seasonNum > 0 || epNum > 0 {
						q = fmt.Sprintf("%s S%02dE%02d", showName, seasonNum, epNum)
					} else {
						q = fmt.Sprintf("%s S%sE%s", showName, req.Season, req.Episode)
					}
					for name := range req.EffectiveByIndexer {
						req.PerIndexerQuery[name] = []string{q}
					}
				}
			}
		}
	}
	params.Req = req
	params.ContentIDs = contentIDs
	params.ImdbForText = imdbForText
	params.TmdbForText = tmdbForText
	params.AvailIndexers = s.availNZBIndexerHosts
	return params, nil
}

func (s *Server) runAvailNZBPhase(ctx context.Context, params *SearchParams, str *stream.Stream, device *auth.Device) ([]Stream, []*release.Release, *availnzb.ReleasesResult) {
	contentIDs := params.ContentIDs
	availIndexers := params.AvailIndexers
	if s.availClient == nil || s.availClient.BaseURL == "" || (contentIDs.ImdbID == "" && contentIDs.TvdbID == "") {
		return nil, nil, nil
	}
	availResult, _ := s.availClient.GetReleases(contentIDs.ImdbID, contentIDs.TvdbID, contentIDs.Season, contentIDs.Episode, availIndexers, s.validator.GetProviderHosts())
	if availResult == nil || len(availResult.Releases) == 0 {
		return nil, nil, nil
	}
	var availReleases []*release.Release
	for _, rws := range availResult.Releases {
		if rws == nil || rws.Release == nil || !rws.Available || rws.Release.Link == "" {
			continue
		}
		availReleases = append(availReleases, rws.Release)
	}
	if len(availReleases) == 0 {
		return nil, nil, availResult
	}
	candidates := s.triageCandidates(str, availReleases)
	logger.Debug("AvailNZB phase", "releases", len(availReleases), "after_triage", len(candidates))
	var streams []Stream
	seen := make(map[string]bool)
	for _, cand := range candidates {
		if cand.Release == nil {
			continue
		}
		rel := cand.Release
		norm := release.NormalizeTitle(rel.Title)
		if seen[norm] {
			continue
		}
		seen[norm] = true
		downloadURL := addAPIKeyToDownloadURL(rel.Link, s.config.Indexers)
		sessionID := fmt.Sprintf("%x", md5.Sum([]byte(rel.DetailsURL)))
		_, err := s.sessionManager.CreateDeferredSession(sessionID, downloadURL, rel, s.indexer, contentIDs)
		if err != nil {
			logger.Debug("AvailNZB deferred session failed", "title", rel.Title, "err", err)
			continue
		}
		var streamURL string
		if device != nil {
			streamURL = fmt.Sprintf("%s/%s/play/%s", s.baseURL, device.Token, sessionID)
		}
		sizeGB := float64(rel.Size) / (1024 * 1024 * 1024)
		displayTitle := rel.Title + "\n[AvailNZB]"
		stream := buildStreamMetadata(streamURL, displayTitle, cand, sizeGB, rel.Size, rel)
		streams = append(streams, stream)
	}
	sort.Slice(streams, func(i, j int) bool {
		return streamScore(streams[i]) > streamScore(streams[j])
	})
	return streams, availReleases, availResult
}

func (s *Server) GetAvailNZBStreams(ctx context.Context, contentType, id string, device *auth.Device) ([]Stream, error) {
	str := s.getGlobalStream()
	params, err := s.buildSearchParams(contentType, id, str)
	if err != nil {
		return nil, err
	}
	streams, _, _ := s.runAvailNZBPhase(ctx, params, str, device)
	if streams == nil {
		return []Stream{}, nil
	}
	return streams, nil
}

func (s *Server) resolveStreamSlot(ctx context.Context, key StreamSlotKey, index int, device *auth.Device, skipFilterSort bool) (*session.Session, error) {
	if key.StreamID == "" {
		key.StreamID = s.getDefaultStreamID()
	}
	list, err := s.buildOrderedPlayList(ctx, key, skipFilterSort)
	if err != nil || list == nil {
		return nil, fmt.Errorf("build play list: %w", err)
	}
	if index < 0 {
		return nil, fmt.Errorf("index %d out of range (candidates %d)", index, len(list.Candidates))
	}
	if index >= len(list.Candidates) {
		if len(list.Candidates) == 0 {
			return nil, fmt.Errorf("index %d out of range (candidates 0)", index)
		}
		requested := index
		index = len(list.Candidates) - 1
		logger.Debug("Stream slot index out of range, using last candidate", "requested", requested, "candidates", len(list.Candidates))
	}
	cand := list.Candidates[index]
	rel := cand.Release
	if rel == nil || rel.Link == "" {
		return nil, fmt.Errorf("no release at index %d", index)
	}
	downloadURL := addAPIKeyToDownloadURL(rel.Link, s.config.Indexers)
	idx := s.indexer
	if rel.SourceIndexer != nil {
		if i, ok := rel.SourceIndexer.(indexer.Indexer); ok {
			idx = i
		}
	}
	sessionID := key.SlotPath(index)
	_, err = s.sessionManager.CreateDeferredSession(sessionID, downloadURL, rel, idx, list.Params.ContentIDs)
	if err != nil {
		return nil, fmt.Errorf("create deferred session: %w", err)
	}
	base := s.baseURLWithToken(device)
	token := ""
	if device != nil {
		token = device.Token
	}
	ourSlotPath := sessionID
	order := s.sessionManager.GetDeviceFailoverOrder(token)
	var fallbackURLs []string
	var ourPosition int = -1
	var matchedByPath bool
	if len(order) > 0 {
		for p, entry := range order {
			if entry == ourSlotPath {
				ourPosition = p
				matchedByPath = true
				break
			}
			if !strings.HasPrefix(entry, streamSlotPrefix) && entry == key.StreamID {
				if legacyOccurrenceIndex(order, entry, p) == index {
					ourPosition = p
					break
				}
			}
		}
	}
	if ourPosition >= 0 && !matchedByPath {
		logger.Debug("Failover order: matched by legacy (stream ID + occurrence); send playPath as BingeGroup for exact order", "stream", key.StreamID, "index", index)
	}

	maxIndex := len(list.Candidates) - 1
	if ourPosition >= 0 {
		for j := ourPosition + 1; j < len(order); j++ {
			entry := order[j]
			if strings.HasPrefix(entry, streamSlotPrefix) {
				if _, _, _, orderIndex, ok := parseStreamSlotID(entry); ok && orderIndex <= maxIndex {
					fallbackURLs = append(fallbackURLs, base+"/play/"+entry)
				}
			} else {
				ci := legacyOccurrenceIndex(order, entry, j)
				if ci <= maxIndex {
					fallbackURLs = append(fallbackURLs, base+"/play/"+formatStreamSlotPath(entry, key.ContentType, key.ID, ci))
				}
			}
		}
	} else {
		for i := index + 1; i < len(list.Candidates); i++ {
			fallbackURLs = append(fallbackURLs, base+"/play/"+key.SlotPath(i))
		}
	}
	s.sessionManager.SetFallbackStreams(sessionID, fallbackURLs)
	sess, err := s.sessionManager.GetSession(sessionID)
	if err != nil {
		return nil, err
	}
	return sess, nil
}

type nextReleaseState struct {
	mu         sync.Mutex
	NextIndex  int
	BoundIndex int
}

func (s *Server) isSlotFailed(key StreamSlotKey, slotIndex int) bool {
	slotSessionID := key.SlotPath(slotIndex)
	sess, err := s.sessionManager.GetSession(slotSessionID)
	if err != nil || sess == nil {
		return false
	}
	files := sess.Files
	if len(files) == 0 && sess.File != nil {
		files = []*loader.File{sess.File}
	}
	for _, f := range files {
		if f != nil && f.IsFailed() {
			return true
		}
	}
	return false
}

func (s *Server) handleNextRelease(w http.ResponseWriter, r *http.Request, device *auth.Device) {
	sessionID := strings.TrimPrefix(r.URL.Path, "/next/")
	if sessionID == "" {
		http.Error(w, "Missing stream slot", http.StatusBadRequest)
		return
	}
	streamId, contentType, id, index, ok := parseStreamSlotID(sessionID)
	if !ok {
		http.Error(w, "Invalid stream slot", http.StatusBadRequest)
		return
	}
	if streamId == "" {
		streamId = s.getDefaultStreamID()
	}
	slotKey := StreamSlotKey{StreamID: streamId, ContentType: contentType, ID: id}

	var playIndex int
	if index == 0 {
		stateKey := ":" + slotKey.CacheKey()
		if device != nil && device.Token != "" {
			stateKey = device.Token + ":" + slotKey.CacheKey()
		}
		isAIOStreams := strings.Contains(r.Header.Get("User-Agent"), "AIOStreams")
		list, err := s.buildOrderedPlayList(r.Context(), slotKey, isAIOStreams)
		maxIdx := 0
		if err == nil && list != nil && len(list.Candidates) > 1 {
			maxIdx = len(list.Candidates) - 1
		}
		v, _ := s.nextReleaseIndex.LoadOrStore(stateKey, &nextReleaseState{NextIndex: 1, BoundIndex: -1})
		state := v.(*nextReleaseState)
		state.mu.Lock()
		if state.BoundIndex >= 0 {
			playIndex = state.BoundIndex
			if playIndex > maxIdx {
				playIndex = maxIdx
			}
			state.mu.Unlock()
		} else {
			playIndex = state.NextIndex
			if playIndex > maxIdx {
				playIndex = maxIdx
			}
			for playIndex <= maxIdx && s.isSlotFailed(slotKey, playIndex) {
				playIndex++
			}
			if playIndex > maxIdx {
				playIndex = maxIdx
			}
			state.NextIndex = playIndex + 1
			state.BoundIndex = playIndex
			state.mu.Unlock()
		}
	} else {
		playIndex = index + 1
	}

	nextSlot := slotKey.SlotPath(playIndex)
	nextURL := s.baseURLWithToken(device) + "/play/" + nextSlot + "?next=1"
	logger.Info("Next release redirect", "from", sessionID, "to", nextSlot)
	w.Header().Set("Location", nextURL)
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	http.Redirect(w, r, nextURL, http.StatusTemporaryRedirect)
}

func (s *Server) clearNextReleaseBound(device *auth.Device, key StreamSlotKey) {
	if key.StreamID == "" || key.ContentType == "" || key.ID == "" {
		return
	}
	stateKey := ":" + key.CacheKey()
	if device != nil && device.Token != "" {
		stateKey = device.Token + ":" + key.CacheKey()
	}
	if v, ok := s.nextReleaseIndex.Load(stateKey); ok {
		if state, _ := v.(*nextReleaseState); state != nil {
			state.mu.Lock()
			state.BoundIndex = -1
			state.mu.Unlock()
		}
	}
}

func parseStreamSlotID(sessionID string) (streamId, contentType, id string, index int, ok bool) {
	if !strings.HasPrefix(sessionID, streamSlotPrefix) {
		return "", "", "", 0, false
	}
	rest := strings.TrimPrefix(sessionID, streamSlotPrefix)
	parts := strings.Split(rest, ":")
	if len(parts) < 3 {
		return "", "", "", 0, false
	}
	index, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return "", "", "", 0, false
	}
	if len(parts) == 3 {
		contentType = parts[0]
		id = parts[1]
		return "", contentType, id, index, true
	}
	streamId = parts[0]
	contentType = parts[1]
	id = strings.Join(parts[2:len(parts)-1], ":")
	return streamId, contentType, id, index, true
}

func legacyOccurrenceIndex(order []string, entry string, pos int) int {
	count := 0
	for k := 0; k <= pos; k++ {
		if order[k] == entry {
			count++
		}
	}
	return count - 1
}

func (s *Server) handlePlay(w http.ResponseWriter, r *http.Request, device *auth.Device) {
	sessionID := strings.TrimPrefix(r.URL.Path, "/play/")
	logger.Info("Play request", "session", sessionID)

	requestedSessionID := sessionID

	chainStart := r.URL.Query().Get("from")
	if chainStart == "" {
		chainStart = sessionID
	}

	skipKnownGood := r.URL.Query().Get("next") != ""

	if !skipKnownGood {
		for {
			if knownGood, found := s.sessionManager.GetKnownGoodSlot(sessionID); found && knownGood != sessionID {
				logger.Info("Serving known-good stream directly (player reused original URL)", "requested", sessionID, "serving", knownGood)
				sessionID = knownGood
			} else {
				break
			}
		}
	}

	var (
		sess   *session.Session
		stream io.ReadSeekCloser
		name   string
		size   int64
	)

	var err error
	sess, err = s.sessionManager.GetSession(sessionID)
	if err != nil {
		if streamId, contentType, id, index, ok := parseStreamSlotID(sessionID); ok {
			skipFilterSort := strings.Contains(r.Header.Get("User-Agent"), "AIOStreams")
			key := StreamSlotKey{StreamID: streamId, ContentType: contentType, ID: id}
			sess, err = s.resolveStreamSlot(r.Context(), key, index, device, skipFilterSort)
			if err != nil {
				logger.Debug("Resolve stream slot failed", "slot", sessionID, "err", err)
				http.Error(w, "Stream slot not found or invalid", http.StatusNotFound)
				return
			}
		} else {
			http.Error(w, "Session expired or not found", http.StatusNotFound)
			return
		}
	}

	skipFilterSort := strings.Contains(r.Header.Get("User-Agent"), "AIOStreams")
	var triedSlotIDs []string
	for {
		triedSlotIDs = append(triedSlotIDs, sessionID)
		s.prefetchNextFallbackNZB(nextFallbackSessionID(sess), device, skipFilterSort)
		stream, name, size, err = s.tryPlaySlot(r.Context(), sess)
		if err == nil {
			break
		}
		s.sessionManager.ClearKnownGoodForSlot(sessionID)
		s.sessionManager.DeleteSession(sessionID)
		nextID := nextFallbackSessionID(sess)
		if nextID == "" {
			logger.Info("No more fallback slots", "last", sessionID, "err", err)
			forceDisconnect(w, s.baseURL)
			return
		}
		logger.Info("Trying next fallback slot (internal)", "from", sessionID, "to", nextID, "err", err)
		sess, err = s.getOrResolveSession(r.Context(), nextID, device, skipFilterSort)
		if err != nil {
			logger.Debug("Resolve next fallback failed", "next", nextID, "err", err)
			forceDisconnect(w, s.baseURL)
			return
		}
		sessionID = nextID
	}
	defer stream.Close()

	if s.availReporter != nil {
		s.availReporter.ReportGood(sess)
	}

	winningSessionID := sessionID
	for _, id := range triedSlotIDs {
		s.sessionManager.SetKnownGoodSlot(id, winningSessionID)
	}
	if chainStart != "" && chainStart != winningSessionID {
		s.sessionManager.SetKnownGoodSlot(chainStart, winningSessionID)
	}
	if requestedSessionID != winningSessionID {
		s.sessionManager.SetKnownGoodSlot(requestedSessionID, winningSessionID)
	}

	if tStr := r.URL.Query().Get("t"); tStr != "" && r.Header.Get("Range") == "" {
		if tSec, parseOK := seek.ParseTSeconds(tStr); parseOK {
			if byteOffset, seekOK := seek.TimeToByteOffset(stream, size, name, tSec); seekOK {
				r.Header.Set("Range", "bytes="+strconv.FormatInt(byteOffset, 10)+"-")
			}
		}
	}

	clientIP, _, _ := net.SplitHostPort(r.RemoteAddr)
	if clientIP == "" {
		clientIP = r.RemoteAddr
	}
	s.sessionManager.StartPlayback(sessionID, clientIP)
	defer s.sessionManager.EndPlayback(sessionID, clientIP)

	monitoredStream := &StreamMonitor{
		ReadSeekCloser: stream,
		sessionID:      sessionID,
		clientIP:       clientIP,
		manager:        s.sessionManager,
		lastUpdate:     time.Now(),
	}

	logger.Info("Serving media", "name", name, "size", size, "session", sessionID)

	ext := strings.ToLower(filepath.Ext(name))
	if ext == ".mkv" {
		w.Header().Set("Content-Type", "video/x-matroska")
	} else if ext == ".avi" {
		w.Header().Set("Content-Type", "video/x-msvideo")
	} else if ext == ".mp4" || ext == ".m4v" {
		w.Header().Set("Content-Type", "video/mp4")
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filepath.Base(name)))
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w = newWriteTimeoutResponseWriter(w, 10*time.Minute)
	bufW := newBufferedResponseWriter(w, 256*1024)
	defer bufW.Flush()

	http.ServeContent(bufW, r, name, time.Time{}, monitoredStream)
	logger.Trace("Finished serving media", "session", sessionID)
}

func (s *Server) tryPlaySlot(ctx context.Context, sess *session.Session) (io.ReadSeekCloser, string, int64, error) {
	sessionID := sess.ID
	if _, err := sess.GetOrDownloadNZB(s.sessionManager); err != nil {
		logger.Error("Failed to lazy load NZB", "id", sessionID, "err", err)
		return nil, "", 0, err
	}

	files := sess.Files
	if len(files) == 0 {
		if sess.File != nil {
			files = []*loader.File{sess.File}
		} else {
			logger.Error("No files in session", "id", sessionID)
			if sess.NZB != nil {
				s.validator.InvalidateCache(sess.NZB.Hash())
			}
			return nil, "", 0, fmt.Errorf("no files in session %s", sessionID)
		}
	}

	for _, f := range files {
		if f.IsFailed() {
			logger.Error("Session file has too many failures", "session", sessionID, "file", f.Name())
			s.reportBadRelease(sess, unpack.ErrTooManyZeroFills)
			if sess.NZB != nil {
				s.validator.InvalidateCache(sess.NZB.Hash())
			}
			return nil, "", 0, fmt.Errorf("file %s exceeded failure threshold", f.Name())
		}
	}

	password := ""
	if sess.NZB != nil {
		password = sess.NZB.Password()
	}
	unpackFiles := make([]unpack.UnpackableFile, len(files))
	for i := range files {
		unpackFiles[i] = files[i]
	}
	stream, name, size, bp, err := unpack.GetMediaStream(ctx, unpackFiles, sess.Blueprint, password)
	if bp != nil && sess.Blueprint == nil {
		sess.SetBlueprint(bp)
	}
	if err != nil {
		logger.Error("Failed to open media stream", "id", sessionID, "err", err)
		s.reportBadRelease(sess, err)
		if sess.NZB != nil {
			s.validator.InvalidateCache(sess.NZB.Hash())
		}
		return nil, "", 0, err
	}
	return stream, name, size, nil
}

func nextFallbackSessionID(sess *session.Session) string {
	nextURL := sess.FirstFallbackStreamURL()
	if nextURL == "" {
		return ""
	}
	idx := strings.Index(nextURL, "/play/")
	if idx < 0 {
		return ""
	}
	return nextURL[idx+len("/play/"):]
}

// prefetchNextFallbackNZB starts a background goroutine to resolve the next fallback slot and
// download its NZB so that when we fail over to it, tryPlaySlot may find the NZB already loaded.
func (s *Server) prefetchNextFallbackNZB(nextSlotID string, device *auth.Device, skipFilterSort bool) {
	if nextSlotID == "" {
		return
	}
	go func(id string, dev *auth.Device, skip bool) {
		ctx := context.Background()
		sess, err := s.getOrResolveSession(ctx, id, dev, skip)
		if err != nil {
			return
		}
		if _, err := sess.GetOrDownloadNZB(s.sessionManager); err != nil {
			logger.Trace("Prefetch NZB failed (next fallback)", "slot", id, "err", err)
		}
	}(nextSlotID, device, skipFilterSort)
}

func (s *Server) getOrResolveSession(ctx context.Context, sessionID string, device *auth.Device, skipFilterSort bool) (*session.Session, error) {
	sess, err := s.sessionManager.GetSession(sessionID)
	if err == nil {
		return sess, nil
	}
	if streamId, contentType, id, index, ok := parseStreamSlotID(sessionID); ok {
		key := StreamSlotKey{StreamID: streamId, ContentType: contentType, ID: id}
		sess, err = s.resolveStreamSlot(ctx, key, index, device, skipFilterSort)
		if err != nil {
			return nil, err
		}
		return sess, nil
	}
	return nil, err
}

func (s *Server) reportBadRelease(sess *session.Session, streamErr error) {
	errMsg := streamErr.Error()
	if !strings.Contains(errMsg, "compressed") && !strings.Contains(errMsg, "encrypted") &&
		!strings.Contains(errMsg, "EOF") && !errors.Is(streamErr, unpack.ErrTooManyZeroFills) {
		return
	}
	if s.availReporter != nil {
		s.availReporter.ReportBad(sess, errMsg)
	}
}

func (s *Server) handleDebugPlay(w http.ResponseWriter, r *http.Request, device *auth.Device) {
	nzbPath := r.URL.Query().Get("nzb")
	if nzbPath == "" {
		http.Error(w, "Missing 'nzb' query parameter (URL or file path)", http.StatusBadRequest)
		return
	}

	logger.Info("Debug Play request", "nzb", nzbPath)

	var nzbData []byte
	var err error

	if strings.HasPrefix(nzbPath, "/") || (len(nzbPath) > 2 && nzbPath[1] == ':') {

		logger.Debug("Reading NZB from local file", "path", nzbPath)
		nzbData, err = os.ReadFile(nzbPath)
		if err != nil {
			logger.Error("Failed to read local NZB file", "path", nzbPath, "err", err)
			http.Error(w, "Failed to read local NZB file: "+err.Error(), http.StatusInternalServerError)
			return
		}
	} else {

		dlCtx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		nzbData, err = s.indexer.DownloadNZB(dlCtx, nzbPath)
		cancel()
		if err != nil {

			httpClient := &http.Client{Timeout: 60 * time.Second}
			resp, httpErr := httpClient.Get(nzbPath)
			if httpErr != nil {
				logger.Error("Failed to download NZB", "url", nzbPath, "err", err, "httpErr", httpErr)
				http.Error(w, "Failed to download NZB: "+err.Error(), http.StatusInternalServerError)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				msg := fmt.Sprintf("Failed to download NZB (HTTP %d)", resp.StatusCode)
				logger.Error(msg, "url", nzbPath)
				http.Error(w, msg, http.StatusInternalServerError)
				return
			}

			nzbData, err = io.ReadAll(resp.Body)
			if err != nil {
				http.Error(w, "Failed to read NZB body", http.StatusInternalServerError)
				return
			}
		}
	}

	nzbParsed, err := nzb.Parse(bytes.NewReader(nzbData))
	if err != nil {
		logger.Error("Failed to parse NZB", "err", err)
		http.Error(w, "Failed to parse NZB", http.StatusInternalServerError)
		return
	}

	sessionID := fmt.Sprintf("debug-%x", nzbPath)

	sess, err := s.sessionManager.CreateSession(sessionID, nzbParsed, nil, nil)
	if err != nil {
		logger.Error("Failed to create session", "err", err)
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	files := sess.Files
	if len(files) == 0 {
		http.Error(w, "No files in NZB", http.StatusInternalServerError)
		return
	}
	unpackFiles := make([]unpack.UnpackableFile, len(files))
	for i := range files {
		unpackFiles[i] = files[i]
	}
	password := ""
	if sess.NZB != nil {
		password = sess.NZB.Password()
	}
	stream, name, size, bp, err := unpack.GetMediaStream(r.Context(), unpackFiles, sess.Blueprint, password)
	if bp != nil && sess.Blueprint == nil {
		sess.SetBlueprint(bp)
	}
	if err != nil {
		logger.Error("Failed to open media stream", "err", err)
		http.Error(w, "Failed to open media stream: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer stream.Close()

	if tStr := r.URL.Query().Get("t"); tStr != "" && r.Header.Get("Range") == "" {
		if tSec, parseOK := seek.ParseTSeconds(tStr); parseOK {
			if byteOffset, seekOK := seek.TimeToByteOffset(stream, size, name, tSec); seekOK {
				r.Header.Set("Range", "bytes="+strconv.FormatInt(byteOffset, 10)+"-")
			}
		}
	}

	clientIP, _, _ := net.SplitHostPort(r.RemoteAddr)
	if clientIP == "" {
		clientIP = r.RemoteAddr
	}

	s.sessionManager.StartPlayback(sessionID, clientIP)
	defer s.sessionManager.EndPlayback(sessionID, clientIP)

	monitoredStream := &StreamMonitor{
		ReadSeekCloser: stream,
		sessionID:      sessionID,
		clientIP:       clientIP,
		manager:        s.sessionManager,
		lastUpdate:     time.Now(),
	}

	logger.Info("Serving debug media", "name", name, "size", size)
	logger.Debug("HTTP Request", "method", r.Method, "range", r.Header.Get("Range"), "user_agent", r.Header.Get("User-Agent"))

	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Accept-Ranges", "bytes")
	w = newWriteTimeoutResponseWriter(w, 10*time.Minute)
	bufW := newBufferedResponseWriter(w, 256*1024)
	defer bufW.Flush()
	http.ServeContent(bufW, r, name, time.Time{}, monitoredStream)

	logger.Debug("Finished serving debug media")
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"addon":  "streamnzb",
	})
}

func streamScore(s Stream) int {
	return s.Score
}

func forceDisconnect(w http.ResponseWriter, baseURL string) {
	errorVideoURL := strings.TrimSuffix(baseURL, "/") + "/error/failure.mp4"
	logger.Info("Redirecting to error video", "url", errorVideoURL)

	w.Header().Set("Connection", "close")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	http.Redirect(w, &http.Request{Method: "GET"}, errorVideoURL, http.StatusTemporaryRedirect)
}

func (s *Server) Reload(opts *ServerOptions) {
	if opts == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.config = opts.Config
	s.baseURL = opts.BaseURL
	s.indexer = opts.Indexer
	s.validator = opts.Validator
	s.triageService = opts.TriageService
	s.availClient = opts.AvailClient
	if opts.AvailClient != nil {
		s.availReporter = availnzb.NewReporter(opts.AvailClient, opts.Validator)
		if err := opts.AvailClient.RefreshBackbones(); err != nil {
			logger.Debug("AvailNZB backbones refresh on reload", "err", err)
		}
	} else {
		s.availReporter = nil
	}
	s.availNZBIndexerHosts = opts.AvailNZBIndexerHosts
	s.tmdbClient = opts.TMDBClient
	s.tvdbClient = opts.TVDBClient
	s.deviceManager = opts.DeviceManager
	s.streamManager = opts.StreamManager
}

type writeTimeoutResponseWriter struct {
	http.ResponseWriter
	timeout time.Duration
	rc      *http.ResponseController
}

func newWriteTimeoutResponseWriter(w http.ResponseWriter, timeout time.Duration) *writeTimeoutResponseWriter {
	return &writeTimeoutResponseWriter{
		ResponseWriter: w,
		timeout:        timeout,
		rc:             http.NewResponseController(w),
	}
}

func (w *writeTimeoutResponseWriter) Write(p []byte) (n int, err error) {
	if setErr := w.rc.SetWriteDeadline(time.Now().Add(w.timeout)); setErr != nil {
		return 0, setErr
	}
	return w.ResponseWriter.Write(p)
}

type bufferedResponseWriter struct {
	http.ResponseWriter
	bw *bufio.Writer
}

func newBufferedResponseWriter(w http.ResponseWriter, size int) *bufferedResponseWriter {
	return &bufferedResponseWriter{
		ResponseWriter: w,
		bw:             bufio.NewWriterSize(w, size),
	}
}

func (b *bufferedResponseWriter) Write(p []byte) (n int, err error) {
	return b.bw.Write(p)
}

func (b *bufferedResponseWriter) Flush() {
	_ = b.bw.Flush()
	if f, ok := b.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

type StreamMonitor struct {
	io.ReadSeekCloser
	sessionID  string
	clientIP   string
	manager    *session.Manager
	lastUpdate time.Time
	mu         sync.Mutex
}

func (s *StreamMonitor) Read(p []byte) (n int, err error) {
	n, err = s.ReadSeekCloser.Read(p)
	if time.Since(s.lastUpdate) > 10*time.Second {
		s.mu.Lock()
		if time.Since(s.lastUpdate) > 10*time.Second {
			s.manager.KeepAlive(s.sessionID, s.clientIP)
			s.lastUpdate = time.Now()
		}
		s.mu.Unlock()
	}
	return n, err
}

func (s *StreamMonitor) Close() error {
	if s.ReadSeekCloser != nil {
		return s.ReadSeekCloser.Close()
	}
	return nil
}
