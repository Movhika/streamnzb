package search

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"streamnzb/pkg/core/logger"
	"streamnzb/pkg/release"
	"streamnzb/pkg/search/parser"
)

var seriesFilterSuffixRE = regexp.MustCompile(`(?i)\s+s[0-9]{1,2}(?:e[0-9]{1,3})?$`)

func movieYearMatches(expectYear, gotYear int) bool {
	if expectYear <= 0 || gotYear <= 0 {
		return true
	}
	return gotYear >= expectYear-1 && gotYear <= expectYear+1
}

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

func parseSeriesFilterQuery(filterQuery string) string {
	norm := release.NormalizeTitle(filterQuery)
	norm = strings.TrimSpace(norm)
	if norm == "" {
		return ""
	}
	trimmed := strings.TrimSpace(seriesFilterSuffixRE.ReplaceAllString(norm, ""))
	if trimmed != "" {
		return trimmed
	}
	return norm
}

var titleArticles = map[string]bool{"the": true, "a": true, "an": true}

func titleWordsForMatch(s string) []string {
	if parsed := parser.ParseReleaseTitle(s); parsed != nil {
		if parsedTitle := strings.TrimSpace(parsed.Title); parsedTitle != "" {
			s = parsedTitle
		}
	}
	return release.NormalizeTitleWordsForMatch(s)
}

func fuzzyTitleMatches(expect, gotTitle string) bool {
	expectWords := titleWordsForMatch(expect)
	gotWords := titleWordsForMatch(gotTitle)
	if len(expectWords) == 0 {
		return true
	}
	if len(gotWords) == 0 {
		return false
	}
	if len(gotWords) < len(expectWords) {
		return false
	}
	if len(expectWords) == 1 {
		return len(gotWords) == 1 && gotWords[0] == expectWords[0]
	}
	// Reject when the release title has far more words than expected
	// (prevents "The Science of Interstellar" matching "Interstellar").
	if len(gotWords) > len(expectWords)+2 {
		return false
	}
	// Find expectWords as a contiguous block in gotWords.
	// Words before the block must all be common articles.
	for i := 0; i <= len(gotWords)-len(expectWords); i++ {
		match := true
		for j, w := range expectWords {
			if gotWords[i+j] != w {
				match = false
				break
			}
		}
		if match {
			for _, pre := range gotWords[:i] {
				if !titleArticles[pre] {
					return false
				}
			}
			return true
		}
	}
	return false
}

func normalizedTitleMatches(expect, gotTitle string) bool {
	expectNorm := release.NormalizeTitleForDedup(expect)
	gotNorm := release.NormalizeTitleForDedup(gotTitle)
	if gotNorm == "" {
		return false
	}
	if expectNorm == "" {
		return true
	}
	// Exact or prefix match (with optional 4-digit year suffix)
	if gotNorm == expectNorm {
		return true
	}
	if strings.HasPrefix(gotNorm, expectNorm) {
		rest := gotNorm[len(expectNorm):]
		if rest == "" {
			return true
		}
		if len(rest) == 4 {
			allDigit := true
			for _, r := range rest {
				if !unicode.IsDigit(r) {
					allDigit = false
					break
				}
			}
			if allDigit {
				return true
			}
		}
	}
	// Fall back to fuzzy: every expected word must appear as a word in the release title.
	return fuzzyTitleMatches(expect, gotTitle)
}

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
		expectTitle = parseSeriesFilterQuery(filterQuery)
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
			if expectEpisode > 0 {
				if !parsed.MatchesEpisodeRequest(expectSeason, expectEpisode) {
					logger.Trace("FilterResults dropped: episode_request",
						"expect_season", expectSeason,
						"expect_episode", expectEpisode,
						"got_seasons", parsed.Seasons,
						"got_episodes", parsed.Episodes,
						"complete", parsed.Complete,
						"release", rel.Title,
					)
					continue
				}
			} else if expectSeason > 0 && !parsed.HasSeason(expectSeason) {
				logger.Trace("FilterResults dropped: season",
					"expect_season", expectSeason,
					"got_seasons", parsed.Seasons,
					"release", rel.Title,
				)
				continue
			}
		}

		if !movieYearMatches(expectYear, parsed.Year) {
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
