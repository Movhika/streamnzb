package pttoptions

import "strings"

// All canonical option slices for PTT-based filter/sort. Used for validation and UI.

// Audio options (go-ptt / PTT)
var AudioOptions = []string{
	"DTS Lossless", "DTS Lossy", "Atmos", "TrueHD", "FLAC", "DDP", "EAC3", "DD", "AC3", "AAC", "PCM", "OPUS", "HQ", "MP3",
}

// BitDepth options
var BitDepthOptions = []string{"8bit", "10bit", "12bit"}

// Channels options
var ChannelsOptions = []string{"2.0", "5.1", "7.1", "stereo", "mono"}

// Codec options (canonical; parser normalizes h264→AVC, etc.)
var CodecOptions = []string{"AVC", "HEVC", "MPEG-2", "DivX", "Xvid"}

// Container options
var ContainerOptions = []string{"mkv", "avi", "mp4", "wmv", "mpg", "mpeg"}

// Edition options
var EditionOptions = []string{
	"Anniversary Edition", "Ultimate Edition", "Director's Cut", "Extended Edition",
	"Collector's Edition", "Theatrical", "Uncut", "IMAX", "Diamond Edition", "Remastered",
}

// HDR / visual tag options (HDR + SDR)
var HDROptions = []string{"DV", "HDR10+", "HDR", "SDR"}

// Quality options (Cam, Web, Broadcast, Physical, Other)
var QualityOptions = []string{
	"CAM", "TeleSync", "TeleCine", "SCR",
	"WEB", "WEB-DL", "WEBRip", "WEB-DLRip",
	"HDTV", "HDTVRip", "PDTV", "TVRip", "SATRip",
	"BluRay", "BluRay REMUX", "REMUX", "BRRip", "BDRip", "UHDRip", "HDRip", "DVD", "DVDRip", "PPVRip", "R5",
	"XviD", "DivX",
}

// Region options
var RegionOptions = []string{"R0", "R1", "R2", "R2J", "R3", "R4", "R5", "R6", "R7", "R8", "R9", "PAL", "NTSC", "SECAM"}

// Resolution options (canonical for filter/sort; 4k = 2160p)
var ResolutionOptions = []string{"4k", "2160p", "2k", "1440p", "1080p", "720p", "576p", "480p", "360p", "240p"}

// ResolutionGroupOptions are the grouped resolution names used in sort (4k, 1080p, 720p, sd)
var ResolutionGroupOptions = []string{"4k", "1080p", "720p", "sd"}

// ThreeD options
var ThreeDOptions = []string{"3D", "3D HSBS", "3D SBS", "3D HOU", "3D OU"}

// LanguageOptions: special + ISO 639-1 (subset used in PTT). Stored in config; indexers often return full names (e.g. "English").
var LanguageOptions = []string{
	"multi subs", "multi audio", "dual audio",
	"en", "ja", "ko", "zh", "zh-tw", "fr", "es", "es-419", "pt", "it", "de", "ru", "uk", "nl", "da", "fi", "sv", "no", "el", "lt", "lv", "et", "pl", "cs", "sk", "hu", "ro", "bg", "sr", "hr", "sl", "hi", "te", "ta", "ml", "kn", "mr", "gu", "pa", "bn", "vi", "id", "th", "ms", "ar", "tr", "he", "fa",
}

// languageFullNameToCode maps full language names (lowercase) to config/short code. Used to normalize UI or legacy config to codes.
var languageFullNameToCode = map[string]string{
	"multi subs": "multi subs", "multi audio": "multi audio", "dual audio": "dual audio",
	"english": "en", "japanese": "ja", "korean": "ko", "chinese": "zh", "french": "fr", "spanish": "es",
	"portuguese": "pt", "italian": "it", "german": "de", "russian": "ru", "ukrainian": "uk", "dutch": "nl",
	"danish": "da", "finnish": "fi", "swedish": "sv", "norwegian": "no", "greek": "el", "lithuanian": "lt",
	"latvian": "lv", "estonian": "et", "polish": "pl", "czech": "cs", "slovak": "sk", "hungarian": "hu",
	"romanian": "ro", "bulgarian": "bg", "serbian": "sr", "croatian": "hr", "slovenian": "sl", "hindi": "hi",
	"telugu": "te", "tamil": "ta", "malayalam": "ml", "kannada": "kn", "marathi": "mr", "gujarati": "gu",
	"punjabi": "pa", "bengali": "bn", "vietnamese": "vi", "indonesian": "id", "thai": "th", "malay": "ms",
	"arabic": "ar", "turkish": "tr", "hebrew": "he", "persian": "fa",
}

