package triage

import (
	"strings"

	"streamnzb/pkg/core/config"
	"streamnzb/pkg/core/config/pttoptions"
	"streamnzb/pkg/release"
	"streamnzb/pkg/search/parser"
)

// matchSingle returns true if value matches any in list (case-insensitive, contains or equal).
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

// matchMulti returns true if any of values matches any in list.
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

// matchGroup returns true if the release group equals any entry in list (case-insensitive).
// Whole-word only: "CiNE" does not match "CiNEPHiLES", "E" does not match "EPSiLON".
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

func checkQuality(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	if len(cfg.QualityAvoid) > 0 && matchSingle(p.Quality, cfg.QualityAvoid) {
		return false
	}
	if len(cfg.QualityInclude) > 0 && !matchSingle(p.Quality, cfg.QualityInclude) {
		return false
	}
	return true
}

func checkResolution(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	if len(cfg.ResolutionAvoid) > 0 && matchResolution(p, cfg.ResolutionAvoid) {
		return false
	}
	if len(cfg.ResolutionInclude) > 0 && !matchResolution(p, cfg.ResolutionInclude) {
		return false
	}
	return true
}

func checkCodec(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	if len(cfg.CodecAvoid) > 0 && p.Codec != "" && matchSingle(p.Codec, cfg.CodecAvoid) {
		return false
	}
	if len(cfg.CodecInclude) > 0 {
		if p.Codec == "" {
			return false
		}
		if !matchSingle(p.Codec, cfg.CodecInclude) {
			return false
		}
	}
	return true
}

func checkAudio(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	if len(cfg.AudioAvoid) > 0 && matchMulti(p.Audio, cfg.AudioAvoid) {
		return false
	}
	if len(cfg.AudioInclude) > 0 && !matchMulti(p.Audio, cfg.AudioInclude) {
		return false
	}
	return true
}

func checkChannels(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	if len(cfg.ChannelsAvoid) > 0 && matchMulti(p.Channels, cfg.ChannelsAvoid) {
		return false
	}
	if len(cfg.ChannelsInclude) > 0 && !matchMulti(p.Channels, cfg.ChannelsInclude) {
		return false
	}
	return true
}

func checkHDR(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	if len(cfg.HDRAvoid) > 0 && matchVisualTags(p, cfg.HDRAvoid) {
		return false
	}
	if len(cfg.HDRInclude) > 0 && !matchVisualTags(p, cfg.HDRInclude) {
		return false
	}
	return true
}

func checkBitDepth(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	if len(cfg.BitDepthAvoid) > 0 && p.BitDepth != "" && matchSingle(p.BitDepth, cfg.BitDepthAvoid) {
		return false
	}
	if len(cfg.BitDepthInclude) > 0 && p.BitDepth != "" && !matchSingle(p.BitDepth, cfg.BitDepthInclude) {
		return false
	}
	return true
}

func checkContainer(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	if len(cfg.ContainerAvoid) > 0 && matchSingle(p.Container, cfg.ContainerAvoid) {
		return false
	}
	if len(cfg.ContainerInclude) > 0 && !matchSingle(p.Container, cfg.ContainerInclude) {
		return false
	}
	return true
}

func checkEdition(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	if len(cfg.EditionAvoid) > 0 && matchSingle(p.Edition, cfg.EditionAvoid) {
		return false
	}
	if len(cfg.EditionInclude) > 0 && !matchSingle(p.Edition, cfg.EditionInclude) {
		return false
	}
	return true
}

func checkThreeD(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	if len(cfg.ThreeDAvoid) > 0 && p.ThreeD != "" && matchSingle(p.ThreeD, cfg.ThreeDAvoid) {
		return false
	}
	if len(cfg.ThreeDInclude) > 0 && p.ThreeD != "" && !matchSingle(p.ThreeD, cfg.ThreeDInclude) {
		return false
	}
	return true
}

func checkNetwork(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	if len(cfg.NetworkAvoid) > 0 && matchSingle(p.Network, cfg.NetworkAvoid) {
		return false
	}
	if len(cfg.NetworkInclude) > 0 && !matchSingle(p.Network, cfg.NetworkInclude) {
		return false
	}
	return true
}

func checkRegion(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	if len(cfg.RegionAvoid) > 0 && matchSingle(p.Region, cfg.RegionAvoid) {
		return false
	}
	if len(cfg.RegionInclude) > 0 && !matchSingle(p.Region, cfg.RegionInclude) {
		return false
	}
	return true
}

func checkGroup(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	if len(cfg.GroupAvoid) > 0 && matchGroup(p.Group, cfg.GroupAvoid) {
		return false
	}
	if len(cfg.GroupInclude) > 0 && !matchGroup(p.Group, cfg.GroupInclude) {
		return false
	}
	return true
}

func checkBooleans(cfg *config.FilterConfig, p *parser.ParsedRelease) bool {
	if cfg.DubbedAvoid != nil && *cfg.DubbedAvoid && p.Dubbed {
		return false
	}
	if cfg.HardcodedAvoid != nil && *cfg.HardcodedAvoid && p.Hardcoded {
		return false
	}
	if cfg.ProperInclude != nil && *cfg.ProperInclude && !p.Proper {
		return false
	}
	if cfg.RepackInclude != nil && *cfg.RepackInclude && !p.Repack {
		return false
	}
	if cfg.RepackAvoid != nil && *cfg.RepackAvoid && p.Repack {
		return false
	}
	if cfg.ExtendedInclude != nil && *cfg.ExtendedInclude && !p.Extended {
		return false
	}
	if cfg.UnratedInclude != nil && *cfg.UnratedInclude && !p.Unrated {
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

func checkLanguages(cfg *config.FilterConfig, p *parser.ParsedRelease, rel *release.Release) bool {
	languages := mergeReleaseLanguages(p.Languages, nil)
	if rel != nil && len(rel.Languages) > 0 {
		languages = mergeReleaseLanguages(p.Languages, rel.Languages)
	}
	if len(cfg.LanguagesAvoid) > 0 {
		for _, lang := range languages {
			for _, avoid := range cfg.LanguagesAvoid {
				if languageMatches(avoid, lang) {
					return false
				}
			}
		}
	}
	if len(cfg.LanguagesInclude) > 0 {
		if len(languages) == 0 {
			return false
		}
		if !matchLanguages(languages, cfg.LanguagesInclude) {
			return false
		}
	}
	return true
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
