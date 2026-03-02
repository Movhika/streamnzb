package triage

import (
	"regexp"
	"strings"
	"time"

	"streamnzb/pkg/core/config"
	"streamnzb/pkg/core/config/pttoptions"
	"streamnzb/pkg/release"
	"streamnzb/pkg/search/parser"
)

func getAvailStatus(rel *release.Release) string {
	if rel == nil || rel.Available == nil {
		return "unknown"
	}
	if *rel.Available {
		return "available"
	}
	return "unavailable"
}

func matchSingle(value string, list []string) bool {
	if value == "" || len(list) == 0 {
		return false
	}
	v := strings.ToLower(strings.TrimSpace(value))
	for _, s := range list {
		if strings.Contains(v, strings.ToLower(s)) || strings.ToLower(s) == v {
			return true
		}
	}
	return false
}

func matchMulti(values []string, list []string) bool {
	if len(list) == 0 {
		return false
	}
	for _, v := range values {
		if v == "" {
			continue
		}
		if matchSingle(v, list) {
			return true
		}
	}
	return false
}

func matchResolution(p *parser.ParsedRelease, list []string) bool {
	group := pttoptions.NormalizeResolutionToGroup(p.Resolution)
	if matchSingle(group, list) {
		return true
	}
	return matchSingle(p.Resolution, list)
}

func matchVisualTags(p *parser.ParsedRelease, list []string) bool {
	tags := make([]string, 0, len(p.HDR)+1)
	tags = append(tags, p.HDR...)
	if p.ThreeD != "" {
		tags = append(tags, p.ThreeD)
	}
	return matchMulti(tags, list)
}

func matchGroup(group string, list []string) bool {
	if group == "" || len(list) == 0 {
		return false
	}
	g := strings.ToLower(strings.TrimSpace(group))
	for _, s := range list {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == g {
			return true
		}
	}
	return false
}

func isIncludedBypass(cfg *config.FilterConfig, p *parser.ParsedRelease, rel *release.Release) bool {
	if matchSingle(p.Quality, cfg.QualityIncluded) {
		return true
	}
	if matchResolution(p, cfg.ResolutionIncluded) {
		return true
	}
	if matchSingle(p.Codec, cfg.CodecIncluded) {
		return true
	}
	if matchMulti(p.Audio, cfg.AudioIncluded) {
		return true
	}
	if matchMulti(p.Channels, cfg.ChannelsIncluded) {
		return true
	}
	if matchVisualTags(p, cfg.HDRIncluded) {
		return true
	}
	if p.BitDepth != "" && matchSingle(p.BitDepth, cfg.BitDepthIncluded) {
		return true
	}
	if matchSingle(p.Container, cfg.ContainerIncluded) {
		return true
	}
	if matchSingle(p.Edition, cfg.EditionIncluded) {
		return true
	}
	if p.ThreeD != "" && matchSingle(p.ThreeD, cfg.ThreeDIncluded) {
		return true
	}
	if matchSingle(p.Network, cfg.NetworkIncluded) {
		return true
	}
	if matchSingle(p.Region, cfg.RegionIncluded) {
		return true
	}
	if matchGroup(p.Group, cfg.GroupIncluded) {
		return true
	}

	if len(cfg.LanguagesIncluded) > 0 {
		languages := mergeReleaseLanguages(p.Languages, nil)
		if rel != nil && len(rel.Languages) > 0 {
			languages = mergeReleaseLanguages(p.Languages, rel.Languages)
		}
		if matchLanguages(languages, cfg.LanguagesIncluded) {
			return true
		}
	}

	if matchSingle(getAvailStatus(rel), cfg.AvailNZBIncluded) {
		return true
	}
	return false
}

func checkQualityExcluded(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	return len(cfg.QualityExcluded) > 0 && matchSingle(p.Quality, cfg.QualityExcluded)
}

func checkResolutionExcluded(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	return len(cfg.ResolutionExcluded) > 0 && matchResolution(p, cfg.ResolutionExcluded)
}

func checkCodecExcluded(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	return len(cfg.CodecExcluded) > 0 && p.Codec != "" && matchSingle(p.Codec, cfg.CodecExcluded)
}

func checkAudioExcluded(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	return len(cfg.AudioExcluded) > 0 && matchMulti(p.Audio, cfg.AudioExcluded)
}

