package triage

import (
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
}

// NewService creates a new triage service
func NewService(filterConfig *config.FilterConfig, sortConfig config.SortConfig) *Service {
	return &Service{
		FilterConfig: filterConfig,
		SortConfig:   sortConfig,
	}
}

// Filter processes search results and returns candidates sorted by score
func (s *Service) Filter(releases []*release.Release) []Candidate {
	var candidates []Candidate

	for _, rel := range releases {
		if rel == nil {
			continue
		}
		// Parse title
		parsed := parser.ParseReleaseTitle(rel.Title)

		// Check if it passes filters
		if s.FilterConfig != nil {
			if !s.shouldInclude(rel, parsed) {
				continue // Skip this result
			}
		}

		// Determine group (preserved for metadata but no longer used for selection)
		group := parsed.ResolutionGroup()

		// Calculate score (order-list scaling 10→0 per category + grab/age)
		score := s.calculateScore(rel, parsed)

		// Prioritize ID-based results over text-based (ForceQuery dual search)
		if rel.QuerySource == "id" {
			score += 50_000_000 // Large boost so ID results sort first
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

	// Sort candidates by Score (descending)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	return candidates
}

// shouldInclude checks if a release passes all filter criteria (Include/Avoid per category).
func (s *Service) shouldInclude(rel *release.Release, parsed *parser.ParsedRelease) bool {
	cfg := s.FilterConfig
	if !checkQuality(cfg, parsed) ||
		!checkResolution(cfg, parsed) ||
		!checkCodec(cfg, parsed) ||
		!checkAudio(cfg, parsed) ||
		!checkChannels(cfg, parsed) ||
		!checkHDR(cfg, parsed) ||
		!checkBitDepth(cfg, parsed) ||
		!checkContainer(cfg, parsed) ||
		!checkEdition(cfg, parsed) ||
		!checkThreeD(cfg, parsed) ||
		!checkNetwork(cfg, parsed) ||
		!checkRegion(cfg, parsed) ||
		!checkLanguages(cfg, parsed, rel) ||
		!checkGroup(cfg, parsed) ||
		!checkBooleans(cfg, parsed) ||
		!checkSize(cfg, rel) ||
		!checkYear(cfg, parsed) {
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
	// 10 * (n - 1 - i) / (n - 1)
	return 10 * (n - 1 - firstMatchIndex) / (n - 1)
}

// firstMatchIndex returns the 0-based index of the first list option that matches the release for this category, or -1.
func (s *Service) firstMatchResolution(p *parser.ParsedRelease) int {
	group := pttoptions.NormalizeResolutionToGroup(p.Resolution)
	for i, opt := range s.SortConfig.ResolutionOrder {
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
	for i, opt := range s.SortConfig.CodecOrder {
		if strings.Contains(strings.ToLower(p.Codec), strings.ToLower(opt)) {
			return i
		}
	}
	return -1
}

func (s *Service) firstMatchAudio(p *parser.ParsedRelease) int {
	for i, opt := range s.SortConfig.AudioOrder {
		for _, a := range p.Audio {
			if strings.Contains(strings.ToLower(a), strings.ToLower(opt)) {
				return i
			}
		}
	}
	return -1
}

func (s *Service) firstMatchQuality(p *parser.ParsedRelease) int {
	for i, opt := range s.SortConfig.QualityOrder {
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
	for i, opt := range s.SortConfig.VisualTagOrder {
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
	for i, opt := range s.SortConfig.ChannelsOrder {
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
	g := strings.ToLower(p.Group)
	for i, opt := range s.SortConfig.GroupOrder {
		opt = strings.ToLower(strings.TrimSpace(opt))
		if opt == g || strings.Contains(g, opt) {
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
	for i, opt := range s.SortConfig.LanguagesOrder {
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
	order := s.SortConfig.ResolutionOrder
	if len(order) > 0 {
		score += orderListScore(len(order), s.firstMatchResolution(p))
	}
	order = s.SortConfig.CodecOrder
	if len(order) > 0 {
		score += orderListScore(len(order), s.firstMatchCodec(p))
	}
	order = s.SortConfig.AudioOrder
	if len(order) > 0 {
		score += orderListScore(len(order), s.firstMatchAudio(p))
	}
	order = s.SortConfig.QualityOrder
	if len(order) > 0 {
		score += orderListScore(len(order), s.firstMatchQuality(p))
	}
	order = s.SortConfig.VisualTagOrder
	if len(order) > 0 {
		score += orderListScore(len(order), s.firstMatchVisualTag(p))
	}
	order = s.SortConfig.ChannelsOrder
	if len(order) > 0 {
		score += orderListScore(len(order), s.firstMatchChannels(p))
	}
	order = s.SortConfig.BitDepthOrder
	if len(order) > 0 {
		score += orderListScore(len(order), s.firstMatchSingle(p.BitDepth, order))
	}
	order = s.SortConfig.ContainerOrder
	if len(order) > 0 {
		score += orderListScore(len(order), s.firstMatchSingle(p.Container, order))
	}
	order = s.SortConfig.LanguagesOrder
	if len(order) > 0 {
		score += orderListScore(len(order), s.firstMatchLanguages(p, rel))
	}
	order = s.SortConfig.GroupOrder
	if len(order) > 0 {
		score += orderListScore(len(order), s.firstMatchGroup(p))
	}
	order = s.SortConfig.EditionOrder
	if len(order) > 0 {
		score += orderListScore(len(order), s.firstMatchSingle(p.Edition, order))
	}
	order = s.SortConfig.NetworkOrder
	if len(order) > 0 {
		score += orderListScore(len(order), s.firstMatchSingle(p.Network, order))
	}
	order = s.SortConfig.RegionOrder
	if len(order) > 0 {
		score += orderListScore(len(order), s.firstMatchSingle(p.Region, order))
	}
	order = s.SortConfig.ThreeDOrder
	if len(order) > 0 {
		score += orderListScore(len(order), s.firstMatchSingle(p.ThreeD, order))
	}

	// Age
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
	// Grabs
	if rel.Grabs > 0 {
		score += int(float64(rel.Grabs) * s.SortConfig.GrabWeight)
	}
	return score
}

// deduplicateReleases removes duplicate releases based on normalized name
// Keeps the release with the highest score (best indexer, most grabs, etc.)
func (s *Service) deduplicateReleases(candidates []Candidate) []Candidate {
	seen := make(map[string]*Candidate)

	for i := range candidates {
		candidate := &candidates[i]

		// Use release.NormalizeTitleForDedup so minor formatting differences across indexers collapse
		normalized := release.NormalizeTitleForDedup(candidate.Release.Title)
		if normalized == "" {
			continue
		}

		existing, exists := seen[normalized]
		if !exists {
			seen[normalized] = candidate
			continue
		}

		// Keep the better release (higher score; on tie, prefer ID-based)
		if candidate.Score > existing.Score {
			seen[normalized] = candidate
		} else if candidate.Score == existing.Score && candidate.QuerySource == "id" && existing.QuerySource != "id" {
			seen[normalized] = candidate
		}
	}

	// Convert map back to slice
	result := make([]Candidate, 0, len(seen))
	for _, candidate := range seen {
		result = append(result, *candidate)
	}

	return result
}

