package indexer

import (
	"context"
	"encoding/xml"
	"strconv"
	"strings"

	"streamnzb/pkg/core/config"
	"streamnzb/pkg/release"
)

// Indexer defines the interface for interacting with Usenet indexers
type Indexer interface {
	Search(req SearchRequest) (*SearchResponse, error)
	DownloadNZB(ctx context.Context, nzbURL string) ([]byte, error)
	Ping() error
	Name() string
	GetUsage() Usage
}

// IndexerWithResolve is an optional interface implemented by Aggregator (and indexers that can resolve proxy URLs).
// When a direct indexer URL comes from AvailNZB, ResolveDownloadURL searches by title/size/cat
// and returns the result's Link so DownloadNZB succeeds.
// cat is the Newznab category: "2000" for movies, "5000" for TV, or "" for general.
type IndexerWithResolve interface {
	ResolveDownloadURL(ctx context.Context, directURL, title string, size int64, cat string) (resolvedURL string, err error)
}

// Usage represents the current API and download usage for an indexer
type Usage struct {
	APIHitsLimit         int
	APIHitsUsed          int
	APIHitsRemaining     int
	DownloadsLimit       int
	DownloadsUsed        int
	DownloadsRemaining   int
	AllTimeAPIHitsUsed   int
	AllTimeDownloadsUsed int
}

// SearchRequest represents a search query
type SearchRequest struct {
	Query   string // Search query (may be set per-indexer by aggregator from PerIndexerQuery)
	IMDbID  string
	TMDBID  string
	TVDBID  string
	Cat     string
	Limit   int
	Season  string
	Episode string

	// Per-indexer effective config (handler sets; aggregator copies into OptionalOverrides per indexer)
	EffectiveByIndexer map[string]*config.IndexerSearchConfig `json:"-"`
	PerIndexerQuery    map[string][]string                    `json:"-"` // indexer name -> text queries (e.g. primary + original title for non-en movies)

	// Set by aggregator for each indexer before calling Search (merged effective config for this indexer)
	OptionalOverrides *config.IndexerSearchConfig `json:"-"`
}

// SearchResponse represents the Newznab XML response. After aggregation, items are normalized
// (NormalizeSearchResponse) so Link and Size are populated from Enclosure/attributes when missing.
// Releases is populated from Items by NormalizeSearchResponse for use by triage and handlers.
type SearchResponse struct {
	XMLName  xml.Name           `xml:"rss"`
	Channel  Channel            `xml:"channel"`
	Releases []*release.Release `xml:"-"` // Populated by NormalizeSearchResponse
}

// NewznabResponse contains metadata about the results
type NewznabResponse struct {
	Offset int `xml:"offset,attr"`
	Total  int `xml:"total,attr"`
}

// Channel contains the search results
type Channel struct {
	Response NewznabResponse `xml:"http://www.newznab.com/DTD/2010/feeds/attributes/ response"`
	Items    []Item          `xml:"item"`
}

// Item represents a single NZB result. After normalization, Link (or Enclosure.URL) is the NZB
// download URL and Size is set from enclosure length or size attribute when present.
type Item struct {
	Title       string      `xml:"title"`
	Link        string      `xml:"link"`
	GUID        string      `xml:"guid"`
	PubDate     string      `xml:"pubDate"`
	Category    string      `xml:"category"`
	Description string      `xml:"description"`
	Comments    string      `xml:"comments"` // Underlying indexer details URL when from aggregator (NZBHydra/Prowlarr); used for DetailsURL/AvailNZB match.
	Size        int64       `xml:"size"`
	Enclosure   Enclosure   `xml:"enclosure"`
	Attributes  []Attribute `xml:"attr"`

	// SourceIndexer is the indexer that provided this item
	// This is not part of the XML, but populated by the client
	SourceIndexer Indexer `xml:"-"`

	// ActualIndexer is the underlying indexer name when aggregated (from Newznab attributes).
	ActualIndexer string `xml:"-"`

	// ActualGUID is the underlying indexer GUID when aggregated (extracted from the link field).
	ActualGUID string `xml:"-"`

	// QuerySource is "id" or "text" when using ForceQuery dual search. Used to prioritize ID-based results.
	QuerySource string `xml:"-"`
}

