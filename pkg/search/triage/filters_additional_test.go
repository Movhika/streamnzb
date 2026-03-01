package triage

import (
	"strings"
	"testing"

	"streamnzb/pkg/core/config"
	"streamnzb/pkg/release"
	"streamnzb/pkg/search/parser"
)

// Test Quality Filtering (3-tier: required / excluded)
func TestCheckQuality(t *testing.T) {
	tests := []struct {
		name       string
		cfg        *config.FilterConfig
		parsed     *parser.ParsedRelease
		shouldPass bool
	}{
		{
			name: "WEB-DL passes when required",
			cfg: &config.FilterConfig{
				QualityRequired: []string{"WEB-DL", "BluRay"},
			},
			parsed:     &parser.ParsedRelease{Quality: "WEB-DL"},
			shouldPass: true,
		},
		{
			name: "CAM rejected when not required",
			cfg: &config.FilterConfig{
				QualityRequired: []string{"WEB-DL", "BluRay"},
			},
			parsed:     &parser.ParsedRelease{Quality: "CAM"},
			shouldPass: false,
		},
		{
			name: "Excluded quality rejected",
			cfg: &config.FilterConfig{
				QualityExcluded: []string{"CAM", "TS"},
			},
			parsed:     &parser.ParsedRelease{Quality: "CAM"},
			shouldPass: false,
		},
		{
			name: "Non-excluded quality passes",
			cfg: &config.FilterConfig{
				QualityExcluded: []string{"CAM"},
			},
			parsed:     &parser.ParsedRelease{Quality: "WEB-DL"},
			shouldPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			excluded := checkQualityExcluded(tt.cfg, tt.parsed)
			required := checkQualityRequired(tt.cfg, tt.parsed)
			result := !excluded && !required
			if result != tt.shouldPass {
				t.Errorf("quality filter = %v, want %v", result, tt.shouldPass)
			}
		})
	}
}

// Test Audio Filtering (3-tier: required / excluded)
func TestCheckAudio(t *testing.T) {
	tests := []struct {
		name       string
		cfg        *config.FilterConfig
		parsed     *parser.ParsedRelease
		shouldPass bool
	}{
		{
			name: "Required audio present passes",
			cfg: &config.FilterConfig{
				AudioRequired: []string{"Atmos"},
			},
			parsed:     &parser.ParsedRelease{Audio: []string{"DDP5.1", "Atmos"}},
			shouldPass: true,
		},
		{
			name: "Required audio missing rejected",
			cfg: &config.FilterConfig{
				AudioRequired: []string{"Atmos"},
			},
			parsed:     &parser.ParsedRelease{Audio: []string{"DDP5.1"}},
			shouldPass: false,
		},
		{
			name: "Allowed audio passes",
			cfg: &config.FilterConfig{
				AudioRequired: []string{"DDP", "TrueHD"},
			},
			parsed:     &parser.ParsedRelease{Audio: []string{"DDP5.1"}},
			shouldPass: true,
		},
		{
			name: "Non-allowed audio rejected",
			cfg: &config.FilterConfig{
				AudioRequired: []string{"DDP", "TrueHD"},
			},
			parsed:     &parser.ParsedRelease{Audio: []string{"AAC"}},
			shouldPass: false,
		},
		{
			name: "Min channels 5.1 passes",
			cfg: &config.FilterConfig{
				ChannelsRequired: []string{"5.1", "7.1"},
			},
			parsed:     &parser.ParsedRelease{Channels: []string{"7.1"}},
			shouldPass: true,
		},
		{
			name: "Below min channels rejected",
			cfg: &config.FilterConfig{
				ChannelsRequired: []string{"5.1", "7.1"},
			},
			parsed:     &parser.ParsedRelease{Channels: []string{"2.0"}},
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result bool
			if strings.Contains(tt.name, "channel") {
				excluded := checkChannelsExcluded(tt.cfg, tt.parsed)
				required := checkChannelsRequired(tt.cfg, tt.parsed)
				result = !excluded && !required
			} else {
				excluded := checkAudioExcluded(tt.cfg, tt.parsed)
				required := checkAudioRequired(tt.cfg, tt.parsed)
				result = !excluded && !required
			}
			if result != tt.shouldPass {
				t.Errorf("check = %v, want %v", result, tt.shouldPass)
			}
		})
	}
}

