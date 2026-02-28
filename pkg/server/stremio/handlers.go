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

// Server represents the Stremio addon HTTP server
type Server struct {
	mu                   sync.RWMutex
	manifest             *Manifest
	version              string // raw version for API/frontend (e.g. dev-9a3e479)
	baseURL              string
	config               *config.Config
	indexer              indexer.Indexer
	validator            *validation.Checker
	sessionManager       *session.Manager
	triageService        *triage.Service
	availClient          *availnzb.Client
	availReporter        *availnzb.Reporter
	availNZBIndexerHosts []string // Underlying indexer hostnames for AvailNZB GetReleases
	tmdbClient           *tmdb.Client
	tvdbClient           *tvdb.Client
	deviceManager        *auth.DeviceManager
	streamManager        *stream.Manager
	recentFailures       sync.Map // normalizedTitle → time.Time; short-lived cache to avoid re-validating known-dead releases across requests
	playListCache        sync.Map // key "streamId:contentType:id" -> *playListCacheEntry; reuse ordered list on play
	rawSearchCache       sync.Map // key "contentType:id" -> *rawSearchCacheEntry; one indexer/AvailNZB fetch shared by all streams
	nextReleaseIndex     sync.Map // key "token:streamId:contentType:id" -> *nextReleaseState; iterating "Next release" row + bound for same play session
	webHandler           http.Handler
	apiHandler           http.Handler
}

// FailoverOrderPath is the path AIOStreams POSTs to report failover order (bingeGroup list). Stored in session manager per device. No trailing slash.
const FailoverOrderPath = "/failover_order"

// NewServer creates a new Stremio addon server.
// availNZBIndexerHosts is used to filter AvailNZB GetReleases by indexer; pass nil to get all releases.
func NewServer(cfg *config.Config, baseURL string, port int, indexer indexer.Indexer, validator *validation.Checker,
	sessionMgr *session.Manager, triageService *triage.Service, availClient *availnzb.Client,
	availNZBIndexerHosts []string,
	tmdbClient *tmdb.Client, tvdbClient *tvdb.Client, deviceManager *auth.DeviceManager, streamManager *stream.Manager, version string) (*Server, error) {

	if version == "" {
		version = "dev"
	}
	var availReporter *availnzb.Reporter
	if availClient != nil {
		availReporter = availnzb.NewReporter(availClient, validator)
	}
	s := &Server{
		manifest:             NewManifest(version),
		version:              version,
		baseURL:              baseURL,
		config:               cfg,
		indexer:              indexer,
		validator:            validator,
		sessionManager:       sessionMgr,
		triageService:        triageService,
		availClient:          availClient,
		availReporter:        availReporter,
		availNZBIndexerHosts: availNZBIndexerHosts,
		tmdbClient:           tmdbClient,
		tvdbClient:           tvdbClient,
		deviceManager:        deviceManager,
		streamManager:        streamManager,
	}

	if err := s.CheckPort(port); err != nil {
		return nil, err
	}

	return s, nil
}

// CheckPort verifies if the specified port is available for the addon
func (s *Server) CheckPort(port int) error {
	address := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("addon port %d is already in use", port)
	}
	ln.Close()
	return nil
}

// SetWebHandler sets the handler for static web content (fallback)
func (s *Server) SetWebHandler(h http.Handler) {
	s.webHandler = h
}

// SetAPIHandler sets the handler for API requests
func (s *Server) SetAPIHandler(h http.Handler) {
	s.apiHandler = h
}

// Version returns the raw version for API/frontend (e.g. dev-9a3e479)
func (s *Server) Version() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.version != "" {
		return s.version
	}
	return "dev"
}

