package triage

import (
	"strings"
	"testing"

	"streamnzb/pkg/core/config"
	"streamnzb/pkg/release"
	"streamnzb/pkg/search/parser"
)

// Test Quality Filtering
func TestCheckQuality(t *testing.T) {
	tests := []struct {
		name       string
		cfg        *config.FilterConfig
		parsed     *parser.ParsedRelease
		shouldPass bool
	}{
		{
			name: "WEB-DL passes when allowed",
			cfg: &config.FilterConfig{
				QualityInclude: []string{"WEB-DL", "BluRay"},
			},
			parsed: &parser.ParsedRelease{
				Quality: "WEB-DL",
			},
			shouldPass: true,
		},
		{
			name: "CAM rejected when not allowed",
			cfg: &config.FilterConfig{
				QualityInclude: []string{"WEB-DL", "BluRay"},
			},
			parsed: &parser.ParsedRelease{
				Quality: "CAM",
			},
			shouldPass: false,
		},
		{
			name: "Blocked quality rejected",
			cfg: &config.FilterConfig{
				QualityAvoid: []string{"CAM", "TS"},
			},
			parsed: &parser.ParsedRelease{
				Quality: "CAM",
			},
			shouldPass: false,
		},
		{
			name: "Non-blocked quality passes",
			cfg: &config.FilterConfig{
				QualityAvoid: []string{"CAM"},
			},
			parsed: &parser.ParsedRelease{
				Quality: "WEB-DL",
			},
			shouldPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkQuality(tt.cfg, tt.parsed)
			if result != tt.shouldPass {
				t.Errorf("checkQuality() = %v, want %v", result, tt.shouldPass)
			}
		})
	}
}

// Test Audio Filtering
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
				AudioInclude: []string{"Atmos"},
			},
			parsed: &parser.ParsedRelease{
				Audio: []string{"DDP5.1", "Atmos"},
			},
			shouldPass: true,
		},
		{
			name: "Required audio missing rejected",
			cfg: &config.FilterConfig{
				AudioInclude: []string{"Atmos"},
			},
			parsed: &parser.ParsedRelease{
				Audio: []string{"DDP5.1"},
			},
			shouldPass: false,
		},
		{
			name: "Allowed audio passes",
			cfg: &config.FilterConfig{
				AudioInclude: []string{"DDP", "TrueHD"},
			},
			parsed: &parser.ParsedRelease{
				Audio: []string{"DDP5.1"},
			},
			shouldPass: true,
		},
		{
			name: "Non-allowed audio rejected",
			cfg: &config.FilterConfig{
				AudioInclude: []string{"DDP", "TrueHD"},
			},
			parsed: &parser.ParsedRelease{
				Audio: []string{"AAC"},
			},
			shouldPass: false,
		},
		{
			name: "Min channels 5.1 passes",
			cfg: &config.FilterConfig{
				ChannelsInclude: []string{"5.1", "7.1"},
			},
			parsed: &parser.ParsedRelease{
				Channels: []string{"7.1"},
			},
			shouldPass: true,
		},
		{
			name: "Below min channels rejected",
			cfg: &config.FilterConfig{
				ChannelsInclude: []string{"5.1", "7.1"},
			},
			parsed: &parser.ParsedRelease{
				Channels: []string{"2.0"},
			},
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result bool
			if strings.Contains(tt.name, "channel") {
				result = checkChannels(tt.cfg, tt.parsed)
			} else {
				result = checkAudio(tt.cfg, tt.parsed)
			}
			if result != tt.shouldPass {
				t.Errorf("check = %v, want %v", result, tt.shouldPass)
			}
		})
	}
}