func checkChannelsExcluded(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	return len(cfg.ChannelsExcluded) > 0 && matchMulti(p.Channels, cfg.ChannelsExcluded)
}

func checkHDRExcluded(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	return len(cfg.HDRExcluded) > 0 && matchVisualTags(p, cfg.HDRExcluded)
}

func checkBitDepthExcluded(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	return len(cfg.BitDepthExcluded) > 0 && p.BitDepth != "" && matchSingle(p.BitDepth, cfg.BitDepthExcluded)
}

func checkContainerExcluded(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	return len(cfg.ContainerExcluded) > 0 && matchSingle(p.Container, cfg.ContainerExcluded)
}

func checkEditionExcluded(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	return len(cfg.EditionExcluded) > 0 && matchSingle(p.Edition, cfg.EditionExcluded)
}

func checkThreeDExcluded(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	return len(cfg.ThreeDExcluded) > 0 && p.ThreeD != "" && matchSingle(p.ThreeD, cfg.ThreeDExcluded)
}

func checkNetworkExcluded(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	return len(cfg.NetworkExcluded) > 0 && matchSingle(p.Network, cfg.NetworkExcluded)
}

func checkRegionExcluded(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	return len(cfg.RegionExcluded) > 0 && matchSingle(p.Region, cfg.RegionExcluded)
}

func checkGroupExcluded(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	return len(cfg.GroupExcluded) > 0 && matchGroup(p.Group, cfg.GroupExcluded)
}

func checkAvailNZBExcluded(cfg *config.FilterConfig, rel *release.Release) bool {
	return len(cfg.AvailNZBExcluded) > 0 && matchSingle(getAvailStatus(rel), cfg.AvailNZBExcluded)
}

func checkLanguagesExcluded(cfg *config.FilterConfig, p *parser.ParsedRelease, rel *release.Release) bool {
	if len(cfg.LanguagesExcluded) == 0 {
		return false
	}
	languages := mergeReleaseLanguages(p.Languages, nil)
	if rel != nil && len(rel.Languages) > 0 {
		languages = mergeReleaseLanguages(p.Languages, rel.Languages)
	}
	for _, lang := range languages {
		for _, excl := range cfg.LanguagesExcluded {
			if languageMatches(excl, lang) {
				return true
			}
		}
	}
	return false
}

func checkQualityRequired(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	return len(cfg.QualityRequired) > 0 && !matchSingle(p.Quality, cfg.QualityRequired)
}

func checkResolutionRequired(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	return len(cfg.ResolutionRequired) > 0 && !matchResolution(p, cfg.ResolutionRequired)
}

func checkCodecRequired(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	if len(cfg.CodecRequired) == 0 {
		return false
	}
	if p.Codec == "" {
		return true
	}
	return !matchSingle(p.Codec, cfg.CodecRequired)
}

func checkAudioRequired(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	return len(cfg.AudioRequired) > 0 && !matchMulti(p.Audio, cfg.AudioRequired)
}

func checkChannelsRequired(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	return len(cfg.ChannelsRequired) > 0 && !matchMulti(p.Channels, cfg.ChannelsRequired)
}

func checkHDRRequired(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	return len(cfg.HDRRequired) > 0 && !matchVisualTags(p, cfg.HDRRequired)
}

func checkBitDepthRequired(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	return len(cfg.BitDepthRequired) > 0 && p.BitDepth != "" && !matchSingle(p.BitDepth, cfg.BitDepthRequired)
}

func checkContainerRequired(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	return len(cfg.ContainerRequired) > 0 && !matchSingle(p.Container, cfg.ContainerRequired)
}

func checkEditionRequired(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	return len(cfg.EditionRequired) > 0 && !matchSingle(p.Edition, cfg.EditionRequired)
}

func checkThreeDRequired(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	return len(cfg.ThreeDRequired) > 0 && p.ThreeD != "" && !matchSingle(p.ThreeD, cfg.ThreeDRequired)
}

func checkNetworkRequired(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	return len(cfg.NetworkRequired) > 0 && !matchSingle(p.Network, cfg.NetworkRequired)
}

func checkRegionRequired(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	return len(cfg.RegionRequired) > 0 && !matchSingle(p.Region, cfg.RegionRequired)
}

