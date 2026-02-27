package seek

import (
	"io"
	"strconv"
	"strings"
)

const (
	// MaxBytesToRead is the maximum prefix read from the stream to parse duration.
	MaxBytesToRead = 2 * 1024 * 1024 // 2 MB
)

// TimeToByteOffset converts a start time tSeconds (in seconds) to a byte offset
// using the container's duration. It reads a prefix of the seekable stream,
// detects format from filename (and optionally magic), parses duration, then
// seeks the stream back to 0. Returns the byte offset and true on success.
// On unknown format or parse error, returns (0, false) and the stream position
// is undefined (caller should Seek(0, io.SeekStart) if they need to continue).
func TimeToByteOffset(stream io.ReadSeeker, size int64, filename string, tSeconds float64) (offset int64, ok bool) {
	if size <= 0 || tSeconds < 0 {
		return 0, false
	}
	format := formatFromFilename(filename)
	if format == "" {
		return 0, false
	}
	// Read prefix
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
	// Seek back so ServeContent can use the stream from start
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
		// Clamp to near end
		if size > 1 {
			return size - 1, true
		}
		return 0, true
	}
	// Linear approximation: offset = (t / duration) * size
	offset = int64((tSeconds / durationSec) * float64(size))
	if offset < 0 {
		offset = 0
	}
	if offset >= size {
		offset = size - 1
	}
	return offset, true
}

// formatFromFilename returns "mp4", "mkv", or "" for unsupported.
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

// ParseTSeconds parses the t= query parameter as seconds (integer or float).
// Returns (value, true) or (0, false) on invalid or negative.
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