// Test HDR Filtering
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
				HDRInclude: []string{"HDR", "HDR10", "HDR10+", "DV", "3D"},
			},
			parsed: &parser.ParsedRelease{
				HDR: []string{"HDR10"},
			},
			shouldPass: true,
		},
		{
			name: "HDR required but missing rejected",
			cfg: &config.FilterConfig{
				HDRInclude: []string{"HDR", "HDR10", "HDR10+", "DV", "3D"},
			},
			parsed: &parser.ParsedRelease{
				HDR: []string{},
			},
			shouldPass: false,
		},
		{
			name: "Blocked HDR type rejected",
			cfg: &config.FilterConfig{
				HDRAvoid: []string{"DV"},
			},
			parsed: &parser.ParsedRelease{
				HDR: []string{"DV", "HDR10"},
			},
			shouldPass: false,
		},
		{
			name: "SDR blocked and present rejected",
			cfg: &config.FilterConfig{
				HDRAvoid: []string{"SDR"},
			},
			parsed: &parser.ParsedRelease{
				HDR: []string{"SDR"},
			},
			shouldPass: false,
		},
		{
			name: "Allowed HDR type passes",
			cfg: &config.FilterConfig{
				HDRInclude: []string{"HDR10", "HDR10+"},
			},
			parsed: &parser.ParsedRelease{
				HDR: []string{"HDR10"},
			},
			shouldPass: true,
		},
		{
			name: "Non-allowed HDR type rejected",
			cfg: &config.FilterConfig{
				HDRInclude: []string{"HDR10"},
			},
			parsed: &parser.ParsedRelease{
				HDR: []string{"DV"},
			},
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkHDR(tt.cfg, tt.parsed)
			if result != tt.shouldPass {
				t.Errorf("checkHDR() = %v, want %v", result, tt.shouldPass)
			}
		})
	}
}

// Test Language Filtering
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
				LanguagesInclude: []string{"English"},
			},
			parsed: &parser.ParsedRelease{
				Languages: []string{"English"},
			},
			shouldPass: true,
		},
		{
			name: "Required language missing rejected",
			cfg: &config.FilterConfig{
				LanguagesInclude: []string{"English"},
			},
			parsed: &parser.ParsedRelease{
				Languages: []string{"German"},
			},
			shouldPass: false,
		},
		{
			name: "Allowed language passes",
			cfg: &config.FilterConfig{
				LanguagesInclude: []string{"English", "German"},
			},
			parsed: &parser.ParsedRelease{
				Languages: []string{"English"},
			},
			shouldPass: true,
		},
		{
			name: "Non-allowed language rejected",
			cfg: &config.FilterConfig{
				LanguagesInclude: []string{"English"},
			},
			parsed: &parser.ParsedRelease{
				Languages: []string{"French"},
			},
			shouldPass: false,
		},
		{
			name: "Config 'en' matches parser 'en'",
			cfg: &config.FilterConfig{
				LanguagesInclude: []string{"en"},
			},
			parsed: &parser.ParsedRelease{
				Languages: []string{"en"},
			},
			shouldPass: true,
		},
		{
			name: "Config 'en' matches parser 'English' via normalization",
			cfg: &config.FilterConfig{
				LanguagesInclude: []string{"en"},
			},
			parsed: &parser.ParsedRelease{
				Languages: []string{"English"},
			},
			shouldPass: true,
		},
		{
			name: "Config 'multi' matches 'multi subs'",
			cfg: &config.FilterConfig{
				LanguagesInclude: []string{"multi"},
			},
			parsed: &parser.ParsedRelease{
				Languages: []string{"multi subs"},
			},
			shouldPass: true,
		},
		{
			name: "Indexer language (English) used when parser has none",
			cfg: &config.FilterConfig{
				LanguagesInclude: []string{"en"},
			},
			parsed: &parser.ParsedRelease{
				Languages: []string{},
			},
			rel:        &release.Release{Languages: []string{"English"}},
			shouldPass: true,
		},
		{
			name: "Release with no language rejected when allowed list is set",
			cfg: &config.FilterConfig{
				LanguagesInclude: []string{"fi", "fin"},
			},
			parsed: &parser.ParsedRelease{
				Languages: []string{},
			},
			rel:        nil,
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkLanguages(tt.cfg, tt.parsed, tt.rel)
			if result != tt.shouldPass {
				t.Errorf("checkLanguages() = %v, want %v", result, tt.shouldPass)
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
			name: "CAM blocked and present rejected",
			cfg: &config.FilterConfig{
				QualityAvoid: []string{"CAM", "TeleSync", "TeleCine", "SCR"},
			},
			parsed: &parser.ParsedRelease{
				Quality: "CAM",
			},
			shouldPass: false,
		},
		{
			name: "TS blocked and present rejected",
			cfg: &config.FilterConfig{
				QualityAvoid: []string{"CAM", "TeleSync", "TeleCine", "SCR"},
			},
			parsed: &parser.ParsedRelease{
				Quality: "TeleSync",
			},
			shouldPass: false,
		},
		{
			name: "Proper required and present passes",
			cfg: &config.FilterConfig{
				ProperInclude: ptrBool(true),
			},
			parsed: &parser.ParsedRelease{
				Proper: true,
			},
			shouldPass: true,
		},
		{
			name: "Proper required but missing rejected",
			cfg: &config.FilterConfig{
				ProperInclude: ptrBool(true),
			},
			parsed: &parser.ParsedRelease{
				Proper: false,
			},
			shouldPass: false,
		},
		{
			name: "Repack not allowed and present rejected",
			cfg: &config.FilterConfig{
				RepackAvoid: ptrBool(true),
			},
			parsed: &parser.ParsedRelease{
				Repack: true,
			},
			shouldPass: false,
		},
		{
			name: "Repack allowed and present passes",
			cfg: &config.FilterConfig{
				RepackInclude: ptrBool(true),
			},
			parsed: &parser.ParsedRelease{
				Repack: true,
			},
			shouldPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkBooleans(tt.cfg, tt.parsed) && checkQuality(tt.cfg, tt.parsed)
			if result != tt.shouldPass {
				t.Errorf("checkBooleans+checkQuality = %v, want %v", result, tt.shouldPass)
			}
		})
	}
}

