package triage

import (
	"testing"

	"streamnzb/pkg/core/config"
	"streamnzb/pkg/release"
	"streamnzb/pkg/search/parser"
)

// Test Resolution Filtering (3-tier: required / excluded)
func TestCheckResolution(t *testing.T) {
	tests := []struct {
		name       string
		cfg        *config.FilterConfig
		parsed     *parser.ParsedRelease
		shouldPass bool
	}{
		{
			name: "1080p passes with required 1080p",
			cfg: &config.FilterConfig{
				ResolutionRequired: []string{"1080p", "4k"},
			},
			parsed:     &parser.ParsedRelease{Resolution: "1080p"},
			shouldPass: true,
		},
		{
			name: "720p rejected with required only 1080p/4k",
			cfg: &config.FilterConfig{
				ResolutionRequired: []string{"1080p", "4k"},
			},
			parsed:     &parser.ParsedRelease{Resolution: "720p"},
			shouldPass: false,
		},
		{
			name: "SD/480p rejected with required only 1080p/4k",
			cfg: &config.FilterConfig{
				ResolutionRequired: []string{"1080p", "4k"},
			},
			parsed:     &parser.ParsedRelease{Resolution: "480p"},
			shouldPass: false,
		},
		{
			name: "Empty resolution rejected when required set",
			cfg: &config.FilterConfig{
				ResolutionRequired: []string{"1080p"},
			},
			parsed:     &parser.ParsedRelease{Resolution: ""},
			shouldPass: false,
		},
		{
			name:       "Empty resolution allowed when no filter set",
			cfg:        &config.FilterConfig{},
			parsed:     &parser.ParsedRelease{Resolution: ""},
			shouldPass: true,
		},
		{
			name: "4K passes with required 4k",
			cfg: &config.FilterConfig{
				ResolutionRequired: []string{"4k", "1080p"},
			},
			parsed:     &parser.ParsedRelease{Resolution: "2160p"},
			shouldPass: true,
		},
		{
			name: "4K rejected with excluded 4k",
			cfg: &config.FilterConfig{
				ResolutionExcluded: []string{"4k"},
			},
			parsed:     &parser.ParsedRelease{Resolution: "2160p"},
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			excluded := checkResolutionExcluded(tt.cfg, tt.parsed)
			required := checkResolutionRequired(tt.cfg, tt.parsed)
			result := !excluded && !required
			if result != tt.shouldPass {
				t.Errorf("resolution filter = %v, want %v", result, tt.shouldPass)
			}
		})
	}
}

// Test Codec Filtering (3-tier: required / excluded)
func TestCheckCodec(t *testing.T) {
	tests := []struct {
		name       string
		cfg        *config.FilterConfig
		parsed     *parser.ParsedRelease
		shouldPass bool
	}{
		{
			name: "H264 passes when required",
			cfg: &config.FilterConfig{
				CodecRequired: []string{"H264", "HEVC"},
			},
			parsed:     &parser.ParsedRelease{Codec: "H264"},
			shouldPass: true,
		},
		{
			name: "HEVC passes when required",
			cfg: &config.FilterConfig{
				CodecRequired: []string{"H264", "HEVC"},
			},
			parsed:     &parser.ParsedRelease{Codec: "HEVC"},
			shouldPass: true,
		},
		{
			name: "AV1 rejected when only H264/HEVC required",
			cfg: &config.FilterConfig{
				CodecRequired: []string{"H264", "HEVC"},
			},
			parsed:     &parser.ParsedRelease{Codec: "AV1"},
			shouldPass: false,
		},
		{
			name: "Empty codec rejected when required list set",
			cfg: &config.FilterConfig{
				CodecRequired: []string{"H264", "HEVC"},
			},
			parsed:     &parser.ParsedRelease{Codec: ""},
			shouldPass: false,
		},
		{
			name:       "Empty codec allowed when no filter set",
			cfg:        &config.FilterConfig{},
			parsed:     &parser.ParsedRelease{Codec: ""},
			shouldPass: true,
		},
		{
			name: "Excluded codec rejected",
			cfg: &config.FilterConfig{
				CodecExcluded: []string{"AV1"},
			},
			parsed:     &parser.ParsedRelease{Codec: "AV1"},
			shouldPass: false,
		},
		{
			name: "Non-excluded codec passes",
			cfg: &config.FilterConfig{
				CodecExcluded: []string{"AV1"},
			},
			parsed:     &parser.ParsedRelease{Codec: "H264"},
			shouldPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			excluded := checkCodecExcluded(tt.cfg, tt.parsed)
			required := checkCodecRequired(tt.cfg, tt.parsed)
			result := !excluded && !required
			if result != tt.shouldPass {
				t.Errorf("codec filter = %v, want %v", result, tt.shouldPass)
			}
		})
	}
}

// Test File Size Filtering
func TestCheckSize(t *testing.T) {
	tests := []struct {
		name       string
		cfg        *config.FilterConfig
		rel        *release.Release
		shouldPass bool
	}{
		{
			name:       "0 bytes always rejected",
			cfg:        &config.FilterConfig{},
			rel:        &release.Release{Size: 0},
			shouldPass: false,
		},
		{
			name:       "Negative size always rejected",
			cfg:        &config.FilterConfig{},
			rel:        &release.Release{Size: -1},
			shouldPass: false,
		},
		{
			name:       "Valid size passes",
			cfg:        &config.FilterConfig{},
			rel:        &release.Release{Size: 1024 * 1024 * 1024}, // 1 GB
			shouldPass: true,
		},
		{
			name:       "Too small rejected with min size",
			cfg:        &config.FilterConfig{MinSizeGB: 2.0},
			rel:        &release.Release{Size: 1024 * 1024 * 1024}, // 1 GB
			shouldPass: false,
		},
		{
			name:       "Meets min size passes",
			cfg:        &config.FilterConfig{MinSizeGB: 1.0},
			rel:        &release.Release{Size: 1024 * 1024 * 1024}, // 1 GB
			shouldPass: true,
		},
		{
			name:       "Too large rejected with max size",
			cfg:        &config.FilterConfig{MaxSizeGB: 5.0},
			rel:        &release.Release{Size: 10 * 1024 * 1024 * 1024}, // 10 GB
			shouldPass: false,
		},
		{
			name:       "Within max size passes",
			cfg:        &config.FilterConfig{MaxSizeGB: 10.0},
			rel:        &release.Release{Size: 5 * 1024 * 1024 * 1024}, // 5 GB
			shouldPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkSize(tt.cfg, tt.rel)
			if result != tt.shouldPass {
				size := int64(0)
				if tt.rel != nil {
					size = tt.rel.Size
				}
				t.Errorf("checkSize() = %v, want %v (size: %d bytes)", result, tt.shouldPass, size)
			}
		})
	}
}