func checkGroupRequired(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	return len(cfg.GroupRequired) > 0 && !matchGroup(p.Group, cfg.GroupRequired)
}

func checkAvailNZBRequired(cfg *config.FilterConfig, rel *release.Release) bool {
	return len(cfg.AvailNZBRequired) > 0 && !matchSingle(getAvailStatus(rel), cfg.AvailNZBRequired)
}

func checkLanguagesRequired(cfg *config.FilterConfig, p *parser.ParsedRelease, rel *release.Release) bool {
	if len(cfg.LanguagesRequired) == 0 {
		return false
	}
	languages := mergeReleaseLanguages(p.Languages, nil)
	if rel != nil && len(rel.Languages) > 0 {
		languages = mergeReleaseLanguages(p.Languages, rel.Languages)
	}
	if len(languages) == 0 {
		return true
	}
	return !matchLanguages(languages, cfg.LanguagesRequired)
}

func checkBooleans(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	if cfg.DubbedExcluded != nil && *cfg.DubbedExcluded && p.Dubbed {
		return false
	}
	if cfg.HardcodedExcluded != nil && *cfg.HardcodedExcluded && p.Hardcoded {
		return false
	}
	if cfg.ProperRequired != nil && *cfg.ProperRequired && !p.Proper {
		return false
	}
	if cfg.RepackRequired != nil && *cfg.RepackRequired && !p.Repack {
		return false
	}
	if cfg.RepackExcluded != nil && *cfg.RepackExcluded && p.Repack {
		return false
	}
	if cfg.ExtendedRequired != nil && *cfg.ExtendedRequired && !p.Extended {
		return false
	}
	if cfg.UnratedRequired != nil && *cfg.UnratedRequired && !p.Unrated {
		return false
	}
	return true
}

func languageMatches(configVal, releaseLang string) bool {
	c := strings.TrimSpace(strings.ToLower(configVal))
	r := strings.TrimSpace(strings.ToLower(releaseLang))
	if c == r {
		return true
	}
	isoToFull := map[string]string{
		"en": "english", "de": "german", "fr": "french", "es": "spanish", "it": "italian",
		"ja": "japanese", "ko": "korean", "zh": "chinese", "zh-tw": "chinese", "ru": "russian",
		"pt": "portuguese", "nl": "dutch", "pl": "polish", "tr": "turkish", "ar": "arabic",
		"hi": "hindi", "uk": "ukrainian", "da": "danish", "fi": "finnish", "fin": "finnish", "sv": "swedish",
		"no": "norwegian", "el": "greek", "lt": "lithuanian", "lv": "latvian", "et": "estonian",
		"cs": "czech", "sk": "slovak", "hu": "hungarian", "ro": "romanian", "bg": "bulgarian",
		"sr": "serbian", "hr": "croatian", "sl": "slovenian", "te": "telugu", "ta": "tamil",
		"ml": "malayalam", "kn": "kannada", "mr": "marathi", "gu": "gujarati", "pa": "punjabi",
		"bn": "bengali", "vi": "vietnamese", "id": "indonesian", "th": "thai", "ms": "malay",
		"he": "hebrew", "fa": "persian", "es-419": "spanish",
	}
	if full, ok := isoToFull[c]; ok && full == r {
		return true
	}
	for iso, full := range isoToFull {
		if full == c && (r == full || r == iso) {
			return true
		}
	}
	if c == "multi" && (strings.HasPrefix(r, "multi") || r == "multi") {
		return true
	}
	return false
}

func mergeReleaseLanguages(parsed []string, fromRelease []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, s := range parsed {
		k := strings.TrimSpace(strings.ToLower(s))
		if k != "" && !seen[k] {
			seen[k] = true
			out = append(out, s)
		}
	}
	for _, s := range fromRelease {
		k := strings.TrimSpace(strings.ToLower(s))
		if k != "" && !seen[k] {
			seen[k] = true
			out = append(out, s)
		}
	}
	return out
}

func matchLanguages(languages []string, list []string) bool {
	for _, lang := range languages {
		for _, allowed := range list {
			if languageMatches(allowed, lang) {
				return true
			}
		}
	}
	return false
}

