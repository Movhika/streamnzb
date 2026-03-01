package release

import (
	"net"
	"net/url"
	"strconv"
	"strings"
	"unicode"
)

// IsPrivateReleaseURL returns true if the URL host is private/local (localhost).
// We must not report such URLs to AvailNZB — they are private and useless to others.
func IsPrivateReleaseURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return true // treat unparseable as private to be safe
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

// Release is a unified representation of an NZB release from indexers or AvailNZB.
// Used for comparison (by normalized title) and as a common type across the app.
// AvailNZB may return partial data (e.g. no SourceIndexer for download).
type Release struct {
	Title         string // Release name (e.g. "Movie.2024.1080p.BluRay.x264-GROUP")
	Link          string // NZB download URL
	DetailsURL    string // Stable identifier for AvailNZB/reporting
	Size          int64
	Indexer       string      // Actual indexer name (NZBGeek, Drunken Slug, etc.)
	SourceIndexer interface{} // Indexer client for DownloadNZB. Nil when from AvailNZB.

	// Optional fields from indexer search (empty when from AvailNZB)
	PubDate     string   // RFC1123/RFC1123Z for age scoring
	GUID        string   // For session ID when skipping validation
	QuerySource string   // "id" or "text" — ID-based results prioritized
	Grabs       int      // From newznab grabs attribute, for popularity scoring
	Languages   []string // From Newznab language attribute (e.g. "English", "German")

	Available *bool   // nil = unknown, true/false = checked via AvailNZB
	Duration  float64 // Duration in seconds (0 = unknown). Populated by Easynews.
}

// Key returns a stable key for the release: DetailsURL if set, otherwise "normalizedTitle:size".
// Used for deduplication and map keys (e.g. releaseScores).
func Key(r *Release) string {
	if r == nil {
		return ""
	}
	if r.DetailsURL != "" {
		return r.DetailsURL
	}
	return NormalizeTitle(r.Title) + ":" + strconv.FormatInt(r.Size, 10)
}

// EqualByTitle returns true if both releases have the same normalized title.
// Use for matching indexer results with AvailNZB releases (e.g. filtering RAR).
func (r *Release) EqualByTitle(other *Release) bool {
	if r == nil || other == nil {
		return r == other
	}
	return NormalizeTitle(r.Title) == NormalizeTitle(other.Title)
}

// NormalizeTitle normalizes a release title for comparison (lowercase, trimmed).
func NormalizeTitle(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// NormalizeTitleForDedup normalizes a title for deduplication by keeping only
// alphanumeric characters (letters and digits). Used so minor formatting
// differences (spaces, dots, parentheses, dashes) across indexers collapse.
func NormalizeTitleForDedup(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// NormalizeTitleForFilename maps common accented/umlaut characters to ASCII
// so that e.g. "König der Löwen" matches filenames like "Koenig der Loewen".
// Covers German (ü, ö, ä, ß) and common Latin accents.
var filenameReplacer = strings.NewReplacer(
	"ü", "ue", "Ü", "UE", "ö", "oe", "Ö", "OE", "ä", "ae", "Ä", "AE", "ß", "ss",
	"á", "a", "à", "a", "â", "a", "ã", "a", "é", "e", "è", "e", "ê", "e", "í", "i",
	"ó", "o", "ò", "o", "ô", "o", "ú", "u", "ù", "u", "û", "u", "ñ", "n", "ç", "c",
)

// NormalizeTitleForFilename returns s with umlauts and common accents replaced by ASCII equivalents.
func NormalizeTitleForFilename(s string) string {
	return filenameReplacer.Replace(s)
}
