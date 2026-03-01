package fileutil

import "strings"

// ExtractFilename extracts a clean filename from an NZB subject line or file path.
// Handles quoted filenames, yEnc markers, (x/y) and [x/y] segment counters, and path separators.
func ExtractFilename(subject string) string {
	// Quoted filename takes priority
	if start := strings.Index(subject, "\""); start != -1 {
		if end := strings.Index(subject[start+1:], "\""); end != -1 {
			return strings.Trim(subject[start+1:start+1+end], "\"' ")
		}
	}

	clean := strings.TrimSpace(subject)

	// Strip trailing (x/y) or [x/y] segment counters
	if idx := strings.LastIndex(clean, " ("); idx != -1 {
		suffix := clean[idx:]
		if strings.Contains(suffix, "/") && strings.HasSuffix(suffix, ")") {
			clean = strings.TrimSpace(clean[:idx])
		}
	}
	if idx := strings.LastIndex(clean, " ["); idx != -1 {
		suffix := clean[idx:]
		if strings.Contains(suffix, "/") && strings.HasSuffix(suffix, "]") {
			clean = strings.TrimSpace(clean[:idx])
		}
	}

	// Strip trailing " yEnc"
	clean = strings.TrimSuffix(clean, " yEnc")
	clean = strings.TrimSpace(clean)

	// Handle file paths (RAR entry names can contain directory separators)
	if idx := strings.LastIndex(clean, "/"); idx != -1 {
		clean = clean[idx+1:]
	}
	if idx := strings.LastIndex(clean, "\\"); idx != -1 {
		clean = clean[idx+1:]
	}

	return strings.Trim(clean, "\"' ")
}

// videoExtensions is the set of extensions considered video for streaming/selection.
var videoExtensions = map[string]bool{
	".mkv": true, ".mp4": true, ".avi": true, ".m4v": true,
	".mov": true, ".wmv": true, ".flv": true, ".webm": true,
	".mpg": true, ".mpeg": true, ".m2ts": true, ".ts": true,
	".iso": true, ".vob": true,
}

// IsVideoExtension returns true if ext is a video extension (e.g. ".mkv", ".mp4").
func IsVideoExtension(ext string) bool {
	return videoExtensions[strings.ToLower(ext)]
}

// IsVideoFile returns true if name has a video extension.
func IsVideoFile(name string) bool {
	lower := strings.ToLower(name)
	for ext := range videoExtensions {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

// IsVideoOrArchiveExtension returns true for video extensions or archives that typically contain video
// (.rar, .7z, and .rNN RAR volume extensions). Used when classifying NZB files.
func IsVideoOrArchiveExtension(ext string) bool {
	ext = strings.ToLower(ext)
	if videoExtensions[ext] {
		return true
	}
	switch ext {
	case ".rar", ".7z":
		return true
	}
	// .r00, .r01, .r99 (RAR volume naming)
	if len(ext) == 4 && ext[0] == '.' && ext[1] == 'r' && ext[2] >= '0' && ext[2] <= '9' && ext[3] >= '0' && ext[3] <= '9' {
		return true
	}
	return false
}
