package stremio

import (
	"streamnzb/pkg/release"
	"streamnzb/pkg/search/parser"
)

// StreamResponse represents the response to a stream request
type StreamResponse struct {
	Streams []Stream `json:"streams"`
}

// Stream represents a single stream option
type Stream struct {
	// URL for direct streaming (HTTP video file)
	URL string `json:"url,omitempty"`

	// ExternalUrl for external player (alternative to URL)
	ExternalUrl string `json:"externalUrl,omitempty"`

	// Display name in Stremio
	Name string `json:"name,omitempty"`

	// Score from triage (higher = better); used for sorting, not sent to client
	Score int `json:"-"`

	// ParsedMetadata from triage (PTT-parsed release); used for resolution grouping, quality, etc.
	ParsedMetadata *parser.ParsedRelease `json:"-"`

	// Release is the canonical release for deduplication and identity; not sent to client
	Release *release.Release `json:"-"`

	// Optional metadata (shown in Stremio UI)
	Title         string         `json:"title,omitempty"`
	Description   string         `json:"description,omitempty"`
	BehaviorHints *BehaviorHints `json:"behaviorHints,omitempty"`
	StreamType    string         `json:"streamType,omitempty"`
}

// BehaviorHints provides hints to Stremio (and aggregators like AIOStreams) about stream behavior.
// See Stremio SDK: https://github.com/Stremio/stremio-addon-sdk/blob/master/docs/api/responses/stream.md
type BehaviorHints struct {
	NotWebReady      bool     `json:"notWebReady,omitempty"`
	BingeGroup       string   `json:"bingeGroup,omitempty"`
	CountryWhitelist []string `json:"countryWhitelist,omitempty"`
	VideoSize        int64    `json:"videoSize,omitempty"`
	Filename         string   `json:"filename,omitempty"`
	// Cached is true when AvailNZB reports the release as cached, false otherwise; omitted when unknown.
	Cached *bool `json:"cached,omitempty"`
}

// SearchReleasesResponse is the response for the search releases API (indexer + AvailNZB, tagged by availability and per-stream).
type SearchReleasesResponse struct {
	Streams  []SearchStreamInfo `json:"streams"`
	Releases []SearchReleaseTag `json:"releases"`
}

// SearchStreamInfo is one stream for the search UI (filter/sort by stream).
type SearchStreamInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// SearchReleaseTag is one release with availability and per-stream tags.
type SearchReleaseTag struct {
	Title        string            `json:"title"`
	Link         string            `json:"link"`
	DetailsURL   string            `json:"details_url"`
	Size         int64             `json:"size"`
	Indexer      string            `json:"indexer"`
	Availability string            `json:"availability"` // "Available", "Unavailable", "Unknown"
	StreamTags   []SearchStreamTag `json:"stream_tags"`
}

// SearchStreamTag is per-stream: does this release fit the stream's filters and its priority score.
type SearchStreamTag struct {
	StreamID   string `json:"stream_id"`
	StreamName string `json:"stream_name"`
	Fits       bool   `json:"fits"`
	Score      int    `json:"score"`
}
