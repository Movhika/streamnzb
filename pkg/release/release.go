package release

import (
	"net"
	"net/url"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
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
	return strings.Join(normalizeTitleWords(s, true), "")
}

// NormalizeTitleLettersOnly returns a lowercase, letters-and-spaces-only form for fuzzy matching.
// Numbers, punctuation, and "&" (normalized to "and") are handled so years/versions don't affect title match.
// Dots and common separators become spaces so "Star.Trek.Starfleet" keeps word boundaries.
// Season/episode/year are filtered separately in FilterResults.
func NormalizeTitleLettersOnly(s string) string {
	return strings.Join(normalizeTitleWords(s, false), " ")
}

func NormalizeTitleWordsForMatch(s string) []string {
	return normalizeTitleWords(s, true)
}

func normalizeTitleWords(s string, keepDigits bool) []string {
	s = normalizeTitleForMatchBase(s)
	s = strings.ReplaceAll(s, "&", " and ")
	for _, sep := range []string{".", "-", "_", ":", "  "} {
		s = strings.ReplaceAll(s, sep, " ")
	}
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || (keepDigits && unicode.IsNumber(r)) {
			b.WriteRune(r)
		} else if r == ' ' || r == '\t' {
			b.WriteRune(' ')
		}
	}
	words := strings.Fields(b.String())
	for i, word := range words {
		words[i] = canonicalizeCommonTitleWord(word)
	}
	return words
}

func normalizeTitleForMatchBase(s string) string {
	s = strings.TrimSpace(s)
	s = repairMojibakeUTF8(s)
	s = strings.ToLower(s)
	s = stripDiacritics(s)
	return NormalizeTitleForFilename(s)
}

func canonicalizeCommonTitleWord(word string) string {
	switch word {
	case "pokmon", "pokamon":
		return "pokemon"
	default:
		return word
	}
}

func repairMojibakeUTF8(s string) string {
	best := s
	for range 2 {
		candidate, ok := decodeLatin1AsUTF8(best)
		if !ok || mojibakeScore(candidate) >= mojibakeScore(best) {
			break
		}
		best = candidate
	}
	return best
}

func decodeLatin1AsUTF8(s string) (string, bool) {
	buf := make([]byte, 0, len(s))
	for _, r := range s {
		if r > 255 {
			return "", false
		}
		buf = append(buf, byte(r))
	}
	if !utf8.Valid(buf) {
		return "", false
	}
	return string(buf), true
}

func mojibakeScore(s string) int {
	count := 0
	for _, r := range s {
		switch r {
		case 'Ã', 'Â', 'ã', 'â', '©', '€', '™', '�':
			count++
		}
	}
	return count
}

func stripDiacritics(s string) string {
	decomposed := norm.NFD.String(s)
	var b strings.Builder
	for _, r := range decomposed {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		b.WriteRune(r)
	}
	return norm.NFC.String(b.String())
}

var filenameReplacer = strings.NewReplacer(
	"ü", "ue", "Ü", "UE", "ö", "oe", "Ö", "OE", "ä", "ae", "Ä", "AE", "ß", "ss",
	"á", "a", "à", "a", "â", "a", "ã", "a", "é", "e", "è", "e", "ê", "e", "í", "i",
	"ó", "o", "ò", "o", "ô", "o", "ú", "u", "ù", "u", "û", "u", "ñ", "n", "ç", "c",
)

func NormalizeTitleForFilename(s string) string {
	return filenameReplacer.Replace(s)
}