// Test Group Filtering
func TestCheckGroup(t *testing.T) {
	tests := []struct {
		name       string
		cfg        *config.FilterConfig
		parsed     *parser.ParsedRelease
		shouldPass bool
	}{
		{
			name: "Blocked group rejected",
			cfg: &config.FilterConfig{
				GroupAvoid: []string{"YIFY", "RARBG"},
			},
			parsed: &parser.ParsedRelease{
				Group: "YIFY",
			},
			shouldPass: false,
		},
		{
			name: "Non-blocked group passes",
			cfg: &config.FilterConfig{
				GroupAvoid: []string{"YIFY"},
			},
			parsed: &parser.ParsedRelease{
				Group: "FLUX",
			},
			shouldPass: true,
		},
		{
			name: "Empty group passes",
			cfg: &config.FilterConfig{
				GroupAvoid: []string{"YIFY"},
			},
			parsed: &parser.ParsedRelease{
				Group: "",
			},
			shouldPass: true,
		},
		{
			name: "Avoid CiNE does not match CiNEPHiLES (whole-word)",
			cfg: &config.FilterConfig{
				GroupAvoid: []string{"CiNE"},
			},
			parsed: &parser.ParsedRelease{
				Group: "CiNEPHiLES",
			},
			shouldPass: true,
		},
		{
			name: "Avoid E does not match EPSiLON (whole-word)",
			cfg: &config.FilterConfig{
				GroupAvoid: []string{"E"},
			},
			parsed: &parser.ParsedRelease{
				Group: "EPSiLON",
			},
			shouldPass: true,
		},
		{
			name: "Avoid CiNE does match CiNE (exact)",
			cfg: &config.FilterConfig{
				GroupAvoid: []string{"CiNE"},
			},
			parsed: &parser.ParsedRelease{
				Group: "CiNE",
			},
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkGroup(tt.cfg, tt.parsed)
			if result != tt.shouldPass {
				t.Errorf("checkGroup() = %v, want %v", result, tt.shouldPass)
			}
		})
	}
}
