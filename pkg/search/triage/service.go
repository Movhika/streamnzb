package triage

import (
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"streamnzb/pkg/core/config"
	"streamnzb/pkg/core/config/pttoptions"
	"streamnzb/pkg/release"
	"streamnzb/pkg/search/parser"
)

type Candidate struct {
	Release     *release.Release
	Metadata    *parser.ParsedRelease
	Group       string
	SortKey     []int
	Score       int
	QuerySource string
}

type Service struct {
	FilterConfig *config.FilterConfig
	SortConfig   config.SortConfig

	compiledRegexExcluded  []*regexp.Regexp
	compiledRegexRequired  []*regexp.Regexp
	compiledRegexPreferred []*regexp.Regexp
}

func compilePatterns(patterns []string) []*regexp.Regexp {
	out := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		if p == "" {
			continue
		}
		re, err := regexp.Compile("(?i)" + p)
		if err == nil {
			out = append(out, re)
		}
	}
	return out
}

func NewService(filterConfig *config.FilterConfig, sortConfig config.SortConfig) *Service {
	s := &Service{
		FilterConfig: filterConfig,
		SortConfig:   sortConfig,
	}
	if filterConfig != nil {
		s.compiledRegexExcluded = compilePatterns(filterConfig.RegexExcluded)
		s.compiledRegexRequired = compilePatterns(filterConfig.RegexRequired)
	}
	s.compiledRegexPreferred = compilePatterns(sortConfig.RegexPreferred)
	return s
}

func (s *Service) Filter(releases []*release.Release) []Candidate {
	var candidates []Candidate

	for _, rel := range releases {
		if rel == nil {
			continue
		}
		parsed := parser.ParseReleaseTitle(rel.Title)

		if s.FilterConfig != nil {
			if !s.shouldInclude(rel, parsed) {
				continue
			}
		}

		group := parsed.ResolutionGroup()
		sortKey := s.buildSortKey(rel, parsed)
		score := keyToScore(sortKey)

		querySource := rel.QuerySource
		if querySource == "" {
			querySource = "id"
		}
		candidates = append(candidates, Candidate{
			Release:     rel,
			Metadata:    parsed,
			Group:       group,
			SortKey:     sortKey,
			Score:       score,
			QuerySource: querySource,
		})
	}

	candidates = s.deduplicateReleases(candidates)

	sort.Slice(candidates, func(i, j int) bool {
		a, b := candidates[i].SortKey, candidates[j].SortKey
		for k := 0; k < len(a) && k < len(b); k++ {
			if a[k] != b[k] {
				return a[k] > b[k]
			}
		}
		return len(a) > len(b)
	})

	return candidates
}

func (s *Service) shouldInclude(rel *release.Release, parsed *parser.ParsedRelease) bool {
	cfg := s.FilterConfig

	if isIncludedBypass(cfg, parsed, rel) {
		return true
	}

	if checkQualityExcluded(cfg, parsed) ||
		checkResolutionExcluded(cfg, parsed) ||
		checkCodecExcluded(cfg, parsed) ||
		checkAudioExcluded(cfg, parsed) ||
		checkChannelsExcluded(cfg, parsed) ||
		checkHDRExcluded(cfg, parsed) ||
		checkBitDepthExcluded(cfg, parsed) ||
		checkContainerExcluded(cfg, parsed) ||
		checkEditionExcluded(cfg, parsed) ||
		checkThreeDExcluded(cfg, parsed) ||
		checkNetworkExcluded(cfg, parsed) ||
		checkRegionExcluded(cfg, parsed) ||
		checkGroupExcluded(cfg, parsed) ||
		checkAvailNZBExcluded(cfg, rel) ||
		checkLanguagesExcluded(cfg, parsed, rel) ||
		checkKeywordsExcluded(cfg, rel) ||
		checkRegexExcluded(s.compiledRegexExcluded, rel) {
		return false
	}

	if checkQualityRequired(cfg, parsed) ||
		checkResolutionRequired(cfg, parsed) ||
		checkCodecRequired(cfg, parsed) ||
		checkAudioRequired(cfg, parsed) ||
		checkChannelsRequired(cfg, parsed) ||
		checkHDRRequired(cfg, parsed) ||
		checkBitDepthRequired(cfg, parsed) ||
		checkContainerRequired(cfg, parsed) ||
		checkEditionRequired(cfg, parsed) ||
		checkThreeDRequired(cfg, parsed) ||
		checkNetworkRequired(cfg, parsed) ||
		checkRegionRequired(cfg, parsed) ||
		checkGroupRequired(cfg, parsed) ||
		checkAvailNZBRequired(cfg, rel) ||
		checkLanguagesRequired(cfg, parsed, rel) ||
		checkKeywordsRequired(cfg, rel) ||
		checkRegexRequired(s.compiledRegexRequired, rel) {
		return false
	}

	if !checkBooleans(cfg, parsed) ||
		!checkSizeWithResolution(cfg, rel, parsed) ||
		!checkYear(cfg, parsed) ||
		!checkAge(cfg, rel) ||
		!checkBitrate(cfg, rel) {
		return false
	}
	return true
}

