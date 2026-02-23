package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"streamnzb/pkg/auth"
	"streamnzb/pkg/core/logger"
	"streamnzb/pkg/services/metadata/tmdb"
)

// tmdbSearchResult is the JSON shape returned by GET /api/tmdb/search
type tmdbSearchResult struct {
	ID        string `json:"id"`         // For movie: tmdb id; for series: "tmdb:123"
	TMDBID    int    `json:"tmdb_id"`
	Title     string `json:"title"`
	Year      string `json:"year,omitempty"`
	MediaType string `json:"media_type"` // "movie" or "tv"
	IMDbID    string `json:"imdb_id,omitempty"`
	TVDBID    int    `json:"tvdb_id,omitempty"`
	PosterURL string `json:"poster_url,omitempty"` // Small poster for dropdown (w92)
}

func (s *Server) handleTMDBSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query().Get("q")
	if q == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]tmdbSearchResult{})
		return
	}
	client := tmdb.NewClient(s.tmdbAPIKey)
	resp, err := client.SearchMulti(q)
	if err != nil {
		logger.Debug("TMDB search failed", "query", q, "err", err)
		http.Error(w, "Search failed", http.StatusBadGateway)
		return
	}
	results := make([]tmdbSearchResult, 0, len(resp.Results))
	for _, r := range resp.Results {
		year := r.ReleaseDate
		if year == "" {
			year = r.FirstAirDate
		}
		if len(year) > 4 {
			year = year[:4]
		}
		title := r.Title
		if title == "" {
			title = r.Name
		}
		posterURL := ""
		if r.PosterPath != "" {
			posterURL = "https://image.tmdb.org/t/p/w92" + r.PosterPath
		}
		item := tmdbSearchResult{
			TMDBID:    r.ID,
			Title:     title,
			Year:      year,
			MediaType: r.MediaType,
			PosterURL: posterURL,
		}
		if r.MediaType == "movie" {
			item.ID = strconv.Itoa(r.ID)
			ext, err := client.GetExternalIDs(r.ID, "movie")
			if err == nil && ext.IMDbID != "" {
				item.IMDbID = ext.IMDbID
				item.ID = ext.IMDbID
			}
		} else {
			item.ID = "tmdb:" + strconv.Itoa(r.ID)
			ext, err := client.GetExternalIDs(r.ID, "tv")
			if err == nil {
				if ext.IMDbID != "" {
					item.IMDbID = ext.IMDbID
				}
				if ext.TVDBID != 0 {
					item.TVDBID = ext.TVDBID
				}
			}
		}
		results = append(results, item)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (s *Server) handleStreams(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	device, ok := auth.DeviceFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	contentType := r.URL.Query().Get("type")
	id := r.URL.Query().Get("id")
	if contentType == "" || id == "" {
		http.Error(w, "type and id are required", http.StatusBadRequest)
		return
	}
	if contentType != "movie" && contentType != "series" {
		http.Error(w, "type must be movie or series", http.StatusBadRequest)
		return
	}
	streams, err := s.strmServer.GetStreams(r.Context(), contentType, id, device)
	if err != nil {
		logger.Error("GetStreams failed", "type", contentType, "id", id, "err", err)
		http.Error(w, "Stream search failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"streams": streams})
}

func (s *Server) handleStreamsAvail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	device, ok := auth.DeviceFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	contentType := r.URL.Query().Get("type")
	id := r.URL.Query().Get("id")
	if contentType == "" || id == "" {
		http.Error(w, "type and id are required", http.StatusBadRequest)
		return
	}
	if contentType != "movie" && contentType != "series" {
		http.Error(w, "type must be movie or series", http.StatusBadRequest)
		return
	}
	streams, err := s.strmServer.GetAvailNZBStreams(r.Context(), contentType, id, device)
	if err != nil {
		logger.Error("GetAvailNZBStreams failed", "type", contentType, "id", id, "err", err)
		http.Error(w, "AvailNZB streams failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"streams": streams})
}
