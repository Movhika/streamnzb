package search

import (
	"sort"
	"strconv"
	"strings"
	"unicode"

	"streamnzb/pkg/core/logger"
	"streamnzb/pkg/release"
	"streamnzb/pkg/search/parser"
)

// parseFilterQuery extracts normalized title and optional year from a filter query.
// For "Title 2014" returns (normalizedTitle, 2014). For "Title" or "Show S02E03" year is 0.
func parseFilterQuery(filterQuery string) (normTitle string, year int) {
	norm := release.NormalizeTitle(filterQuery)
	norm = strings.TrimSpace(norm)
	if norm == "" {
		return "", 0
	}
	for i := len(norm) - 1; i >= 0; i-- {
		if norm[i] >= '0' && norm[i] <= '9' {
			continue
		}
		if norm[i] == ' ' && i+1 < len(norm) {
			trailing := strings.TrimSpace(norm[i+1:])
			if len(trailing) == 4 {
				if y, err := strconv.Atoi(trailing); err == nil && y >= 1900 && y <= 2100 {
					return strings.TrimSpace(norm[:i]), y
				}
			}
		}
		break
	}
	return norm, 0
}

// normalizedTitleMatches returns true if got (release parsed title) matches expect (query title).
// Both sides are normalized to letters and digits only so "Terminator: Dark Fate" matches "Terminator Dark Fate".
// Equal or expect is prefix of got with only digits (e.g. year) after.
func normalizedTitleMatches(expect, gotTitle string) bool {
	expectNorm := release.NormalizeTitleForDedup(expect)
	gotNorm := release.NormalizeTitleForDedup(gotTitle)
	if gotNorm == "" {
		return false
	}
	if expectNorm == "" {
		return true
	}
	if gotNorm == expectNorm {
		return true
	}
	if !strings.HasPrefix(gotNorm, expectNorm) {
		return false
	}
	rest := gotNorm[len(expectNorm):]
	if rest == "" {
		return true
	}
	for _, r := range rest {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return len(rest) == 4
}

// FilterResults keeps releases that match the content: for movie normalized title, for series
// normalized title + season + episode; for all, if the release has a parsed year and we have
// an expected year, they must match.
func FilterResults(releases []*release.Release, contentType, filterQuery, season, episode string) []*release.Release {
	if contentType != "movie" && contentType != "series" {
		return releases
	}
	expectSeason, _ := strconv.Atoi(season)
	expectEpisode, _ := strconv.Atoi(episode)

	var expectTitle string
	var expectYear int
	if contentType == "movie" {
		expectTitle, expectYear = parseFilterQuery(filterQuery)
	} else {
		expectTitle = release.NormalizeTitle(strings.Split(filterQuery, " S")[0])
	}

	var out []*release.Release
	for _, rel := range releases {
		if rel == nil {
			continue
		}
		parsed := parser.ParseReleaseTitle(rel.Title)

		if contentType == "movie" {
			if !normalizedTitleMatches(expectTitle, parsed.Title) {
				logger.Trace("FilterResults dropped: title",
					"expect_title", expectTitle,
					"got_title", parsed.Title,
					"release", rel.Title,
				)
				continue
			}
		} else {
			if !normalizedTitleMatches(expectTitle, parsed.Title) {
				logger.Trace("FilterResults dropped: title",
					"expect_title", expectTitle,
					"got_title", parsed.Title,
					"release", rel.Title,
				)
				continue
			}
			if expectSeason > 0 && parsed.Season != expectSeason {
				logger.Trace("FilterResults dropped: season",
					"expect_season", expectSeason,
					"got_season", parsed.Season,
					"release", rel.Title,
				)
				continue
			}
			if expectEpisode > 0 && parsed.Episode != expectEpisode {
				logger.Trace("FilterResults dropped: episode",
					"expect_episode", expectEpisode,
					"got_episode", parsed.Episode,
					"release", rel.Title,
				)
				continue
			}
		}

		// For all: if release has a parsed year and we have an expected year, they must match
		if parsed.Year > 0 && expectYear > 0 && parsed.Year != expectYear {
			logger.Trace("FilterResults dropped: year",
				"expect_year", expectYear,
				"got_year", parsed.Year,
				"release", rel.Title,
			)
			continue
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
