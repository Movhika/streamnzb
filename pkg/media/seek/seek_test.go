package seek

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"
)

func TestInspectStreamStartParsesMP4AndResetsOffset(t *testing.T) {
	data := makeTestMP4(1000, 10000)
	stream := bytes.NewReader(data)

	info, err := InspectStreamStart(stream, int64(len(data)), "video.mp4", len(data))
	if err != nil {
		t.Fatalf("InspectStreamStart returned error: %v", err)
	}
	if !info.HeaderValid {
		t.Fatal("expected MP4 header to be valid")
	}
	if !info.DurationKnown {
		t.Fatal("expected MP4 duration to be detected")
	}
	if info.DurationSec != 10 {
		t.Fatalf("expected duration 10s, got %v", info.DurationSec)
	}
	if pos, err := stream.Seek(0, io.SeekCurrent); err != nil {
		t.Fatalf("failed to read current offset: %v", err)
	} else if pos != 0 {
		t.Fatalf("expected stream to be reset to start, got offset %d", pos)
	}
}

func TestTimeToByteOffsetFromDuration(t *testing.T) {
	if offset, ok := TimeToByteOffsetFromDuration(100, 20, 5); !ok || offset != 25 {
		t.Fatalf("expected offset 25 with ok=true, got offset=%d ok=%t", offset, ok)
	}
	if offset, ok := TimeToByteOffsetFromDuration(100, 20, 25); !ok || offset != 99 {
		t.Fatalf("expected end clamp to 99 with ok=true, got offset=%d ok=%t", offset, ok)
	}
	if _, ok := TimeToByteOffsetFromDuration(100, 0, 5); ok {
		t.Fatal("expected zero duration to fail")
	}
}

func makeTestMP4(timescale, duration uint32) []byte {
	ftyp := make([]byte, 16)
	binary.BigEndian.PutUint32(ftyp[0:4], uint32(len(ftyp)))
	copy(ftyp[4:8], []byte("ftyp"))
	copy(ftyp[8:12], []byte("isom"))

	mvhd := make([]byte, 20)
	binary.BigEndian.PutUint32(mvhd[0:4], uint32(len(mvhd)))
	copy(mvhd[4:8], []byte("mvhd"))
	binary.BigEndian.PutUint32(mvhd[12:16], timescale)
	binary.BigEndian.PutUint32(mvhd[16:20], duration)

	moov := make([]byte, 8+len(mvhd))
	binary.BigEndian.PutUint32(moov[0:4], uint32(len(moov)))
	copy(moov[4:8], []byte("moov"))
	copy(moov[8:], mvhd)

	return append(ftyp, moov...)
}