// Test HDR Filtering (3-tier: required / excluded)
func TestCheckHDR(t *testing.T) {
	tests := []struct {
		name       string
		cfg        *config.FilterConfig
		parsed     *parser.ParsedRelease
		shouldPass bool
	}{
		{
			name: "HDR required and present passes",
			cfg: &config.FilterConfig{
				HDRRequired: []string{"HDR", "HDR10", "HDR10+", "DV", "3D"},
			},
			parsed:     &parser.ParsedRelease{HDR: []string{"HDR10"}},
			shouldPass: true,
		},
		{
			name: "HDR required but missing rejected",
			cfg: &config.FilterConfig{
				HDRRequired: []string{"HDR", "HDR10", "HDR10+", "DV", "3D"},
			},
			parsed:     &parser.ParsedRelease{HDR: []string{}},
			shouldPass: false,
		},
		{
			name: "Excluded HDR type rejected",
			cfg: &config.FilterConfig{
				HDRExcluded: []string{"DV"},
			},
			parsed:     &parser.ParsedRelease{HDR: []string{"DV", "HDR10"}},
			shouldPass: false,
		},
		{
			name: "SDR excluded and present rejected",
			cfg: &config.FilterConfig{
				HDRExcluded: []string{"SDR"},
			},
			parsed:     &parser.ParsedRelease{HDR: []string{"SDR"}},
			shouldPass: false,
		},
		{
			name: "Required HDR type passes",
			cfg: &config.FilterConfig{
				HDRRequired: []string{"HDR10", "HDR10+"},
			},
			parsed:     &parser.ParsedRelease{HDR: []string{"HDR10"}},
			shouldPass: true,
		},
		{
			name: "Non-required HDR type rejected",
			cfg: &config.FilterConfig{
				HDRRequired: []string{"HDR10"},
			},
			parsed:     &parser.ParsedRelease{HDR: []string{"DV"}},
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			excluded := checkHDRExcluded(tt.cfg, tt.parsed)
			required := checkHDRRequired(tt.cfg, tt.parsed)
			result := !excluded && !required
			if result != tt.shouldPass {
				t.Errorf("HDR filter = %v, want %v", result, tt.shouldPass)
			}
		})
	}
}

// Test Language Filtering (3-tier: required / excluded)
func TestCheckLanguages(t *testing.T) {
	tests := []struct {
		name       string
		cfg        *config.FilterConfig
		parsed     *parser.ParsedRelease
		rel        *release.Release
		shouldPass bool
	}{
		{
			name: "Required language present passes",
			cfg: &config.FilterConfig{
				LanguagesRequired: []string{"English"},
			},
			parsed:     &parser.ParsedRelease{Languages: []string{"English"}},
			shouldPass: true,
		},
		{
			name: "Required language missing rejected",
			cfg: &config.FilterConfig{
				LanguagesRequired: []string{"English"},
			},
			parsed:     &parser.ParsedRelease{Languages: []string{"German"}},
			shouldPass: false,
		},
		{
			name: "Allowed language passes",
			cfg: &config.FilterConfig{
				LanguagesRequired: []string{"English", "German"},
			},
			parsed:     &parser.ParsedRelease{Languages: []string{"English"}},
			shouldPass: true,
		},
		{
			name: "Non-allowed language rejected",
			cfg: &config.FilterConfig{
				LanguagesRequired: []string{"English"},
			},
			parsed:     &parser.ParsedRelease{Languages: []string{"French"}},
			shouldPass: false,
		},
		{
			name: "Config 'en' matches parser 'en'",
			cfg: &config.FilterConfig{
				LanguagesRequired: []string{"en"},
			},
			parsed:     &parser.ParsedRelease{Languages: []string{"en"}},
			shouldPass: true,
		},
		{
			name: "Config 'en' matches parser 'English' via normalization",
			cfg: &config.FilterConfig{
				LanguagesRequired: []string{"en"},
			},
			parsed:     &parser.ParsedRelease{Languages: []string{"English"}},
			shouldPass: true,
		},
		{
			name: "Config 'multi' matches 'multi subs'",
			cfg: &config.FilterConfig{
				LanguagesRequired: []string{"multi"},
			},
			parsed:     &parser.ParsedRelease{Languages: []string{"multi subs"}},
			shouldPass: true,
		},
		{
			name: "Indexer language (English) used when parser has none",
			cfg: &config.FilterConfig{
				LanguagesRequired: []string{"en"},
			},
			parsed:     &parser.ParsedRelease{Languages: []string{}},
			rel:        &release.Release{Languages: []string{"English"}},
			shouldPass: true,
		},
		{
			name: "Release with no language rejected when required list is set",
			cfg: &config.FilterConfig{
				LanguagesRequired: []string{"fi", "fin"},
			},
			parsed:     &parser.ParsedRelease{Languages: []string{}},
			rel:        nil,
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			excluded := checkLanguagesExcluded(tt.cfg, tt.parsed, tt.rel)
			required := checkLanguagesRequired(tt.cfg, tt.parsed, tt.rel)
			result := !excluded && !required
			if result != tt.shouldPass {
				t.Errorf("languages filter = %v, want %v", result, tt.shouldPass)
			}
		})
	}
}