func checkSize(cfg *config.FilterConfig, rel *release.Release) bool {
	if rel == nil || rel.Size <= 0 {
		return false
	}
	sizeGB := float64(rel.Size) / (1024 * 1024 * 1024)
	if cfg.MinSizeGB > 0 && sizeGB < cfg.MinSizeGB {
		return false
	}
	if cfg.MaxSizeGB > 0 && sizeGB > cfg.MaxSizeGB {
		return false
	}
	return true
}

func checkYear(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	if cfg.MinYear > 0 && p.Year > 0 && p.Year < cfg.MinYear {
		return false
	}
	if cfg.MaxYear > 0 && p.Year > 0 && p.Year > cfg.MaxYear {
		return false
	}
	return true
}

func checkAge(cfg *config.FilterConfig, rel *release.Release) bool {
	if cfg.MinAgeHours <= 0 && cfg.MaxAgeHours <= 0 {
		return true
	}
	if rel == nil || rel.PubDate == "" {
		return true
	}
	pubTime, err := time.Parse(time.RFC1123Z, rel.PubDate)
	if err != nil {
		pubTime, err = time.Parse(time.RFC1123, rel.PubDate)
	}
	if err != nil {
		return true
	}
	ageHours := time.Since(pubTime).Hours()
	if cfg.MinAgeHours > 0 && ageHours < cfg.MinAgeHours {
		return false
	}
	if cfg.MaxAgeHours > 0 && ageHours > cfg.MaxAgeHours {
		return false
	}
	return true
}

func checkKeywordsExcluded(cfg *config.FilterConfig, rel *release.Release) bool {
	if len(cfg.KeywordsExcluded) == 0 || rel == nil {
		return false
	}
	titleLower := strings.ToLower(rel.Title)
	for _, kw := range cfg.KeywordsExcluded {
		if kw != "" && strings.Contains(titleLower, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

func checkKeywordsRequired(cfg *config.FilterConfig, rel *release.Release) bool {
	if len(cfg.KeywordsRequired) == 0 || rel == nil {
		return false
	}
	titleLower := strings.ToLower(rel.Title)
	for _, kw := range cfg.KeywordsRequired {
		if kw != "" && strings.Contains(titleLower, strings.ToLower(kw)) {
			return false
		}
	}
	return true
}

func checkRegexExcluded(compiledExcluded []*regexp.Regexp, rel *release.Release) bool {
	if len(compiledExcluded) == 0 || rel == nil {
		return false
	}
	for _, re := range compiledExcluded {
		if re.MatchString(rel.Title) {
			return true
		}
	}
	return false
}

func checkRegexRequired(compiledRequired []*regexp.Regexp, rel *release.Release) bool {
	if len(compiledRequired) == 0 || rel == nil {
		return false
	}
	for _, re := range compiledRequired {
		if re.MatchString(rel.Title) {
			return false
		}
	}
	return true
}

func checkSizeWithResolution(cfg *config.FilterConfig, rel *release.Release, p *parser.ParsedRelease) bool {
	if rel == nil || rel.Size <= 0 {
		return false
	}
	sizeGB := float64(rel.Size) / (1024 * 1024 * 1024)

	if len(cfg.SizePerResolution) > 0 && p != nil {
		group := pttoptions.NormalizeResolutionToGroup(p.Resolution)
		if sr, ok := cfg.SizePerResolution[group]; ok {
			if sr.MinGB > 0 && sizeGB < sr.MinGB {
				return false
			}
			if sr.MaxGB > 0 && sizeGB > sr.MaxGB {
				return false
			}
			return true
		}
	}

	if cfg.MinSizeGB > 0 && sizeGB < cfg.MinSizeGB {
		return false
	}
	if cfg.MaxSizeGB > 0 && sizeGB > cfg.MaxSizeGB {
		return false
	}
	return true
}

func checkBitrate(cfg *config.FilterConfig, rel *release.Release) bool {
	if cfg.MinBitrateKbps <= 0 && cfg.MaxBitrateKbps <= 0 {
		return true
	}
	if rel == nil || rel.Duration <= 0 || rel.Size <= 0 {
		return true
	}
	bitrateKbps := float64(rel.Size*8) / (rel.Duration * 1000)
	if cfg.MinBitrateKbps > 0 && bitrateKbps < cfg.MinBitrateKbps {
		return false
	}
	if cfg.MaxBitrateKbps > 0 && bitrateKbps > cfg.MaxBitrateKbps {
		return false
	}
	return true
}
