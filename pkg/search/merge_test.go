package search

import (
	"testing"
)

func TestNormalizedTitleMatches(t *testing.T) {
	tests := []struct {
		expect   string
		gotTitle string
		want     bool
	}{
		{"Law & Order", "Law and Order", true},
		{"Law & Order", "Law and Order SVU", true},
		{"Star Trek: Starfleet Academy", "Star.Trek.Starfleet.Academy.S01E01", true},
		{"Star Trek: Starfleet Academy", "Starfleet Academy S01E01", false},
		{"The Walking Dead", "The Walking Dead S06E07", true},
		{"Some Show", "Other Show", false},
		{"Law and Order", "Law & Order", true},
		{"Show 2024", "Show 2024 1080p", true},
	}
	for _, tt := range tests {
		got := normalizedTitleMatches(tt.expect, tt.gotTitle)
		if got != tt.want {
			t.Errorf("normalizedTitleMatches(%q, %q) = %v, want %v", tt.expect, tt.gotTitle, got, tt.want)
		}
	}
}