// Test Other Filters (CAM, Proper, Repack)
func TestCheckOther(t *testing.T) {
	ptrBool := func(b bool) *bool { return &b }
	tests := []struct {
		name       string
		cfg        *config.FilterConfig
		parsed     *parser.ParsedRelease
		shouldPass bool
	}{
		{
			name: "CAM excluded and present rejected",
			cfg: &config.FilterConfig{
				QualityExcluded: []string{"CAM", "TeleSync", "TeleCine", "SCR"},
			},
			parsed:     &parser.ParsedRelease{Quality: "CAM"},
			shouldPass: false,
		},
		{
			name: "TS excluded and present rejected",
			cfg: &config.FilterConfig{
				QualityExcluded: []string{"CAM", "TeleSync", "TeleCine", "SCR"},
			},
			parsed:     &parser.ParsedRelease{Quality: "TeleSync"},
			shouldPass: false,
		},
		{
			name: "Proper required and present passes",
			cfg: &config.FilterConfig{
				ProperRequired: ptrBool(true),
			},
			parsed:     &parser.ParsedRelease{Proper: true},
			shouldPass: true,
		},
		{
			name: "Proper required but missing rejected",
			cfg: &config.FilterConfig{
				ProperRequired: ptrBool(true),
			},
			parsed:     &parser.ParsedRelease{Proper: false},
			shouldPass: false,
		},
		{
			name: "Repack excluded and present rejected",
			cfg: &config.FilterConfig{
				RepackExcluded: ptrBool(true),
			},
			parsed:     &parser.ParsedRelease{Repack: true},
			shouldPass: false,
		},
		{
			name: "Repack required and present passes",
			cfg: &config.FilterConfig{
				RepackRequired: ptrBool(true),
			},
			parsed:     &parser.ParsedRelease{Repack: true},
			shouldPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qualityExcluded := checkQualityExcluded(tt.cfg, tt.parsed)
			qualityRequired := checkQualityRequired(tt.cfg, tt.parsed)
			booleans := checkBooleans(tt.cfg, tt.parsed)
			result := !qualityExcluded && !qualityRequired && booleans
			if result != tt.shouldPass {
				t.Errorf("checkBooleans+quality filter = %v, want %v", result, tt.shouldPass)
			}
		})
	}
}

// Test Group Filtering (3-tier: required / excluded)
func TestCheckGroup(t *testing.T) {
	tests := []struct {
		name       string
		cfg        *config.FilterConfig
		parsed     *parser.ParsedRelease
		shouldPass bool
	}{
		{
			name: "Excluded group rejected",
			cfg: &config.FilterConfig{
				GroupExcluded: []string{"YIFY", "RARBG"},
			},
			parsed:     &parser.ParsedRelease{Group: "YIFY"},
			shouldPass: false,
		},
		{
			name: "Non-excluded group passes",
			cfg: &config.FilterConfig{
				GroupExcluded: []string{"YIFY"},
			},
			parsed:     &parser.ParsedRelease{Group: "FLUX"},
			shouldPass: true,
		},
		{
			name: "Empty group passes",
			cfg: &config.FilterConfig{
				GroupExcluded: []string{"YIFY"},
			},
			parsed:     &parser.ParsedRelease{Group: ""},
			shouldPass: true,
		},
		{
			name: "Exclude CiNE does not match CiNEPHiLES (whole-word)",
			cfg: &config.FilterConfig{
				GroupExcluded: []string{"CiNE"},
			},
			parsed:     &parser.ParsedRelease{Group: "CiNEPHiLES"},
			shouldPass: true,
		},
		{
			name: "Exclude E does not match EPSiLON (whole-word)",
			cfg: &config.FilterConfig{
				GroupExcluded: []string{"E"},
			},
			parsed:     &parser.ParsedRelease{Group: "EPSiLON"},
			shouldPass: true,
		},
		{
			name: "Exclude CiNE does match CiNE (exact)",
			cfg: &config.FilterConfig{
				GroupExcluded: []string{"CiNE"},
			},
			parsed:     &parser.ParsedRelease{Group: "CiNE"},
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			excluded := checkGroupExcluded(tt.cfg, tt.parsed)
			required := checkGroupRequired(tt.cfg, tt.parsed)
			result := !excluded && !required
			if result != tt.shouldPass {
				t.Errorf("group filter = %v, want %v", result, tt.shouldPass)
			}
		})
	}
}
