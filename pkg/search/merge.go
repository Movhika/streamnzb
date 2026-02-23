package search

import (
	"sort"
	"strconv"
	"strings"
	"unicode"

	"streamnzb/pkg/release"
	"streamnzb/pkg/search/parser"
)

// movieQueryTitleAndYear splits a movie text query like "The Boys 1962" into normalized title and year.
// Returns (normalizedTitle, year). Year is 0 if not present or not a valid 4-digit year.
func movieQueryTitleAndYear(textQuery string) (title string, year int) {
	norm := release.NormalizeTitle(textQuery)
	norm = strings.TrimSpace(norm)
	if norm == "" {
		return "", 0
	}
	// Strip trailing 4-digit year
	for i := len(norm) - 1; i >= 0; i-- {
		if norm[i] >= '0' && norm[i] <= '9' {
			continue
		}
		if norm[i] == ' ' && i+1 < len(norm) {
			trailing := strings.TrimSpace(norm[i+1:])
			if len(trailing) == 4 {
				if y, err := strconv.Atoi(trailing); err == nil && y >= 1900 && y <= 2100 {
					year = y
					title = strings.TrimSpace(norm[:i])
					return title, year
				}
			}
		}
		break
	}
	return norm, 0
}

// movieTitleMatches returns true if the release's parsed title matches the search title strictly.
// expectTitle is normalized; we require the release title to equal expectTitle or to start with
// expectTitle followed only by space, digits (year), or end — so "the boys" matches "the boys"
// and "the boys 1962" but not "the boys next door" or "miracle the boys of 80".
func movieTitleMatches(expectTitle, gotTitle string) bool {
	got := release.NormalizeTitle(gotTitle)
	if got == "" {
		return false
	}
	if expectTitle == "" {
		return true
	}
	if got == expectTitle {
		return true
	}
	if !strings.HasPrefix(got, expectTitle) {
		return false
	}
	// expectTitle is a prefix; next character must be end, space, or digit (year)
	rest := got[len(expectTitle):]
	rest = strings.TrimLeft(rest, " ")
	if rest == "" {
		return true
	}
	// Allow only digits (year) after the title
	for _, r := range rest {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return len(rest) == 4 // single 4-digit year
}

// FilterTextResultsByContent keeps only releases where ptt-parsed title matches the content.
// For movies, matching is strict: title must be phrase/prefix match and year must match when present.
func FilterTextResultsByContent(releases []*release.Release, contentType, textQuery, season, episode string) []*release.Release {
	if contentType != "movie" && contentType != "series" {
		return releases
	}
	expectSeason, _ := strconv.Atoi(season)
	expectEpisode, _ := strconv.Atoi(episode)
	var expectShow string
	var expectMovieTitle string
	var expectMovieYear int
	if contentType == "movie" {
		expectMovieTitle, expectMovieYear = movieQueryTitleAndYear(textQuery)
	} else if contentType == "series" && (expectSeason > 0 || expectEpisode > 0) {
		expectShow = release.NormalizeTitle(strings.Split(textQuery, " S")[0])
	}

	var out []*release.Release
	for _, rel := range releases {
		if rel == nil {
			continue
		}
		parsed := parser.ParseReleaseTitle(rel.Title)
		if contentType == "movie" {
			if !movieTitleMatches(expectMovieTitle, parsed.Title) {
				continue
			}
			// When query includes a year, require release year to match (if release has a parsed year)
			if expectMovieYear > 0 && parsed.Year > 0 && parsed.Year != expectMovieYear {
				continue
			}
		} else {
			gotShow := release.NormalizeTitle(parsed.Title)
			if gotShow == "" {
				continue
			}
			if expectShow != "" && !strings.Contains(gotShow, expectShow) && !strings.Contains(expectShow, gotShow) {
				continue
			}
			if expectSeason > 0 && parsed.Season != expectSeason {
				continue
			}
			if expectEpisode > 0 && parsed.Episode != expectEpisode {
				continue
			}
		}
		out = append(out, rel)
	}
	return out
}

// MergeAndDedupeSearchResults merges ID and text results, preferring ID-based when duplicates.
func MergeAndDedupeSearchResults(releases []*release.Release) []*release.Release {
	sort.SliceStable(releases, func(i, j int) bool {
		return releases[i].QuerySource == "id" && releases[j].QuerySource != "id"
	})
	seenTitle := make(map[string]bool)
	var result []*release.Release
	for _, rel := range releases {
		if rel == nil {
			continue
		}
		normTitle := release.NormalizeTitleForDedup(rel.Title)
		if normTitle == "" {
			continue
		}
		if seenTitle[normTitle] {
			continue
		}
		seenTitle[normTitle] = true
		result = append(result, rel)
	}
	return result
}
