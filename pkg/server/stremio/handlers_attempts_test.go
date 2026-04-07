package stremio

import (
	"testing"

	"streamnzb/pkg/media/unpack"
	"streamnzb/pkg/session"
)

func TestRecordAttemptParamsIncludesServedFile(t *testing.T) {
	server := &Server{}
	sess := &session.Session{
		ID:           "stream:global:series:tt2261227:2:2:2",
		StreamName:   "Stream04",
		ContentType:  "series",
		ContentID:    "tt2261227:2:2",
		ContentTitle: "Altered Carbon",
	}
	sess.SetSelectedPlaybackFile("Altered.Carbon.S02E03.1080p.mkv")

	params := server.recordAttemptParams(sess)
	if got := params.ServedFile; got != "Altered.Carbon.S02E03.1080p.mkv" {
		t.Fatalf("ServedFile = %q, want %q", got, "Altered.Carbon.S02E03.1080p.mkv")
	}
	if got := params.StreamName; got != "Stream04" {
		t.Fatalf("StreamName = %q, want %q", got, "Stream04")
	}
	if got := params.ContentTitle; got != "Altered Carbon" {
		t.Fatalf("ContentTitle = %q, want %q", got, "Altered Carbon")
	}
}

func TestCacheReturnedPlaybackBlueprintReplacesStaleBlueprint(t *testing.T) {
	sess := &session.Session{}
	stale := &unpack.DirectBlueprint{FileName: "Show.S01E04.mkv", FileIndex: 1, Target: unpack.EpisodeTarget{Season: 1, Episode: 4}}
	fresh := &unpack.DirectBlueprint{FileName: "Show.S01E01.mkv", FileIndex: 0, Target: unpack.EpisodeTarget{Season: 1, Episode: 1}}
	sess.SetBlueprint(stale)

	cacheReturnedPlaybackBlueprint(sess, fresh)

	if got := sess.Blueprint; got != fresh {
		t.Fatalf("expected session blueprint to be replaced, got %#v", got)
	}
}