const keyToScoreBase = 10

const keyToScoreMaxExp = 18

func keyToScore(key []int) int {
	score := 0
	for i, v := range key {
		if v < 0 {
			v = 0
		}
		exp := len(key) - 1 - i
		if exp > keyToScoreMaxExp {
			exp = keyToScoreMaxExp
		}
		if exp <= 0 {
			score += v
		} else {
			score += v * int(math.Pow(float64(keyToScoreBase), float64(exp)))
		}
	}
	return score
}

func (s *Service) firstMatchResolution(p *parser.ParsedRelease) int {
	group := pttoptions.NormalizeResolutionToGroup(p.Resolution)
	for i, opt := range s.SortConfig.PreferredResolution {
		if strings.EqualFold(opt, group) || strings.Contains(strings.ToLower(p.Resolution), strings.ToLower(opt)) {
			return i
		}
	}
	return -1
}

func (s *Service) firstMatchCodec(p *parser.ParsedRelease) int {
	if p.Codec == "" {
		return -1
	}
	for i, opt := range s.SortConfig.PreferredCodec {
		if strings.Contains(strings.ToLower(p.Codec), strings.ToLower(opt)) {
			return i
		}
	}
	return -1
}

func (s *Service) firstMatchAudio(p *parser.ParsedRelease) int {
	for i, opt := range s.SortConfig.PreferredAudio {
		for _, a := range p.Audio {
			if strings.Contains(strings.ToLower(a), strings.ToLower(opt)) {
				return i
			}
		}
	}
	return -1
}

func (s *Service) firstMatchQuality(p *parser.ParsedRelease) int {
	for i, opt := range s.SortConfig.PreferredQuality {
		if strings.Contains(strings.ToLower(p.Quality), strings.ToLower(opt)) {
			return i
		}
	}
	return -1
}

func (s *Service) firstMatchVisualTag(p *parser.ParsedRelease) int {
	tags := make([]string, 0, len(p.HDR)+1)
	tags = append(tags, p.HDR...)
	if p.ThreeD != "" {
		tags = append(tags, p.ThreeD)
	}
	for i, opt := range s.SortConfig.PreferredVisualTag {
		optLower := strings.ToLower(opt)
		for _, tag := range tags {
			if strings.Contains(strings.ToLower(tag), optLower) || (optLower == "3d" && strings.HasPrefix(strings.ToLower(tag), "3d")) {
				return i
			}
		}
	}
	return -1
}

func (s *Service) firstMatchChannels(p *parser.ParsedRelease) int {
	for i, opt := range s.SortConfig.PreferredChannels {
		for _, ch := range p.Channels {
			if strings.EqualFold(ch, opt) {
				return i
			}
		}
	}
	return -1
}

func (s *Service) firstMatchGroup(p *parser.ParsedRelease) int {
	if p.Group == "" {
		return -1
	}
	g := strings.ToLower(strings.TrimSpace(p.Group))
	for i, opt := range s.SortConfig.PreferredGroup {
		opt = strings.ToLower(strings.TrimSpace(opt))
		if opt == g {
			return i
		}
	}
	return -1
}

func (s *Service) firstMatchSingle(value string, order []string) int {
	if value == "" {
		return -1
	}
	v := strings.ToLower(value)
	for i, opt := range order {
		if strings.Contains(v, strings.ToLower(opt)) {
			return i
		}
	}
	return -1
}

func (s *Service) firstMatchLanguages(p *parser.ParsedRelease, rel *release.Release) int {
	languages := mergeReleaseLanguages(p.Languages, nil)
	if rel != nil && len(rel.Languages) > 0 {
		languages = mergeReleaseLanguages(p.Languages, rel.Languages)
	}
	for i, opt := range s.SortConfig.PreferredLanguages {
		for _, lang := range languages {
			if languageMatches(opt, lang) {
				return i
			}
		}
	}
	return -1
}

