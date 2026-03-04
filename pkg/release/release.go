package release

import (
	"net"
	"net/url"
	"strconv"
	"strings"
	"unicode"
)

func IsPrivateReleaseURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return true
	}
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		host = u.Hostname()
	}
	ip := net.ParseIP(host)
	if ip != nil {
		return ip.IsPrivate() || ip.IsLoopback()
	}
	lower := strings.ToLower(host)
	return lower == "localhost" || strings.HasSuffix(lower, ".local")
}

type Release struct {
	Title         string
	Link          string
	DetailsURL    string
	Size          int64
	Indexer       string
	SourceIndexer interface{}

	PubDate     string
	GUID        string
	QuerySource string
	Grabs       int
	Languages   []string

	Available *bool
	Duration  float64
}

func Key(r *Release) string {
	if r == nil {
		return ""
	}
	if r.DetailsURL != "" {
		return r.DetailsURL
	}
	return NormalizeTitle(r.Title) + ":" + strconv.FormatInt(r.Size, 10)
}

func (r *Release) EqualByTitle(other *Release) bool {
	if r == nil || other == nil {
		return r == other
	}
	return NormalizeTitle(r.Title) == NormalizeTitle(other.Title)
}

func NormalizeTitle(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func NormalizeTitleForDedup(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	// Normalize "&" to "and" so "Law & Order" matches "Law and Order" from indexers.
	s = strings.ReplaceAll(s, "&", " and ")
	s = strings.Join(strings.Fields(s), " ")
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// NormalizeTitleLettersOnly returns a lowercase, letters-and-spaces-only form for fuzzy matching.
// Numbers, punctuation, and "&" (normalized to "and") are handled so years/versions don't affect title match.
// Dots and common separators become spaces so "Star.Trek.Starfleet" keeps word boundaries.
// Season/episode/year are filtered separately in FilterResults.
func NormalizeTitleLettersOnly(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "&", " and ")
	// Treat dots, dashes, underscores as word separators so release titles like "Show.Name.S01E01" tokenize.
	for _, sep := range []string{".", "-", "_", ":", "  "} {
		s = strings.ReplaceAll(s, sep, " ")
	}
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) {
			b.WriteRune(r)
		} else if r == ' ' || r == '\t' {
			b.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

var filenameReplacer = strings.NewReplacer(
	"ü", "ue", "Ü", "UE", "ö", "oe", "Ö", "OE", "ä", "ae", "Ä", "AE", "ß", "ss",
	"á", "a", "à", "a", "â", "a", "ã", "a", "é", "e", "è", "e", "ê", "e", "í", "i",
	"ó", "o", "ò", "o", "ô", "o", "ú", "u", "ù", "u", "û", "u", "ñ", "n", "ç", "c",
)

func NormalizeTitleForFilename(s string) string {
	return filenameReplacer.Replace(s)
}
