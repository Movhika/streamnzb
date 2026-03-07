package stremio

import (
	"reflect"
	"testing"
)

func TestBuildSeriesQueriesReturnsGenericShowName(t *testing.T) {
	got := buildSeriesQueries("Game of Thrones")
	want := []string{"Game of Thrones"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildSeriesQueries() = %#v, want %#v", got, want)
	}
}

func TestBuildSeriesQueriesWithOptionsCanIncludeYear(t *testing.T) {
	got := buildSeriesQueriesWithOptions("Game of Thrones", "2011", true)
	want := []string{"Game of Thrones 2011"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildSeriesQueriesWithOptions() = %#v, want %#v", got, want)
	}
}

func TestBuildSeriesQueriesWithOptionsCanOmitYear(t *testing.T) {
	got := buildSeriesQueriesWithOptions("Game of Thrones", "2011", false)
	want := []string{"Game of Thrones"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildSeriesQueriesWithOptions() = %#v, want %#v", got, want)
	}
}

func TestShouldIncludeMetadataYearInIndexerQuery(t *testing.T) {
	tests := []struct {
		name        string
		indexerType string
		includeYear bool
		want        bool
	}{
		{name: "easynews keeps metadata year", indexerType: "easynews", includeYear: true, want: true},
		{name: "newznab omits metadata year", indexerType: "newznab", includeYear: true, want: false},
		{name: "aggregator omits metadata year", indexerType: "aggregator", includeYear: true, want: false},
		{name: "blank type defaults to newznab behavior", indexerType: "", includeYear: true, want: false},
		{name: "explicit opt out stays off", indexerType: "easynews", includeYear: false, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldIncludeMetadataYearInIndexerQuery(tt.indexerType, tt.includeYear); got != tt.want {
				t.Fatalf("shouldIncludeMetadataYearInIndexerQuery(%q, %v) = %v, want %v", tt.indexerType, tt.includeYear, got, tt.want)
			}
		})
	}
}