func (s *Service) buildSortKey(rel *release.Release, p *parser.ParsedRelease) []int {
	type catContribution struct {
		order []string
		idx   int
	}

	boolMatch := func(matched bool) catContribution {
		if matched {
			return catContribution{[]string{"yes"}, 0}
		}
		return catContribution{[]string{"yes"}, -1}
	}

	kwMatched := false
	if len(s.SortConfig.KeywordsPreferred) > 0 {
		titleLower := strings.ToLower(rel.Title)
		for _, kw := range s.SortConfig.KeywordsPreferred {
			if kw != "" && strings.Contains(titleLower, strings.ToLower(kw)) {
				kwMatched = true
				break
			}
		}
	}

	rxMatched := false
	if len(s.compiledRegexPreferred) > 0 {
		for _, re := range s.compiledRegexPreferred {
			if re.MatchString(rel.Title) {
				rxMatched = true
				break
			}
		}
	}

	availStatus := getAvailStatus(rel)

	cats := map[string]catContribution{
		"resolution": {s.SortConfig.PreferredResolution, s.firstMatchResolution(p)},
		"quality":    {s.SortConfig.PreferredQuality, s.firstMatchQuality(p)},
		"codec":      {s.SortConfig.PreferredCodec, s.firstMatchCodec(p)},
		"visual_tag": {s.SortConfig.PreferredVisualTag, s.firstMatchVisualTag(p)},
		"audio":      {s.SortConfig.PreferredAudio, s.firstMatchAudio(p)},
		"channels":   {s.SortConfig.PreferredChannels, s.firstMatchChannels(p)},
		"bit_depth":  {s.SortConfig.PreferredBitDepth, s.firstMatchSingle(p.BitDepth, s.SortConfig.PreferredBitDepth)},
		"container":  {s.SortConfig.PreferredContainer, s.firstMatchSingle(p.Container, s.SortConfig.PreferredContainer)},
		"languages":  {s.SortConfig.PreferredLanguages, s.firstMatchLanguages(p, rel)},
		"group":      {s.SortConfig.PreferredGroup, s.firstMatchGroup(p)},
		"edition":    {s.SortConfig.PreferredEdition, s.firstMatchSingle(p.Edition, s.SortConfig.PreferredEdition)},
		"network":    {s.SortConfig.PreferredNetwork, s.firstMatchSingle(p.Network, s.SortConfig.PreferredNetwork)},
		"region":     {s.SortConfig.PreferredRegion, s.firstMatchSingle(p.Region, s.SortConfig.PreferredRegion)},
		"three_d":    {s.SortConfig.PreferredThreeD, s.firstMatchSingle(p.ThreeD, s.SortConfig.PreferredThreeD)},
		"keywords":   boolMatch(kwMatched),
		"regex":      boolMatch(rxMatched),
		"availnzb":   {s.SortConfig.PreferredAvailNZB, s.firstMatchSingle(availStatus, s.SortConfig.PreferredAvailNZB)},
	}

	defaultCategoryOrder := []string{
		"resolution", "quality", "codec", "visual_tag", "audio", "channels",
		"bit_depth", "container", "languages", "group", "edition", "network", "region", "three_d",
		"keywords", "regex", "availnzb",
	}

	var ordered []catContribution
	if len(s.SortConfig.SortCriteriaOrder) > 0 {
		for _, key := range s.SortConfig.SortCriteriaOrder {
			key = strings.TrimSpace(strings.ToLower(key))
			if key == "size" {
				continue
			}
			c, ok := cats[key]
			if !ok || len(c.order) == 0 {
				continue
			}
			ordered = append(ordered, c)
		}
	} else {
		for _, key := range defaultCategoryOrder {
			c, ok := cats[key]
			if !ok || len(c.order) == 0 {
				continue
			}
			ordered = append(ordered, c)
		}
	}

	key := make([]int, 0, len(ordered)+3)
	for _, c := range ordered {
		n := len(c.order)
		if n == 0 {
			continue
		}
		idx := c.idx
		if idx < 0 || idx >= n {
			key = append(key, 0)
		} else {
			key = append(key, n-idx)
		}
	}

	sizeGB := float64(rel.Size) / (1024 * 1024 * 1024)
	if sizeGB > 100 {
		key = append(key, 9)
	} else if sizeGB > 50 {
		key = append(key, 8)
	} else if sizeGB > 20 {
		key = append(key, 7)
	} else if sizeGB > 10 {
		key = append(key, 6)
	} else if sizeGB > 5 {
		key = append(key, 5)
	} else if sizeGB > 2 {
		key = append(key, 4)
	} else if sizeGB > 1 {
		key = append(key, 3)
	} else if sizeGB > 0.5 {
		key = append(key, 2)
	} else if sizeGB > 0 {
		key = append(key, 1)
	} else {
		key = append(key, 0)
	}

	if rel.PubDate != "" {
		pubTime, err := time.Parse(time.RFC1123Z, rel.PubDate)
		if err != nil {
			pubTime, err = time.Parse(time.RFC1123, rel.PubDate)
		}
		if err == nil {
			ageHours := time.Since(pubTime).Hours()
			key = append(key, int(100000.0-ageHours))
		} else {
			key = append(key, 0)
		}
	} else {
		key = append(key, 0)
	}

	key = append(key, rel.Grabs)

	return key
}

func (s *Service) deduplicateReleases(candidates []Candidate) []Candidate {
	seen := make(map[string]*Candidate)

	for i := range candidates {
		candidate := &candidates[i]

		normalized := release.NormalizeTitleForDedup(candidate.Release.Title)
		if normalized == "" {
			continue
		}

		existing, exists := seen[normalized]
		if !exists {
			seen[normalized] = candidate
			continue
		}

		if candidate.Score > existing.Score {
			seen[normalized] = candidate
		} else if candidate.Score == existing.Score && candidate.QuerySource == "id" && existing.QuerySource != "id" {
			seen[normalized] = candidate
		}
	}

	result := make([]Candidate, 0, len(seen))
	for _, candidate := range seen {
		result = append(result, *candidate)
	}

	return result
}
