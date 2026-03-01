package unpack

import (
	"strings"

	"streamnzb/pkg/media/fileutil"
)

const (
	ExtRar  = ".rar"
	ExtZip  = ".zip"
	Ext7z   = ".7z"
	ExtIso  = ".iso"
	ExtMkv  = ".mkv"
	ExtMp4  = ".mp4"
	ExtAvi  = ".avi"
	ExtM2ts = ".m2ts"
	ExtTs   = ".ts"
	ExtVob  = ".vob"
	ExtWmv  = ".wmv"
	ExtFlv  = ".flv"
	ExtWebm = ".webm"
	ExtMov  = ".mov"
	ExtPar2 = ".par2"
	ExtNfo  = ".nfo"
	ExtNzb  = ".nzb"
)

// ExtractFilename extracts a clean filename from an NZB subject line or file path.
func ExtractFilename(subject string) string {
	return fileutil.ExtractFilename(subject)
}

// IsVideoFile returns true if name has a video extension.
func IsVideoFile(name string) bool {
	return fileutil.IsVideoFile(name)
}

func IsArchiveFile(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ExtRar) ||
		strings.HasSuffix(lower, ExtZip) ||
		strings.HasSuffix(lower, Ext7z) ||
		strings.HasSuffix(lower, ExtIso) ||
		IsRarPart(lower) ||
		IsSplitArchivePart(lower)
}

func IsSampleFile(name string) bool {
	return strings.Contains(strings.ToLower(name), "sample")
}

// IsRarPart returns true for .rNN extensions (e.g. .r01, .r99).
func IsRarPart(name string) bool {
	if len(name) < 4 {
		return false
	}
	ext := name[len(name)-4:]
	return ext[0] == '.' && ext[1] == 'r' && isDigit(ext[2]) && isDigit(ext[3])
}

// IsMiddleRarVolume returns true for non-first RAR volumes.
func IsMiddleRarVolume(name string) bool {
	name = strings.ToLower(name)

	// .partXX.rar: first = part1/part01/part001
	if strings.Contains(name, ".part") && strings.HasSuffix(name, ExtRar) {
		if strings.Contains(name, ".part1.rar") ||
			strings.Contains(name, ".part01.rar") ||
			strings.Contains(name, ".part001.rar") {
			return false
		}
		return true
	}

	// .rNN: ALL .rNN are continuations; the first volume is always .rar
	if len(name) >= 4 && name[len(name)-4:len(name)-2] == ".r" {
		last := name[len(name)-2:]
		if last != "ar" {
			return true
		}
	}
	return false
}

// IsSplitArchivePart returns true for .zNN and .NNN split extensions.
func IsSplitArchivePart(name string) bool {
	if len(name) < 4 {
		return false
	}
	ext := strings.ToLower(name[len(name)-4:])

	// .zNN (zip/7z split)
	if ext[0] == '.' && ext[1] == 'z' && isDigit(ext[2]) && isDigit(ext[3]) {
		return true
	}
	// .NNN (7z/HJSplit)
	if ext[0] == '.' && isDigit(ext[1]) && isDigit(ext[2]) && isDigit(ext[3]) {
		return true
	}
	return false
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }
