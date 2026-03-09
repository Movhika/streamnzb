package unpack

import (
	"io"
	"log/slog"
	"testing"

	"streamnzb/pkg/core/logger"
)

func discardTestLogger(t *testing.T) {
	t.Helper()
	oldLogger := logger.Log
	logger.Log = slog.New(slog.NewTextHandler(io.Discard, nil))
	t.Cleanup(func() {
		logger.Log = oldLogger
	})
}

type trackingUnpackableFile struct {
	*memoryUnpackableFile
	ensureCalls int
}

func (f *trackingUnpackableFile) EnsureSegmentMap() error {
	f.ensureCalls++
	return nil
}

func TestAggregateRemainingVolumesFromStartSkipsSegmentDetectionWhenProbeProvidesPackedSize(t *testing.T) {
	discardTestLogger(t)

	firstFile := &trackingUnpackableFile{memoryUnpackableFile: &memoryUnpackableFile{name: "release.part001.rar", data: []byte("first")}}
	secondFile := &trackingUnpackableFile{memoryUnpackableFile: &memoryUnpackableFile{name: "release.part002.rar", data: []byte("second")}}
	thirdFile := &trackingUnpackableFile{memoryUnpackableFile: &memoryUnpackableFile{name: "release.part003.rar", data: []byte("third")}}

	parts := aggregateRemainingVolumesFromStart(
		[]filePart{{name: "movie.mkv", unpackedSize: 1000, dataOffset: 100, packedSize: 200, volFile: firstFile, volName: firstFile.Name(), isMedia: true}},
		[]UnpackableFile{firstFile, secondFile, thirdFile},
		0,
		"movie.mkv",
		1000,
		continuationProbe{dataOffset: 24, packedSize: 300},
	)

	if len(parts) != 3 {
		t.Fatalf("expected 3 parts, got %d", len(parts))
	}
	if got := parts[1].packedSize; got != 300 {
		t.Fatalf("expected second continuation packed size 300, got %d", got)
	}
	if got := parts[2].packedSize; got != 500 {
		t.Fatalf("expected final continuation packed size 500, got %d", got)
	}
	if secondFile.ensureCalls != 0 || thirdFile.ensureCalls != 0 {
		t.Fatalf("expected no EnsureSegmentMap calls with probe metadata, got second=%d third=%d", secondFile.ensureCalls, thirdFile.ensureCalls)
	}
}

func TestAggregateRemainingVolumesFromStartFallsBackToSegmentDetectionWithoutProbePackedSize(t *testing.T) {
	discardTestLogger(t)

	firstFile := &trackingUnpackableFile{memoryUnpackableFile: &memoryUnpackableFile{name: "release.part001.rar", data: []byte("first")}}
	secondFile := &trackingUnpackableFile{memoryUnpackableFile: &memoryUnpackableFile{name: "release.part002.rar", data: make([]byte, 350)}}

	parts := aggregateRemainingVolumesFromStart(
		[]filePart{{name: "movie.mkv", unpackedSize: 500, dataOffset: 100, packedSize: 200, volFile: firstFile, volName: firstFile.Name(), isMedia: true}},
		[]UnpackableFile{firstFile, secondFile},
		0,
		"movie.mkv",
		500,
		continuationProbe{dataOffset: 50},
	)

	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(parts))
	}
	if got := secondFile.ensureCalls; got != 1 {
		t.Fatalf("expected one EnsureSegmentMap call in fallback path, got %d", got)
	}
	if got := parts[1].packedSize; got != 300 {
		t.Fatalf("expected fallback packed size 300, got %d", got)
	}
}