// Attribute represents Newznab attributes
type Attribute struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

// Enclosure represents the enclosure tag (often contains size)
type Enclosure struct {
	URL    string `xml:"url,attr"`
	Length int64  `xml:"length,attr"`
	Type   string `xml:"type,attr"`
}

// GetAttribute retrieves a specific attribute from an item
func (i *Item) GetAttribute(name string) string {
	for _, attr := range i.Attributes {
		if attr.Name == name {
			return attr.Value
		}
	}
	return ""
}

// ToRelease returns a unified Release for comparison and use across the app.
func (i *Item) ToRelease() *release.Release {
	if i == nil {
		return nil
	}
	grabs := 0
	if s := i.GetAttribute("grabs"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			grabs = n
		}
	}
	// Extract language(s) from Newznab attribute (e.g. "English" or "English, Spanish")
	var languages []string
	if lang := i.GetAttribute("language"); lang != "" {
		for _, part := range strings.Split(lang, ",") {
			if t := strings.TrimSpace(part); t != "" {
				languages = append(languages, t)
			}
		}
	}
	indexerName := i.ActualIndexer
	if indexerName == "" && i.SourceIndexer != nil {
		indexerName = i.SourceIndexer.Name()
	}
	return &release.Release{
		Title:         i.Title,
		Link:          i.Link,
		DetailsURL:    i.ReleaseDetailsURL(),
		Size:          i.Size,
		Indexer:       indexerName,
		SourceIndexer: i.SourceIndexer,
		PubDate:       i.PubDate,
		GUID:          i.GUID,
		QuerySource:   i.QuerySource,
		Grabs:         grabs,
		Languages:     languages,
	}
}

// ReleaseDetailsURL returns the stable indexer details URL for this release (for AvailNZB and reporting).
// Most indexers use GUID or details_link for the details page; Link is typically the NZB download URL.
// When from an aggregator (NZBHydra/Prowlarr), Comments often holds the underlying indexer's details URL—
// use it so we match AvailNZB and dedupe correctly (same DetailsURL as AvailNZB reports).
func (i *Item) ReleaseDetailsURL() string {
	comments := strings.TrimSpace(i.Comments)
	if comments != "" && (strings.HasPrefix(comments, "http://") || strings.HasPrefix(comments, "https://")) {
		if idx := strings.Index(comments, "#"); idx >= 0 {
			comments = comments[:idx]
		}
		if comments != "" {
			return comments
		}
	}
	if i.ActualGUID != "" && strings.Contains(i.ActualGUID, "://") {
		return i.ActualGUID
	}
	if i.GUID != "" && strings.Contains(i.GUID, "://") {
		return i.GUID
	}
	return i.Link
}

// NormalizeItem fills Link and Size from Enclosure or attributes when missing, so all indexers
// produce a consistent Item shape regardless of backend XML differences.
// Call this after parsing search results so downstream code can rely on Link and Size.
func NormalizeItem(item *Item) {
	if item == nil {
		return
	}
	if item.Link == "" && item.Enclosure.URL != "" {
		item.Link = item.Enclosure.URL
	}
	if item.Size <= 0 {
		if item.Enclosure.Length > 0 {
			item.Size = item.Enclosure.Length
		} else if s := item.GetAttribute("size"); s != "" {
			if n, err := strconv.ParseInt(s, 10, 64); err == nil {
				item.Size = n
			}
		}
	}
}

// NormalizeSearchResponse runs NormalizeItem on every item and populates Releases.
func NormalizeSearchResponse(resp *SearchResponse) {
	if resp == nil {
		return
	}
	for i := range resp.Channel.Items {
		NormalizeItem(&resp.Channel.Items[i])
	}
	resp.Releases = make([]*release.Release, 0, len(resp.Channel.Items))
	for i := range resp.Channel.Items {
		if rel := resp.Channel.Items[i].ToRelease(); rel != nil {
			resp.Releases = append(resp.Releases, rel)
		}
	}
}
