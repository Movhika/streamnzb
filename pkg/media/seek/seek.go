package seek

import (
	"bytes"
	"io"
	"strconv"
	"strings"
)

const (
	MaxBytesToRead = 2 * 1024 * 1024
)

func TimeToByteOffset(stream io.ReadSeeker, size int64, filename string, tSeconds float64) (offset int64, ok bool) {
	if size <= 0 || tSeconds < 0 {
		return 0, false
	}
	format := formatFromFilename(filename)
	if format == "" {
		return 0, false
	}

	readSize := int(size)
	if readSize > MaxBytesToRead {
		readSize = MaxBytesToRead
	}
	buf := make([]byte, readSize)
	n, err := io.ReadFull(stream, buf)
	if err != nil && err != io.ErrUnexpectedEOF {
		return 0, false
	}
	buf = buf[:n]

	if _, err := stream.Seek(0, io.SeekStart); err != nil {
		return 0, false
	}
	var durationSec float64
	switch format {
	case "mp4":
		durationSec, ok = durationFromMP4(buf)
	case "mkv":
		durationSec, ok = durationFromMKV(buf)
	default:
		return 0, false
	}
	if !ok || durationSec <= 0 {
		return 0, false
	}
	if tSeconds >= durationSec {

		if size > 1 {
			return size - 1, true
		}
		return 0, true
	}

	offset = int64((tSeconds / durationSec) * float64(size))
	if offset < 0 {
		offset = 0
	}
	if offset >= size {
		offset = size - 1
	}
	return offset, true
}

func formatFromFilename(filename string) string {
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".mp4"), strings.HasSuffix(lower, ".m4v"), strings.HasSuffix(lower, ".mov"):
		return "mp4"
	case strings.HasSuffix(lower, ".mkv"), strings.HasSuffix(lower, ".webm"):
		return "mkv"
	default:
		return ""
	}
}

func ParseTSeconds(t string) (float64, bool) {
	if t == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
	if err != nil {
		return 0, false
	}
	if f < 0 {
		return 0, false
	}
	return f, true
}

// EBML header ID (MKV/WebM) - first element in file.
var ebmlHeaderID = []byte{0x1A, 0x45, 0xDF, 0xA3}

// ftyp box type (MP4/MOV).
var ftypBox = []byte{'f', 't', 'y', 'p'}

// ValidateContainerHeader checks that data starts with valid container headers for the given filename.
// Used by the play probe to ensure RAR/7z/direct streams have readable, valid headers before serving.
func ValidateContainerHeader(data []byte, filename string) bool {
	if len(data) < 8 {
		return false
	}
	switch formatFromFilename(filename) {
	case "mkv":
		return len(data) >= 4 && bytes.HasPrefix(data, ebmlHeaderID)
	case "mp4":
		// MP4: optional 4-byte size (big-endian) then "ftyp", or "ftyp" at 0
		if bytes.HasPrefix(data, ftypBox) {
			return true
		}
		if len(data) >= 8 && bytes.Equal(data[4:8], ftypBox) {
			return true
		}
		return false
	default:
		// AVI, etc.: no strict check; caller may use probe size only to trigger segment read
		return true
	}
}