// SetupRoutes configures HTTP routes for the addon
func (s *Server) SetupRoutes(mux *http.ServeMux) {
	// Root handler for manifest and other routes
	// We use a custom handler to handle the optional token prefix
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.RLock()
		deviceManager := s.deviceManager
		webHandler := s.webHandler
		apiHandler := s.apiHandler
		s.mu.RUnlock()

		path := r.URL.Path
		var authenticatedDevice *auth.Device

		// Serve embedded error video directly - bypass token logic so /error/... is never treated as a device token
		if path == "/error/failure.mp4" && webHandler != nil {
			webHandler.ServeHTTP(w, r)
			return
		}

		// Determine if this is a Stremio route that requires device token
		isStremioRoute := path == "/manifest.json" || path == FailoverOrderPath || strings.HasPrefix(path, "/stream/") || strings.HasPrefix(path, "/play/") || strings.HasPrefix(path, "/next/") || strings.HasPrefix(path, "/debug/play")

		// Root path "/" and web UI routes are always accessible (no token required)
		// Only Stremio routes require device tokens in the path

		// Check for device token in path (only if path has a token segment)
		trimmedPath := strings.TrimPrefix(path, "/")
		parts := strings.SplitN(trimmedPath, "/", 2)

		if len(parts) >= 1 && parts[0] != "" {
			token := parts[0]

			// Try to authenticate as a device token
			if deviceManager != nil {
				device, err := deviceManager.AuthenticateToken(token, s.config.GetAdminUsername(), s.config.AdminToken)
				if err == nil && device != nil {
					authenticatedDevice = device
					// Strip token from path for internal routing
					if len(parts) > 1 {
						path = "/" + parts[1]
					} else {
						path = "/"
					}
					r.URL.Path = path
					// Store device in context for handlers to use
					r = r.WithContext(auth.ContextWithDevice(r.Context(), device))
				} else if isStremioRoute {
					// Token in path but doesn't match any device, and this is a Stremio route - unauthorized
					logger.Error("Unauthorized request - invalid device token", "path", path, "remote", r.RemoteAddr)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				// If token doesn't match but it's not a Stremio route, continue (might be web UI route like /login)
			}
		} else if isStremioRoute {
			// Stremio routes require device token in path
			logger.Error("Unauthorized request - Stremio route requires device token", "path", path, "remote", r.RemoteAddr)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		// If no token in path and not a Stremio route, allow access (for web UI routes like /, /login, and API routes which use cookies/headers)

		// Internal routing
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
				// API Handler expects /api/...
				// Current path is /api/... (token stripped)
				// Need to preserve the path for the API mux
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

// handleManifest serves the addon manifest
func (s *Server) handleManifest(w http.ResponseWriter, r *http.Request) {
	logger.Debug("Manifest request", "remote", r.RemoteAddr)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	s.mu.RLock()
	manifest := s.manifest
	s.mu.RUnlock()

	// Configure button (behaviorHints.configurable) only for admin users
	device, _ := auth.DeviceFromContext(r)
	isAdmin := device != nil && device.Username == s.config.GetAdminUsername()

	data, err := manifest.ToJSONForDevice(isAdmin)
	if err != nil {
		http.Error(w, "Failed to generate manifest", http.StatusInternalServerError)
		return
	}

	w.Write(data)
}

// handleStream handles stream requests
func (s *Server) handleStream(w http.ResponseWriter, r *http.Request, device *auth.Device) {
	// Parse URL: /stream/{type}/{id}.json
	path := strings.TrimPrefix(r.URL.Path, "/stream/")
	path = strings.TrimSuffix(path, ".json")

	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		http.Error(w, "Invalid stream URL", http.StatusBadRequest)
		return
	}

	contentType := parts[0] // "movie" or "series"
	id := parts[1]          // IMDb ID (tt1234567) or TMDB ID

	logger.Info("Stream request", "type", contentType, "id", id, "device", func() string {
		if device != nil {
			return device.Username
		}
		return "legacy"
	}())

	// Allow time for indexer search plus NNTP validation across providers.
	// 5s was too short: slow indexers + validation often exceeded it and returned 0 streams.
	const streamRequestTimeout = 30 * time.Second
	ctx, cancel := context.WithTimeout(r.Context(), streamRequestTimeout)
	defer cancel()

	logger.Trace("stream request start", "type", contentType, "id", id)
	isAIOStreams := strings.Contains(r.Header.Get("User-Agent"), "AIOStreams")
	var streams []Stream
	streamsList := s.streamConfigsForStreamRequest()
	for _, str := range streamsList {
		list, err := s.buildOrderedPlayList(ctx, str.ID, contentType, id, isAIOStreams)
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
		s.clearNextReleaseBound(device, str.ID, contentType, id)
		nameLeft := str.Name
		if nameLeft == "" {
			nameLeft = str.ID
		}
		token := ""
		if device != nil {
			token = device.Token
		}
		baseURL := strings.TrimSuffix(s.baseURL, "/")
		if token != "" {
			baseURL += "/" + token
		}
		showAllCandidates := str.ShowAllStream || isAIOStreams
		if showAllCandidates {
			for i, cand := range list.Candidates {
				relTitle := ""
				if cand.Release != nil && cand.Release.Title != "" {
					relTitle = cand.Release.Title
				} else {
					relTitle = fmt.Sprintf("Release %d", i+1)
				}
				isAvail := list.CachedAvailable != nil && cand.Release != nil && cand.Release.DetailsURL != "" && list.CachedAvailable[cand.Release.DetailsURL]
				streamName := nameLeft
				if isAvail {
					streamName = "⚡ " + nameLeft
				}
				desc := "StreamNZB\n" + relTitle
				playPath := streamSlotPrefix + str.ID + ":" + contentType + ":" + id + ":" + strconv.Itoa(i)
				streamURL := baseURL + "/play/" + playPath
				failoverId := "streamnzb-" + playPath
				bingeLabel := bingeGroupLabelFromMeta(cand.Metadata)
				if bingeLabel == "" {
					bingeLabel = nameLeft
				}
				hints := streamBehaviorHints(nameLeft, str.ID, cand.Release, &isAvail, bingeLabel)
				streams = append(streams, Stream{
					FailoverID:    failoverId,
					Name:          streamName,
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
			playPath := streamSlotPrefix + str.ID + ":" + contentType + ":" + id + ":0"
			streamURL := baseURL + "/play/" + playPath
			firstAvail := list.FirstIsAvailGood
			failoverId := "streamnzb-" + playPath
			bingeLabel := bingeGroupLabelFromMeta(firstMeta)
			if bingeLabel == "" {
				bingeLabel = nameLeft
			}
			hints := streamBehaviorHints(nameLeft, str.ID, firstRel, &firstAvail, bingeLabel)
			streams = append(streams, Stream{
				FailoverID:    failoverId,
				Name:          nameLeft,
				URL:           streamURL,
				Description:   description,
				BehaviorHints: hints,
			})
			if len(list.Candidates) >= 2 {
				nextPath := streamSlotPrefix + str.ID + ":" + contentType + ":" + id + ":0"
				nextURL := baseURL + "/next/" + nextPath
				nextName := nameLeft + " (next release)"
				nextDesc := "StreamNZB\nTry next release in list"
				nextFailoverId := "streamnzb-" + nextPath
				nextHints := streamBehaviorHints(nameLeft, str.ID, nil, nil, nameLeft)
				streams = append(streams, Stream{
					FailoverID:    nextFailoverId,
					Name:          nextName,
					URL:           nextURL,
					Description:   nextDesc,
					BehaviorHints: nextHints,
				})
			}
		}
		logger.Debug("Stream rows", "streamId", str.ID, "name", nameLeft, "candidates", len(list.Candidates), "showAllStream", str.ShowAllStream)
	}
	if streams == nil {
		streams = []Stream{}
	}

	response := StreamResponse{
		Streams: streams,
	}

	// Debug: Log the response
	responseJSON, _ := json.MarshalIndent(response, "", "  ")
	logger.Trace("Sending stream response", "json", string(responseJSON))

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	json.NewEncoder(w).Encode(response)
}

// failoverOrderRequest is the body AIOStreams POSTs to FailoverOrderPath.
// Body: { streams: [ { failoverId: "streamnzb-" + playPath }, ... ] }.
type failoverOrderRequest struct {
	Streams []struct {
		FailoverID string `json:"failoverId"`
	} `json:"streams"`
}

// handleFailoverOrder accepts POST with body { streams: [ { failoverId: "streamnzb-<playPath>" }, ... ] }.
// We store the order so we match by exact slot path when resolving play.
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
		// Slot path (stream:streamId:type:id:index) we keep as-is; otherwise validate stream ID exists
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

// GetStreams returns the catalog stream list (one row per stream config, plus optional "Next release" per stream).
// Used by the dashboard API and Stremio stream request.
func (s *Server) GetStreams(ctx context.Context, contentType, id string, device *auth.Device) ([]Stream, error) {
	const streamRequestTimeout = 30 * time.Second
	ctx, cancel := context.WithTimeout(ctx, streamRequestTimeout)
	defer cancel()
	var streams []Stream
	streamsList := s.streamConfigsForStreamRequest()
	for _, str := range streamsList {
		list, err := s.buildOrderedPlayList(ctx, str.ID, contentType, id, false)
		if err != nil || list == nil {
			continue
		}
		if len(list.Candidates) == 0 {
			continue
		}
		s.clearNextReleaseBound(device, str.ID, contentType, id)
		nameLeft := str.Name
		if nameLeft == "" {
			nameLeft = str.ID
		}
		token := ""
		if device != nil {
			token = device.Token
		}
		baseURL := strings.TrimSuffix(s.baseURL, "/")
		if token != "" {
			baseURL += "/" + token
		}
		if str.ShowAllStream {
			for i, cand := range list.Candidates {
				relTitle := ""
				if cand.Release != nil && cand.Release.Title != "" {
					relTitle = cand.Release.Title
				} else {
					relTitle = fmt.Sprintf("Release %d", i+1)
				}
				isAvail := list.CachedAvailable != nil && cand.Release != nil && cand.Release.DetailsURL != "" && list.CachedAvailable[cand.Release.DetailsURL]
				streamName := nameLeft
				if isAvail {
					streamName = "⚡ " + nameLeft
				}
				desc := "StreamNZB\n" + relTitle
				playPath := streamSlotPrefix + str.ID + ":" + contentType + ":" + id + ":" + strconv.Itoa(i)
				streamURL := baseURL + "/play/" + playPath
				failoverId := "streamnzb-" + playPath
				bingeLabel := bingeGroupLabelFromMeta(cand.Metadata)
				if bingeLabel == "" {
					bingeLabel = nameLeft
				}
				hints := streamBehaviorHints(nameLeft, str.ID, cand.Release, &isAvail, bingeLabel)
				streams = append(streams, Stream{
					FailoverID:    failoverId,
					Name:          streamName,
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
			playPath := streamSlotPrefix + str.ID + ":" + contentType + ":" + id + ":0"
			streamURL := baseURL + "/play/" + playPath
			firstAvail := list.FirstIsAvailGood
			failoverId := "streamnzb-" + playPath
			bingeLabel := bingeGroupLabelFromMeta(firstMeta)
			if bingeLabel == "" {
				bingeLabel = nameLeft
			}
			hints := streamBehaviorHints(nameLeft, str.ID, firstRel, &firstAvail, bingeLabel)
			streams = append(streams, Stream{
				FailoverID:    failoverId,
				Name:          nameLeft,
				URL:           streamURL,
				Description:   description,
				BehaviorHints: hints,
			})
			if len(list.Candidates) >= 2 {
				nextPath := streamSlotPrefix + str.ID + ":" + contentType + ":" + id + ":0"
				nextURL := baseURL + "/next/" + nextPath
				nextName := nameLeft + " (next release)"
				nextDesc := "StreamNZB\nTry next release in list"
				nextFailoverId := "streamnzb-" + nextPath
				nextHints := streamBehaviorHints(nameLeft, str.ID, nil, nil, nameLeft)
				streams = append(streams, Stream{
					FailoverID:    nextFailoverId,
					Name:          nextName,
					URL:           nextURL,
					Description:   nextDesc,
					BehaviorHints: nextHints,
				})
			}
		}
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

// addAPIKeyToDownloadURL appends the matching indexer's API key to the download URL (by host). Returns original if no match.
// For Newznab t=get URLs, the API expects parameter "id" (see https://inhies.github.io/Newznab-API/functions/#get);
// if the URL has "guid" but no "id", we set id=guid so indexers that require "id" work.
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

// getGlobalStream returns the default stream for catalog/play when no stream is specified. May return nil if streamManager not set.
func (s *Server) getGlobalStream() *stream.Stream {
	s.mu.RLock()
	sm := s.streamManager
	s.mu.RUnlock()
	if sm == nil {
		return nil
	}
	return sm.GetGlobal()
}

// getDefaultStreamID returns the stream id to use when the URL omits it (legacy 3-part slot). Never empty.
func (s *Server) getDefaultStreamID() string {
	if str := s.getGlobalStream(); str != nil && str.ID != "" {
		return str.ID
	}
	return stream.GlobalStreamID
}

// streamConfigsForStreamRequest returns the list of stream configs to show as rows (one per stream), in stable order: global first, then others by id. Never nil.
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
	// Global first, then rest by id so Stremio always sees the same order (global top, next-release second per stream).
	sort.Slice(list, func(i, j int) bool {
		a, b := list[i], list[j]
		if a == nil || b == nil {
			return a != nil
		}
		ai, bj := a.ID == stream.GlobalStreamID, b.ID == stream.GlobalStreamID
		if ai != bj {
			return ai // global first
		}
		if ai {
			return true // both global, keep order
		}
		return a.ID < b.ID
	})
	return list
}

// triageCandidates returns filtered+sorted candidates using the stream's filters and sorting.
func (s *Server) triageCandidates(str *stream.Stream, releases []*release.Release) []triage.Candidate {
	if str == nil {
		return s.triageService.Filter(releases)
	}
	ts := triage.NewService(&str.Filters, str.Sorting)
	return ts.Filter(releases)
}

// streamSinkKey is the context key for an optional StreamSink callback.
var streamSinkKey = struct{}{}

// streamSlotPrefix is the session ID prefix for play slots: stream:streamId:type:id:index (or legacy stream:type:id:index).
const streamSlotPrefix = "stream:"

// orderedPlayListResult holds the result of building the ordered play list (no validation).
type orderedPlayListResult struct {
	Candidates       []triage.Candidate
	FirstIsAvailGood bool
	Params           *SearchParams
	// CachedAvailable: detailsURL -> true if AvailNZB reports available. Nil if not set (e.g. from cache).
	CachedAvailable map[string]bool
}

// rawSearchResult holds indexer + AvailNZB results for a title (contentType+id). No stream-specific triage.
// Reused across all streams so we run TMDB/indexer/AvailNZB once per title.
type rawSearchResult struct {
	Params          *SearchParams
	AvailReleases   []*release.Release
	IndexerReleases []*release.Release
	CachedAvailable map[string]bool // detailsURL -> true if from AvailNZB available
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

func (s *Server) buildOrderedPlayList(ctx context.Context, streamId, contentType, id string, skipFilterSort bool) (*orderedPlayListResult, error) {
	if streamId == "" {
		streamId = s.getDefaultStreamID()
	}
	list, err := s.buildOrderedPlayListUncached(ctx, streamId, contentType, id, skipFilterSort)
	if err != nil || list == nil {
		return list, err
	}
	// Cache so play/next resolve the same list order.
	cacheKey := streamId + ":" + contentType + ":" + id
	if skipFilterSort {
		cacheKey += "|noFilter"
	}
	s.playListCache.Store(cacheKey, &playListCacheEntry{result: list, until: time.Now().Add(playListCacheTTL)})
	return list, nil
}

func (s *Server) buildOrderedPlayListUncached(ctx context.Context, streamId, contentType, id string, skipFilterSort bool) (*orderedPlayListResult, error) {
	raw, err := s.getOrBuildRawSearchResult(ctx, contentType, id)
	if err != nil || raw == nil {
		return nil, err
	}
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

// getOrBuildRawSearchResult runs TMDB + AvailNZB + indexer search once per (contentType, id); result is shared by all streams.
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

// buildRawSearchResult performs one indexer + AvailNZB fetch for the title. Uses default stream only for buildSearchParams (e.g. indexer overrides).
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

	// When we didn't pass an indexer filter to AvailNZB (e.g. only aggregators), filter AvailNZB results to
	// releases that appear in our indexer results (match by DetailsURL) so we don't show availability for
	// indexers the user doesn't have.
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
			// Mark as ID-based so triage scores them like indexer ID results (we fetched by IMDb/TVDB ID).
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

// GetSearchReleases returns all releases from indexers and AvailNZB for a title, with availability and per-stream tags.
// Used by the search UI to show full results and filter/sort by availability or stream.
func (s *Server) GetSearchReleases(ctx context.Context, contentType, id string) (*SearchReleasesResponse, error) {
	raw, err := s.getOrBuildRawSearchResult(ctx, contentType, id)
	if err != nil || raw == nil {
		return nil, err
	}

	// Build unified list: (release, availability). AvailNZB first (all reports), then indexer-only with "Unknown".
	type releaseWithAvail struct {
		rel   *release.Release
		avail string // "Available", "Unavailable", "Unknown"
	}
	seenDetailsURL := make(map[string]bool)
	seenTitleSize := make(map[string]bool)
	var unified []releaseWithAvail

	addKey := func(detailsURL string, title string, size int64) bool {
		if detailsURL != "" && seenDetailsURL[detailsURL] {
			return true
		}
		key := release.NormalizeTitle(title) + ":" + strconv.FormatInt(size, 10)
		if seenTitleSize[key] {
			return true
		}
		if detailsURL != "" {
			seenDetailsURL[detailsURL] = true
		}
		seenTitleSize[key] = true
		return false
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
			if addKey(r.DetailsURL, r.Title, r.Size) {
				continue
			}
			unified = append(unified, releaseWithAvail{rel: r, avail: avail})
		}
	}
	for _, r := range raw.IndexerReleases {
		if r == nil {
			continue
		}
		if addKey(r.DetailsURL, r.Title, r.Size) {
			continue
		}
		unified = append(unified, releaseWithAvail{rel: r, avail: "Unknown"})
	}

	// Per-release, per-stream: fits and score
	streams := s.streamConfigsForStreamRequest()
	releaseScores := make(map[string]map[string]struct {
		Fits  bool
		Score int
	}) // detailsURL -> streamId -> {Fits, Score}
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
			key := c.Release.DetailsURL
			if key == "" {
				key = release.NormalizeTitle(c.Release.Title) + ":" + strconv.FormatInt(c.Release.Size, 10)
			}
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
		// Mark releases that were not in candidates as not fitting (filtered out)
		for _, u := range unified {
			key := u.rel.DetailsURL
			if key == "" {
				key = release.NormalizeTitle(u.rel.Title) + ":" + strconv.FormatInt(u.rel.Size, 10)
			}
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

	// Build stream list for response
	streamInfos := make([]SearchStreamInfo, 0, len(streams))
	for _, str := range streams {
		if str != nil {
			streamInfos = append(streamInfos, SearchStreamInfo{ID: str.ID, Name: str.Name})
		}
	}

	// Build release list with tags
	releasesOut := make([]SearchReleaseTag, 0, len(unified))
	for _, u := range unified {
		r := u.rel
		key := r.DetailsURL
		if key == "" {
			key = release.NormalizeTitle(r.Title) + ":" + strconv.FormatInt(r.Size, 10)
		}
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

	return &SearchReleasesResponse{Streams: streamInfos, Releases: releasesOut}, nil
}

// releasesToCandidates converts releases to candidates with no stream filtering (score 0, preserve order).
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

// buildOrderedPlayListFromRaw applies one stream's filters/sorting to raw results (triage, merge, filter, sort).
// When skipFilterSort is true (e.g. AIOStreams), stream triage and sort are skipped; only merge, dedupe, and safety filters apply.
func (s *Server) buildOrderedPlayListFromRaw(raw *rawSearchResult, str *stream.Stream, skipFilterSort bool) (*orderedPlayListResult, error) {
	// Set of DetailsURLs that AvailNZB reports as unavailable — exclude these from Stremio play list.
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

	var availCandidates, indexerCandidates []triage.Candidate
	if skipFilterSort {
		availCandidates = releasesToCandidates(raw.AvailReleases)
		indexerCandidates = releasesToCandidates(raw.IndexerReleases)
	} else {
		availCandidates = s.triageCandidates(str, raw.AvailReleases)
		indexerCandidates = s.triageCandidates(str, raw.IndexerReleases)
	}

	seenURL := make(map[string]bool)
	var merged []triage.Candidate
	for _, c := range availCandidates {
		if c.Release == nil || c.Release.DetailsURL == "" {
			continue
		}
		if unavailableDetailsURLs[c.Release.DetailsURL] {
			continue
		}
		seenURL[c.Release.DetailsURL] = true
		merged = append(merged, c)
	}
	for _, c := range indexerCandidates {
		if c.Release == nil || c.Release.DetailsURL == "" {
			continue
		}
		if unavailableDetailsURLs[c.Release.DetailsURL] {
			continue
		}
		if seenURL[c.Release.DetailsURL] {
			continue
		}
		seenURL[c.Release.DetailsURL] = true
		merged = append(merged, c)
	}

	// Filter recent failures
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

	// Filter AvailNZB unhealthy for our providers
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
		// Sort by stream's score only. AvailNZB does not override the stream's priority: we only badge/serve
		// [availNZB] when the stream's #1 choice happens to be AvailNZB-good.
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

// streamScoreFromCandidate returns the triage score for ordering (same as Candidate.Score).
func streamScoreFromCandidate(c triage.Candidate) int {
	return c.Score
}

// StreamSink is called for each stream returned by GetStreams.
// Return false to stop receiving more streams.
type StreamSink func(Stream) bool

// WithStreamSink adds a sink to ctx. When GetStreams is called with this context,
// each stream in the result is passed to the sink (e.g. for WebSocket streaming).
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

// SearchParams holds the built request and IDs for a stream search (contentType + id).
// Built by buildSearchParams for use by buildOrderedPlayList and GetAvailNZBStreams.
type SearchParams struct {
	Req           indexer.SearchRequest
	ContentIDs    *session.AvailReportMeta
	ImdbForText   string
	TmdbForText   string
	AvailIndexers []string
}

// buildSearchParams builds the search request and content IDs for the given contentType and id.
// Used by buildOrderedPlayList and GetAvailNZBStreams. Indexer overrides come from the stream (v1 may be nil).
func (s *Server) buildSearchParams(contentType, id string, str *stream.Stream) (*SearchParams, error) {
	const searchLimit = 1000
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
	// Resolve TVDB (and optionally IMDb) for series so indexers get tvdbid+season+ep and AvailNZB gets correct IDs.
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
				req.IMDbID = contentIDs.ImdbID  // keep search request in sync so indexers and logging see IMDb
				imdbForText = contentIDs.ImdbID // so SearchParams and downstream use resolved IMDb
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
				// Resolve TMDB title once per unique (lang, includeYear, norm) and reuse for indexers to avoid N identical API calls
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
	return &SearchParams{
		Req:           req,
		ContentIDs:    contentIDs,
		ImdbForText:   imdbForText,
		TmdbForText:   tmdbForText,
		AvailIndexers: s.availNZBIndexerHosts,
	}, nil
}

// runAvailNZBPhase runs only the AvailNZB phase: fetch releases, triage, build streams.
// str is used for filters/sorting; device is used only for building play URLs (token).
// Used by GetAvailNZBStreams (e.g. search UI).
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

// GetAvailNZBStreams returns only streams from AvailNZB (no indexer search or validation).
// Used by the search UI to show cached-available results immediately.
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

// resolveStreamSlot builds the ordered play list for the given stream, creates a deferred session for the release at index, and sets fallback URLs.
// streamId empty means default stream; it is normalized to the actual id for sessionID and URLs.
// skipFilterSort when true (e.g. AIOStreams) uses raw order without stream filtering/sorting.
func (s *Server) resolveStreamSlot(ctx context.Context, streamId, contentType, id string, index int, device *auth.Device, skipFilterSort bool) (*session.Session, error) {
	if streamId == "" {
		streamId = s.getDefaultStreamID()
	}
	list, err := s.buildOrderedPlayList(ctx, streamId, contentType, id, skipFilterSort)
	if err != nil || list == nil {
		return nil, fmt.Errorf("build play list: %w", err)
	}
	if index < 0 || index >= len(list.Candidates) {
		return nil, fmt.Errorf("index %d out of range (candidates %d)", index, len(list.Candidates))
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
	sessionID := streamSlotPrefix + streamId + ":" + contentType + ":" + id + ":" + strconv.Itoa(index)
	_, err = s.sessionManager.CreateDeferredSession(sessionID, downloadURL, rel, idx, list.Params.ContentIDs)
	if err != nil {
		return nil, fmt.Errorf("create deferred session: %w", err)
	}
	token := ""
	if device != nil {
		token = device.Token
	}
	base := strings.TrimSuffix(s.baseURL, "/")
	if token != "" {
		base += "/" + token
	}
	// AIOStreams sends back BingeGroup per row; we use playPath as BingeGroup so each id is a full slot path.
	// Order entries are "stream:streamId:type:id:index" (exact match) or legacy stream ID (derive index by occurrence).
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
			// Legacy: entry is stream ID only — match by (streamId, index) occurrence
			if !strings.HasPrefix(entry, streamSlotPrefix) && entry == streamId {
				occurrence := 0
				for k := 0; k <= p; k++ {
					if order[k] == streamId {
						occurrence++
					}
				}
				if occurrence == index+1 {
					ourPosition = p
					break
				}
			}
		}
	}
	if ourPosition >= 0 && !matchedByPath {
		logger.Debug("Failover order: matched by legacy (stream ID + occurrence); send playPath as BingeGroup for exact order", "stream", streamId, "index", index)
	}

	if ourPosition >= 0 {
		for j := ourPosition + 1; j < len(order); j++ {
			entry := order[j]
			if strings.HasPrefix(entry, streamSlotPrefix) {
				fallbackURLs = append(fallbackURLs, base+"/play/"+entry)
			} else {
				// Legacy stream ID: k-th occurrence = candidate index k-1
				count := 0
				for k := 0; k <= j; k++ {
					if order[k] == entry {
						count++
					}
				}
				ci := count - 1
				fallbackURLs = append(fallbackURLs, base+"/play/"+streamSlotPrefix+entry+":"+contentType+":"+id+":"+strconv.Itoa(ci))
			}
		}
	} else {
		// No stored order (e.g. not AIOStreams): same-stream only.
		for i := index + 1; i < len(list.Candidates); i++ {
			fallbackURLs = append(fallbackURLs, base+"/play/"+streamSlotPrefix+streamId+":"+contentType+":"+id+":"+strconv.Itoa(i))
		}
	}

	// Log failover order as release names (use existing list when entry is same stream to avoid N buildOrderedPlayList calls)
	var names []string
	if ourPosition >= 0 && len(order) > 0 {
		for j := ourPosition; j < len(order); j++ {
			entry := order[j]
			if strings.HasPrefix(entry, streamSlotPrefix) {
				sid, _, _, idx, ok := parseStreamSlotID(entry)
				if ok && sid != "" {
					var cand *triage.Candidate
					if sid == streamId && idx < len(list.Candidates) {
						cand = &list.Candidates[idx]
					} else {
						nextList, _ := s.buildOrderedPlayList(ctx, sid, contentType, id, skipFilterSort)
						if nextList != nil && idx < len(nextList.Candidates) {
							cand = &nextList.Candidates[idx]
						}
					}
					if cand != nil && cand.Release != nil && cand.Release.Title != "" {
						names = append(names, cand.Release.Title)
					} else {
						names = append(names, "["+entry+"]")
					}
				} else {
					names = append(names, "["+entry+"]")
				}
			} else {
				count := 0
				for k := 0; k <= j; k++ {
					if order[k] == entry {
						count++
					}
				}
				ci := count - 1
				var cand *triage.Candidate
				if entry == streamId && ci < len(list.Candidates) {
					cand = &list.Candidates[ci]
				} else {
					nextList, _ := s.buildOrderedPlayList(ctx, entry, contentType, id, skipFilterSort)
					if nextList != nil && ci < len(nextList.Candidates) {
						cand = &nextList.Candidates[ci]
					}
				}
				if cand != nil && cand.Release != nil && cand.Release.Title != "" {
					names = append(names, cand.Release.Title)
				} else {
					names = append(names, "["+entry+":"+strconv.Itoa(ci)+"]")
				}
			}
		}
		logger.Info("Failover order (release names)", "stream", streamId, "contentType", contentType, "id", id, "ourPosition", ourPosition, "matchedBySlotPath", matchedByPath, "names", names)
	} else {
		for i := index; i < len(list.Candidates); i++ {
			if r := list.Candidates[i].Release; r != nil && r.Title != "" {
				names = append(names, r.Title)
			} else {
				names = append(names, fmt.Sprintf("[%d]", i))
			}
		}
		logger.Info("Failover order (release names)", "stream", streamId, "contentType", contentType, "id", id, "names", names)
	}
	s.sessionManager.SetFallbackStreams(sessionID, fallbackURLs)
	sess, err := s.sessionManager.GetSession(sessionID)
	if err != nil {
		return nil, err
	}
	return sess, nil
}

// nextReleaseState holds the next index to use and the currently bound index for /next/...:0.
// BoundIndex is set on first request so range/reconnect requests get the same play URL; cleared when stream list is requested.
type nextReleaseState struct {
	mu         sync.Mutex
	NextIndex  int // next index to use when user clicks "Next release" again
	BoundIndex int // current play index for /next/...:0 (-1 = not bound)
}

// isSlotFailed returns true if a session exists for the given stream slot and any of its files
// have exceeded the failure threshold (IsFailed), so that slot should be skipped when resolving "next".
func (s *Server) isSlotFailed(streamId, contentType, id string, slotIndex int) bool {
	slotSessionID := streamSlotPrefix + streamId + ":" + contentType + ":" + id + ":" + strconv.Itoa(slotIndex)
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

// handleNextRelease redirects to the next release in the ordered play list.
// For /next/stream[:streamId]:type:id:0 we use per-user state: first request binds to an index and all
// subsequent requests (range, reconnect) redirect to the same index until the user re-opens the stream list.
// When resolving the next index we skip slots that are already failed (session exists and has IsFailed()),
// so the first redirect goes to the first working slot and avoids double redirects (:0 → :1 failed → :2).
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

	var playIndex int
	if index == 0 {
		key := (func() string {
			if device != nil && device.Token != "" {
				return device.Token + ":" + streamId + ":" + contentType + ":" + id
			}
			return ":" + streamId + ":" + contentType + ":" + id
		})()
		isAIOStreams := strings.Contains(r.Header.Get("User-Agent"), "AIOStreams")
		list, err := s.buildOrderedPlayList(r.Context(), streamId, contentType, id, isAIOStreams)
		maxIdx := 0
		if err == nil && list != nil && len(list.Candidates) > 1 {
			maxIdx = len(list.Candidates) - 1
		}
		// Load or create state; if already bound (range/reconnect), reuse same index
		v, _ := s.nextReleaseIndex.LoadOrStore(key, &nextReleaseState{NextIndex: 1, BoundIndex: -1})
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
			// Skip slots that are already failed so the first redirect goes to a working slot (avoids :0 → :1 failed → :2).
			for playIndex <= maxIdx && s.isSlotFailed(streamId, contentType, id, playIndex) {
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

	nextSlot := streamSlotPrefix + streamId + ":" + contentType + ":" + id + ":" + strconv.Itoa(playIndex)
	base := strings.TrimSuffix(s.baseURL, "/")
	if device != nil && device.Token != "" {
		base += "/" + device.Token
	}
	nextURL := base + "/play/" + nextSlot
	logger.Info("Next release redirect", "from", sessionID, "to", nextSlot)
	w.Header().Set("Location", nextURL)
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	http.Redirect(w, r, nextURL, http.StatusTemporaryRedirect)
}

// clearNextReleaseBound clears the bound index for the "Next release" row when the user opens the stream list,
// so the next click on "Next release" advances to the next index. streamId is the stream config id.
func (s *Server) clearNextReleaseBound(device *auth.Device, streamId, contentType, id string) {
	if streamId == "" || contentType == "" || id == "" {
		return
	}
	key := ":"
	if device != nil && device.Token != "" {
		key = device.Token + ":" + streamId + ":" + contentType + ":" + id
	} else {
		key = ":" + streamId + ":" + contentType + ":" + id
	}
	if v, ok := s.nextReleaseIndex.Load(key); ok {
		if state, _ := v.(*nextReleaseState); state != nil {
			state.mu.Lock()
			state.BoundIndex = -1
			state.mu.Unlock()
		}
	}
}

// parseStreamSlotID parses stream slot paths.
// Legacy 3-part: "stream:contentType:id:index" → streamId="", contentType, id, index (streamId implied default).
// 4-part: "stream:streamId:contentType:id:index" (id may contain colons) → streamId, contentType, id, index.
// Returns streamId (empty for legacy), contentType, id, index and true, or zero values and false.
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

// handlePlay serves video content for a session.
// Each request creates its own stream from the cached blueprint.
// No stream sharing, no mutexes, no caching -- the shared segment
// cache in loader.File handles deduplication automatically.
// Phase 3: sessionID may be a stream slot "stream:type:id:index"; we create a deferred session on first hit and set fallbacks.
func (s *Server) handlePlay(w http.ResponseWriter, r *http.Request, device *auth.Device) {
	sessionID := strings.TrimPrefix(r.URL.Path, "/play/")
	logger.Info("Play request", "session", sessionID)

	sess, err := s.sessionManager.GetSession(sessionID)
	if err != nil {
		// Phase 3: resolve stream slot to a real session (create deferred for this index, set fallbacks)
		if streamId, contentType, id, index, ok := parseStreamSlotID(sessionID); ok {
			// Use unfiltered list when device has stored failover order (slot paths), so slot indices match the list we sent to AIOStreams.
			skipFilterSort := strings.Contains(r.Header.Get("User-Agent"), "AIOStreams")
			if !skipFilterSort && device != nil {
				order := s.sessionManager.GetDeviceFailoverOrder(device.Token)
				if len(order) > 0 && strings.HasPrefix(order[0], streamSlotPrefix) {
					skipFilterSort = true
				}
			}
			sess, err = s.resolveStreamSlot(r.Context(), streamId, contentType, id, index, device, skipFilterSort)
			if err != nil {
				logger.Debug("Resolve stream slot failed", "slot", sessionID, "err", err)
				http.Error(w, "Stream slot not found or invalid", http.StatusNotFound)
				return
			}
			// sess is now the created session (sessionID includes streamId)
		} else {
			http.Error(w, "Session expired or not found", http.StatusNotFound)
			return
		}
	}

	if _, err = sess.GetOrDownloadNZB(s.sessionManager); err != nil {
		logger.Error("Failed to lazy load NZB", "id", sessionID, "err", err)
		redirectToNextStreamOrError(w, r, s.baseURL, sess, true)
		return
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
			forceDisconnect(w, s.baseURL)
			return
		}
	}

	// If any file has exceeded its failure threshold, redirect immediately
	// instead of starting a stream that will fail on the first read.
	for _, f := range files {
		if f.IsFailed() {
			logger.Error("Session file has too many failures, redirecting to next stream", "session", sessionID, "file", f.Name())
			s.reportBadRelease(sess, loader.ErrTooManyZeroFills)
			if sess.NZB != nil {
				s.validator.InvalidateCache(sess.NZB.Hash())
			}
			redirectToNextStreamOrError(w, r, s.baseURL, sess, true)
			return
		}
	}

	// Each request gets its own stream, scoped to the HTTP request context.
	// When the client disconnects, r.Context() is cancelled, which propagates
	// down through VirtualStream -> SegmentReader -> DownloadSegment.
	password := ""
	if sess.NZB != nil {
		password = sess.NZB.Password()
	}
	stream, name, size, bp, err := unpack.GetMediaStream(r.Context(), files, sess.Blueprint, password)
	if bp != nil && sess.Blueprint == nil {
		sess.SetBlueprint(bp)
	}
	if err != nil {
		logger.Error("Failed to open media stream", "id", sessionID, "err", err)
		s.reportBadRelease(sess, err)
		if sess.NZB != nil {
			s.validator.InvalidateCache(sess.NZB.Hash())
		}
		redirectToNextStreamOrError(w, r, s.baseURL, sess, true)
		return
	}
	defer stream.Close()

	// Report successful fetch/stream to AvailNZB (lazy sessions weren't reported at catalog time)
	if s.availReporter != nil {
		s.availReporter.ReportGood(sess)
	}

	// If client sent t= (start time in seconds) and no Range, convert to byte offset for supported containers.
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

	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w = newWriteTimeoutResponseWriter(w, 10*time.Minute)
	bufW := newBufferedResponseWriter(w, 256*1024) // 256KB buffer to smooth NNTP→client throughput
	defer bufW.Flush()

	http.ServeContent(bufW, r, name, time.Time{}, monitoredStream)
	logger.Trace("Finished serving media", "session", sessionID)
}

// reportBadRelease reports unstreamable releases to AvailNZB in the background.
func (s *Server) reportBadRelease(sess *session.Session, streamErr error) {
	errMsg := streamErr.Error()
	if !strings.Contains(errMsg, "compressed") && !strings.Contains(errMsg, "encrypted") &&
		!strings.Contains(errMsg, "EOF") && !errors.Is(streamErr, loader.ErrTooManyZeroFills) {
		return
	}
	if s.availReporter != nil {
		s.availReporter.ReportBad(sess, errMsg)
	}
}

// handleDebugPlay allows playing directly from an NZB URL or local file for debugging
func (s *Server) handleDebugPlay(w http.ResponseWriter, r *http.Request, device *auth.Device) {
	nzbPath := r.URL.Query().Get("nzb")
	if nzbPath == "" {
		http.Error(w, "Missing 'nzb' query parameter (URL or file path)", http.StatusBadRequest)
		return
	}

	logger.Info("Debug Play request", "nzb", nzbPath)

	var nzbData []byte
	var err error

	// Check if it's a local file path (starts with / or drive letter on Windows)
	if strings.HasPrefix(nzbPath, "/") || (len(nzbPath) > 2 && nzbPath[1] == ':') {
		// Local file path
		logger.Debug("Reading NZB from local file", "path", nzbPath)
		nzbData, err = os.ReadFile(nzbPath)
		if err != nil {
			logger.Error("Failed to read local NZB file", "path", nzbPath, "err", err)
			http.Error(w, "Failed to read local NZB file: "+err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		// URL - try indexer download first (60s for debug play)
		dlCtx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		nzbData, err = s.indexer.DownloadNZB(dlCtx, nzbPath)
		cancel()
		if err != nil {
			// Fallback to HTTP GET with timeout to avoid hanging on slow/broken URLs
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

	// Parse NZB
	nzbParsed, err := nzb.Parse(bytes.NewReader(nzbData))
	if err != nil {
		logger.Error("Failed to parse NZB", "err", err)
		http.Error(w, "Failed to parse NZB", http.StatusInternalServerError)
		return
	}

	// Create Session
	// Use hash of path as ID to allow repeating same path
	sessionID := fmt.Sprintf("debug-%x", nzbPath)
	// Or use NZB hash
	// sessionID := nzbParsed.Hash()

	// Create/Get Session (no release metadata for debug path - no AvailNZB reporting)
	sess, err := s.sessionManager.CreateSession(sessionID, nzbParsed, nil, nil)
	if err != nil {
		logger.Error("Failed to create session", "err", err)
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Get Files
	files := sess.Files
	if len(files) == 0 {
		http.Error(w, "No files in NZB", http.StatusInternalServerError)
		return
	}

	password := ""
	if sess.NZB != nil {
		password = sess.NZB.Password()
	}
	stream, name, size, bp, err := unpack.GetMediaStream(r.Context(), files, sess.Blueprint, password)
	if bp != nil && sess.Blueprint == nil {
		sess.SetBlueprint(bp)
	}
	if err != nil {
		logger.Error("Failed to open media stream", "err", err)
		http.Error(w, "Failed to open media stream: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer stream.Close()

	// If client sent t= and no Range, convert to byte offset for supported containers (same as handlePlay).
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

// handleHealth serves health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"addon":  "streamnzb",
	})
}

// streamScore returns the triage score for sorting (higher = better). Uses the score from
// triage which respects the user's priority configuration (resolution, codec, etc.).
func streamScore(s Stream) int {
	return s.Score
}

// redirectToNextStreamOrError redirects to the next stream in the priority list if enableFailover is true and the session has fallback URLs; otherwise redirects to the error video.
func redirectToNextStreamOrError(w http.ResponseWriter, r *http.Request, baseURL string, sess *session.Session, enableFailover bool) {
	if enableFailover {
		if nextURL := sess.FirstFallbackStreamURL(); nextURL != "" {
			var positionLog string
			if r != nil {
				if t := r.URL.Query().Get("t"); t != "" {
					nextURL = appendQueryParam(nextURL, "t", t)
					positionLog = " with t=" + t
				}
			}
			if positionLog != "" {
				logger.Info("Redirecting to next stream"+positionLog, "url", nextURL)
			} else {
				logger.Info("Redirecting to next stream in priority list", "url", nextURL)
			}
			w.Header().Set("Connection", "close")
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			http.Redirect(w, &http.Request{Method: "GET"}, nextURL, http.StatusTemporaryRedirect)
			return
		}
	}
	forceDisconnect(w, baseURL)
}

// appendQueryParam appends key=value to url, using ? or & as needed.
func appendQueryParam(u, key, value string) string {
	parsed, err := url.Parse(u)
	if err != nil {
		return u
	}
	q := parsed.Query()
	q.Set(key, value)
	parsed.RawQuery = q.Encode()
	return parsed.String()
}

// forceDisconnect redirects to the embedded failure video when streaming is unavailable.
// The video is packaged with the binary and served from /error/failure.mp4.
func forceDisconnect(w http.ResponseWriter, baseURL string) {
	errorVideoURL := strings.TrimSuffix(baseURL, "/") + "/error/failure.mp4"
	logger.Info("Redirecting to error video", "url", errorVideoURL)

	w.Header().Set("Connection", "close")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	http.Redirect(w, &http.Request{Method: "GET"}, errorVideoURL, http.StatusTemporaryRedirect)
}

// Reload updates the server components at runtime
func (s *Server) Reload(cfg *config.Config, baseURL string, indexer indexer.Indexer, validator *validation.Checker,
	triage *triage.Service, avail *availnzb.Client, availNZBIndexerHosts []string,
	tmdbClient *tmdb.Client, tvdbClient *tvdb.Client, deviceManager *auth.DeviceManager, streamManager *stream.Manager) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.config = cfg
	s.baseURL = baseURL
	s.indexer = indexer
	s.validator = validator
	s.triageService = triage
	s.availClient = avail
	if avail != nil {
		s.availReporter = availnzb.NewReporter(avail, validator)
		if err := avail.RefreshBackbones(); err != nil {
			logger.Debug("AvailNZB backbones refresh on reload", "err", err)
		}
	} else {
		s.availReporter = nil
	}
	s.availNZBIndexerHosts = availNZBIndexerHosts
	s.tmdbClient = tmdbClient
	s.tvdbClient = tvdbClient
	s.deviceManager = deviceManager
	s.streamManager = streamManager
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

// bufferedResponseWriter wraps ResponseWriter with a bufio.Writer to smooth throughput
// when serving video (fewer small writes to the client, better MP4 playback).
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

// StreamMonitor wraps an io.ReadSeekCloser to provide keep-alive updates
type StreamMonitor struct {
	io.ReadSeekCloser
	sessionID  string
	clientIP   string
	manager    *session.Manager
	lastUpdate time.Time
	mu         sync.Mutex // Protect lastUpdate to be safe, though Read is usually serial
}

func (s *StreamMonitor) Read(p []byte) (n int, err error) {
	n, err = s.ReadSeekCloser.Read(p)

	// Non-blocking update check
	// We don't want to lock on every read, so just check time occasionally
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
