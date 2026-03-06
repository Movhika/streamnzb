package config

import "testing"

func TestMergeIndexerSearchDefaultsSeriesSeasonAndCompleteSearchOn(t *testing.T) {
	merged := MergeIndexerSearch(&IndexerConfig{}, nil, &Config{})
	if merged.EnableSeriesSeasonSearch == nil || !*merged.EnableSeriesSeasonSearch {
		t.Fatalf("expected EnableSeriesSeasonSearch default true, got %#v", merged.EnableSeriesSeasonSearch)
	}
	if merged.EnableSeriesCompleteSearch == nil || !*merged.EnableSeriesCompleteSearch {
		t.Fatalf("expected EnableSeriesCompleteSearch default true, got %#v", merged.EnableSeriesCompleteSearch)
	}
}

func TestMergeIndexerSearchLegacySeriesPackSearchAppliesToSeasonAndComplete(t *testing.T) {
	merged := MergeIndexerSearch(
		&IndexerConfig{EnableSeriesPackSearch: ptrBool(false)},
		nil,
		&Config{},
	)
	if merged.EnableSeriesSeasonSearch == nil || *merged.EnableSeriesSeasonSearch {
		t.Fatalf("expected legacy pack search setting to disable season search, got %#v", merged.EnableSeriesSeasonSearch)
	}
	if merged.EnableSeriesCompleteSearch == nil || *merged.EnableSeriesCompleteSearch {
		t.Fatalf("expected legacy pack search setting to disable complete search, got %#v", merged.EnableSeriesCompleteSearch)
	}
}

func TestMergeIndexerSearchExplicitSeriesSearchOverridesWin(t *testing.T) {
	merged := MergeIndexerSearch(
		&IndexerConfig{
			EnableSeriesPackSearch:     ptrBool(false),
			EnableSeriesSeasonSearch:   ptrBool(true),
			EnableSeriesCompleteSearch: ptrBool(false),
		},
		&IndexerSearchConfig{
			EnableSeriesSeasonSearch:   ptrBool(false),
			EnableSeriesCompleteSearch: ptrBool(true),
		},
		&Config{},
	)
	if merged.EnableSeriesSeasonSearch == nil || *merged.EnableSeriesSeasonSearch {
		t.Fatalf("expected explicit season override to win, got %#v", merged.EnableSeriesSeasonSearch)
	}
	if merged.EnableSeriesCompleteSearch == nil || !*merged.EnableSeriesCompleteSearch {
		t.Fatalf("expected explicit complete override to win, got %#v", merged.EnableSeriesCompleteSearch)
	}
}
