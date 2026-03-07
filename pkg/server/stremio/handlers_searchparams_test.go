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