// NormalizeLanguageToCode returns the short code for the given value (full name or already a code). Unknown values are returned as-is.
func NormalizeLanguageToCode(value string) string {
	v := strings.TrimSpace(strings.ToLower(value))
	if v == "" {
		return value
	}
	if code, ok := languageFullNameToCode[v]; ok {
		return code
	}
	// Already a known code (e.g. "en", "zh-tw")
	for _, code := range LanguageOptions {
		if strings.EqualFold(code, value) {
			return code
		}
	}
	return value
}

// NormalizeLanguageSlice converts full names to short codes so config stores canonical codes.
func NormalizeLanguageSlice(s []string) []string {
	if len(s) == 0 {
		return s
	}
	out := make([]string, 0, len(s))
	seen := make(map[string]bool)
	for _, v := range s {
		code := NormalizeLanguageToCode(v)
		if code != "" && !seen[code] {
			seen[code] = true
			out = append(out, code)
		}
	}
	return out
}

// Network options (streaming / broadcast)
var NetworkOptions = []string{
	"Apple TV", "Amazon", "Netflix", "Nickelodeon", "Disney", "HBO", "Hulu", "CBS", "NBC", "AMC", "PBS", "Crunchyroll", "VICE", "Sony", "Hallmark", "Adult Swim", "Animal Planet", "Cartoon Network",
}

// ReleaseTypes (anime: OAD, OVA, ONA)
var ReleaseTypesOptions = []string{"OAD", "ODA", "OVA", "OAV", "ONA"}

// InSlice returns true if s (after ToLower) equals or is contained in any of the slice elements.
func InSlice(s string, slice []string) bool {
	lower := strings.ToLower(strings.TrimSpace(s))
	for _, v := range slice {
		if strings.ToLower(v) == lower {
			return true
		}
	}
	return false
}

// InSliceContains returns true if any element of slice contains s (case-insensitive), or equals s.
func InSliceContains(s string, slice []string) bool {
	lower := strings.ToLower(strings.TrimSpace(s))
	for _, v := range slice {
		vLower := strings.ToLower(v)
		if vLower == lower || strings.Contains(lower, vLower) || strings.Contains(vLower, lower) {
			return true
		}
	}
	return false
}

// NormalizeCodec maps parser codec strings to canonical CodecOptions (AVC, HEVC, etc.)
func NormalizeCodec(codec string) string {
	c := strings.ToLower(strings.TrimSpace(codec))
	if c == "" {
		return ""
	}
	switch {
	case strings.Contains(c, "avc") || strings.Contains(c, "h264") || strings.Contains(c, "x264"):
		return "AVC"
	case strings.Contains(c, "hevc") || strings.Contains(c, "h265") || strings.Contains(c, "x265"):
		return "HEVC"
	case strings.Contains(c, "mpeg") || strings.Contains(c, "mpeg2"):
		return "MPEG-2"
	case strings.Contains(c, "divx"):
		return "DivX"
	case strings.Contains(c, "xvid"):
		return "Xvid"
	}
	return codec
}

// NormalizeResolutionToGroup maps resolution string to 4k/1080p/720p/sd for sort
func NormalizeResolutionToGroup(res string) string {
	r := strings.ToLower(strings.TrimSpace(res))
	switch {
	case strings.Contains(r, "2160") || strings.Contains(r, "4k"):
		return "4k"
	case strings.Contains(r, "1440") || strings.Contains(r, "2k"):
		return "1080p" // treat 2k as same tier as 1080p for grouping, or add "2k" to ResolutionGroupOptions
	case strings.Contains(r, "1080"):
		return "1080p"
	case strings.Contains(r, "720"):
		return "720p"
	default:
		return "sd"
	}
}
