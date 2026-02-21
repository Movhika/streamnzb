package indexer

import "encoding/xml"

// Caps represents the capabilities of a Newznab indexer as returned by the CAPS endpoint.
type Caps struct {
	Categories    []CapsCategory `json:"categories"`
	Searching     CapsSearching  `json:"searching"`
	Limits        CapsLimits     `json:"limits"`
	RetentionDays int            `json:"retention_days"`
}

// CapsCategory represents a searchable category (with optional subcategories).
type CapsCategory struct {
	ID      string         `xml:"id,attr" json:"id"`
	Name    string         `xml:"name,attr" json:"name"`
	Subcats []CapsCategory `xml:"subcat" json:"subcats,omitempty"`
}

// CapsSearching indicates which search functions the indexer supports.
type CapsSearching struct {
	Search      bool `json:"search"`
	TVSearch    bool `json:"tv_search"`
	MovieSearch bool `json:"movie_search"`
}

// CapsLimits contains the indexer's result limit constraints.
type CapsLimits struct {
	Max     int `json:"max"`
	Default int `json:"default"`
}

// capsXML is the raw XML structure returned by the CAPS endpoint.
type capsXML struct {
	XMLName    xml.Name          `xml:"caps"`
	Limits     capsXMLLimits     `xml:"limits"`
	Retention  capsXMLRetention  `xml:"retention"`
	Searching  capsXMLSearching  `xml:"searching"`
	Categories capsXMLCategories `xml:"categories"`
}

type capsXMLLimits struct {
	Max     int `xml:"max,attr"`
	Default int `xml:"default,attr"`
}

type capsXMLRetention struct {
	Days int `xml:"days,attr"`
}

type capsXMLSearching struct {
	Search      capsXMLSearchType `xml:"search"`
	TVSearch    capsXMLSearchType `xml:"tv-search"`
	MovieSearch capsXMLSearchType `xml:"movie-search"`
}

type capsXMLSearchType struct {
	Available string `xml:"available,attr"`
}

type capsXMLCategories struct {
	Categories []CapsCategory `xml:"category"`
}

// ParseCapsXML parses a raw CAPS XML response into a Caps struct.
func ParseCapsXML(data []byte) (*Caps, error) {
	var raw capsXML
	if err := xml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return &Caps{
		Categories: raw.Categories.Categories,
		Searching: CapsSearching{
			Search:      raw.Searching.Search.Available == "yes",
			TVSearch:    raw.Searching.TVSearch.Available == "yes",
			MovieSearch: raw.Searching.MovieSearch.Available == "yes",
		},
		Limits: CapsLimits{
			Max:     raw.Limits.Max,
			Default: raw.Limits.Default,
		},
		RetentionDays: raw.Retention.Days,
	}, nil
}

// IndexerWithCaps is an optional interface for indexers that support CAPS discovery.
type IndexerWithCaps interface {
	GetCaps() (*Caps, error)
}
