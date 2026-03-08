package api

import (
	"testing"

	"streamnzb/pkg/core/config"
)

func TestValidateConfigRejectsUnresolvedProwlarrIndexerPlaceholder(t *testing.T) {
	enabled := true
	s := &Server{}

	errs := s.validateConfig(&config.Config{
		KeepLogFiles: 9,
		Indexers: []config.IndexerConfig{{
			Enabled: &enabled,
			Name:    "Prowlarr",
			URL:     "http://[::1",
			APIPath: "{indexer_id}/api",
			Type:    "aggregator",
		}},
	})

	if got := errs["indexers.0.api_path"]; got == "" {
		t.Fatalf("expected api_path validation error, got %#v", errs)
	}
	if got := errs["indexers.0.url"]; got != "" {
		t.Fatalf("expected placeholder validation to stop ping before url validation, got url error %q", got)
	}
}
