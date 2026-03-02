package seek

import (
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
