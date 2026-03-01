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

// Candidate represents a filtered search result suitable for deep inspection
type Candidate struct {
	Release     *release.Release
	Metadata    *parser.ParsedRelease
	Group       string // 4K, 1080p, 720p, SD
	Score       int
	QuerySource string // "id" or "text" — ID-based results are prioritized
}

// Service implements smart triage logic
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

// NewService creates a new triage service
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

// Filter processes search results and returns candidates sorted by score
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
		score := s.calculateScore(rel, parsed)

		if rel.QuerySource == "id" {
			score += 50_000_000
		}

		querySource := rel.QuerySource
		if querySource == "" {
			querySource = "id"
		}
		candidates = append(candidates, Candidate{
			Release:     rel,
			Metadata:    parsed,
			Group:       group,
			Score:       score,
			QuerySource: querySource,
		})
	}

	candidates = s.deduplicateReleases(candidates)

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	return candidates
}

// shouldInclude implements 3-tier filtering:
//  1. Included bypass: if the release matches ANY included rule in ANY category, it passes ALL filters.
//  2. Excluded: if the release matches any excluded rule, it is rejected.
//  3. Required: if a required list is set and the release doesn't match, it is rejected.
//  4. Booleans, size, and year are checked last.
func (s *Service) shouldInclude(rel *release.Release, parsed *parser.ParsedRelease) bool {
	cfg := s.FilterConfig

	// Phase 1: Included bypass (cross-category)
	if isIncludedBypass(cfg, parsed, rel) {
		return true
	}

	// Phase 2: Excluded checks
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
		checkLanguagesExcluded(cfg, parsed, rel) ||
		checkKeywordsExcluded(cfg, rel) ||
		checkRegexExcluded(s.compiledRegexExcluded, rel) {
		return false
	}

	// Phase 3: Required checks
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
		checkLanguagesRequired(cfg, parsed, rel) ||
		checkKeywordsRequired(cfg, rel) ||
		checkRegexRequired(s.compiledRegexRequired, rel) {
		return false
	}

	// Phase 4: Booleans, size, year, age, availnzb, bitrate
	if !checkBooleans(cfg, parsed) ||
		!checkSizeWithResolution(cfg, rel, parsed) ||
		!checkYear(cfg, parsed) ||
		!checkAge(cfg, rel) ||
		!checkAvailNZB(cfg, rel) ||
		!checkBitrate(cfg, rel) {
		return false
	}
	return true
}

// orderListScore returns points for first match in ordered list: 10 for 1st, scaling to 0 for last. n=1 => 10.
func orderListScore(n int, firstMatchIndex int) int {
	if n <= 0 {
		return 0
	}
	if n == 1 {
		return 10
	}
	if firstMatchIndex < 0 || firstMatchIndex >= n {
		return 0
	}
	return 10 * (n - 1 - firstMatchIndex) / (n - 1)
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

func (s *Service) calculateScore(rel *release.Release, p *parser.ParsedRelease) int {
	score := 0
	add := func(order []string, firstMatchIndex int, weight float64) {
		if len(order) == 0 {
			return
		}
		pts := orderListScore(len(order), firstMatchIndex)
		score += int(float64(pts) * weight)
	}

	type catContribution struct {
		order []string
		idx   int
	}
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
	}

	defaultCategoryOrder := []string{
		"resolution", "quality", "codec", "visual_tag", "audio", "channels",
		"bit_depth", "container", "languages", "group", "edition", "network", "region", "three_d",
	}

	type weightedCat struct {
		order []string
		idx   int
	}
	var ordered []weightedCat
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
			ordered = append(ordered, weightedCat{c.order, c.idx})
		}
	} else {
		for _, key := range defaultCategoryOrder {
			c, ok := cats[key]
			if !ok || len(c.order) == 0 {
				continue
			}
			ordered = append(ordered, weightedCat{c.order, c.idx})
		}
	}

	n := len(ordered)
	for i, wc := range ordered {
		weight := math.Pow(10, float64(n-1-i))
		add(wc.order, wc.idx, weight)
	}

	if rel.PubDate != "" {
		pubTime, err := time.Parse(time.RFC1123Z, rel.PubDate)
		if err != nil {
			pubTime, err = time.Parse(time.RFC1123, rel.PubDate)
		}
		if err == nil {
			ageHours := time.Since(pubTime).Hours()
			score += int((100000.0 - ageHours) * s.SortConfig.AgeWeight)
		}
	}
	if rel.Grabs > 0 {
		score += int(float64(rel.Grabs) * s.SortConfig.GrabWeight)
	}

	if len(s.SortConfig.KeywordsPreferred) > 0 && s.SortConfig.KeywordsWeight != 0 {
		titleLower := strings.ToLower(rel.Title)
		for _, kw := range s.SortConfig.KeywordsPreferred {
			if kw != "" && strings.Contains(titleLower, strings.ToLower(kw)) {
				score += int(10000 * s.SortConfig.KeywordsWeight)
				break
			}
		}
	}

	if len(s.compiledRegexPreferred) > 0 && s.SortConfig.RegexWeight != 0 {
		for _, re := range s.compiledRegexPreferred {
			if re.MatchString(rel.Title) {
				score += int(10000 * s.SortConfig.RegexWeight)
				break
			}
		}
	}

	if s.SortConfig.AvailNZBWeight != 0 && rel.Available != nil && *rel.Available {
		score += int(10000 * s.SortConfig.AvailNZBWeight)
	}

	return score
}

// deduplicateReleases removes duplicate releases based on normalized name.
// Keeps the release with the highest score.
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
