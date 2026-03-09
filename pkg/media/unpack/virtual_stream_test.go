package unpack

import (
	"context"
	"errors"
	"io"
	"testing"
)

func TestVirtualStreamLiveCount(t *testing.T) {
	before := LiveVirtualStreams()
	stream := NewVirtualStream(context.Background(), nil, 0, 0)
	if got := LiveVirtualStreams(); got != before+1 {
		_ = stream.Close()
		t.Fatalf("expected %d live virtual streams, got %d", before+1, got)
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if got := LiveVirtualStreams(); got != before {
		t.Fatalf("expected %d live virtual streams after close, got %d", before, got)
	}
}

func TestVirtualStreamReadsAcrossPartBoundaries(t *testing.T) {
	stream := NewVirtualStream(context.Background(), []virtualPart{
		{VirtualStart: 0, VirtualEnd: 3, VolFile: &memoryUnpackableFile{name: "part1", data: []byte("abc")}},
		{VirtualStart: 3, VirtualEnd: 7, VolFile: &memoryUnpackableFile{name: "part2", data: []byte("defg")}},
	}, 7, 0)
	defer stream.Close()

	got, err := io.ReadAll(stream)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if string(got) != "abcdefg" {
		t.Fatalf("expected concatenated stream %q, got %q", "abcdefg", string(got))
	}
}

func TestVirtualStreamSeekNearEOFReturnsRemainingBytesThenEOF(t *testing.T) {
	stream := NewVirtualStream(context.Background(), []virtualPart{
		{VirtualStart: 0, VirtualEnd: 3, VolFile: &memoryUnpackableFile{name: "part1", data: []byte("abc")}},
		{VirtualStart: 3, VirtualEnd: 7, VolFile: &memoryUnpackableFile{name: "part2", data: []byte("defg")}},
	}, 7, 0)
	defer stream.Close()

	if _, err := stream.Seek(-2, io.SeekEnd); err != nil {
		t.Fatalf("Seek returned error: %v", err)
	}

	buf := make([]byte, 4)
	n, err := stream.Read(buf)
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected to read 2 bytes near EOF, got %d", n)
	}
	if string(buf[:n]) != "fg" {
		t.Fatalf("expected remaining bytes %q, got %q", "fg", string(buf[:n]))
	}

	n, err = stream.Read(buf)
	if n != 0 || !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF after tail read, got (%d, %v)", n, err)
	}
}

func TestVirtualStreamSeekToEndReturnsEOFOnRead(t *testing.T) {
	stream := NewVirtualStream(context.Background(), []virtualPart{
		{VirtualStart: 0, VirtualEnd: 3, VolFile: &memoryUnpackableFile{name: "part1", data: []byte("abc")}},
		{VirtualStart: 3, VirtualEnd: 7, VolFile: &memoryUnpackableFile{name: "part2", data: []byte("defg")}},
	}, 7, 0)
	defer stream.Close()

	if _, err := stream.Seek(0, io.SeekEnd); err != nil {
		t.Fatalf("Seek returned error: %v", err)
	}

	buf := make([]byte, 1)
	n, err := stream.Read(buf)
	if n != 0 || !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF after seek to end, got (%d, %v)", n, err)
	}
}
