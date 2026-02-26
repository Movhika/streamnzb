package parser

import (
	"strconv"
	"strings"

	"streamnzb/pkg/core/config/pttoptions"

	"github.com/MunifTanjim/go-ptt"
)

// ParsedRelease contains parsed metadata from a release title (go-ptt Result + normalized Codec).
type ParsedRelease struct {
	Title      string
	Year       int
	Resolution string
	Quality    string
	Codec      string   // Canonical: AVC, HEVC, MPEG-2, DivX, Xvid
	Audio      []string
	Channels   []string
	HDR        []string
	Container  string
	Group      string
	Season     int
	Episode    int

	Languages []string
	Network   string
	Repack    bool
	Proper    bool
	Extended  bool
	Unrated   bool
	ThreeD    string
	Size      string
	BitDepth  string
	Dubbed    bool
	Hardcoded bool

	// Extra PTT fields for filter/sort
	Edition      string
	Date         string   // YYYY-MM-DD
	Commentary   bool
	Complete     bool
	Convert      bool
	Documentary  bool
	Remastered   bool
	Retail       bool
	Subbed       bool
	Uncensored   bool
	Upscaled     bool
	Region       string
	ReleaseTypes []string
	EpisodeCode  string
	Site         string
	Extension    string
	Volumes      []int
}

// ParseReleaseTitle parses a release title using go-ptt and normalizes Codec to canonical (AVC, HEVC, etc.).
func ParseReleaseTitle(title string) *ParsedRelease {
	info := ptt.Parse(title)

	codec := pttoptions.NormalizeCodec(info.Codec)
	if codec == "" && info.Codec != "" {
		codec = info.Codec
	}

	parsed := &ParsedRelease{
		Title:       info.Title,
		Resolution:  info.Resolution,
		Quality:     info.Quality,
		Codec:       codec,
		Audio:       info.Audio,
		Channels:    info.Channels,
		HDR:         info.HDR,
		Container:   info.Container,
		Group:       info.Group,
		Languages:   info.Languages,
		Network:     info.Network,
		Repack:      info.Repack,
		Proper:      info.Proper,
		Extended:    info.Extended,
		Unrated:     info.Unrated,
		ThreeD:      info.ThreeD,
		Size:        info.Size,
		BitDepth:    info.BitDepth,
		Dubbed:      info.Dubbed,
		Hardcoded:   info.Hardcoded,
		Edition:     info.Edition,
		Date:        info.Date,
		Commentary:  info.Commentary,
		Complete:    info.Complete,
		Convert:     info.Convert,
		Documentary: info.Documentary,
		Remastered:  info.Remastered,
		Retail:      info.Retail,
		Subbed:      info.Subbed,
		Uncensored:  info.Uncensored,
		Upscaled:    info.Upscaled,
		Region:      info.Region,
		ReleaseTypes: info.ReleaseTypes,
		EpisodeCode: info.EpisodeCode,
		Site:        info.Site,
		Extension:   info.Extension,
		Volumes:     info.Volumes,
	}

	if info.Year != "" {
		if year, err := strconv.Atoi(info.Year); err == nil {
			parsed.Year = year
		}
	}
	if len(info.Seasons) > 0 {
		parsed.Season = info.Seasons[0]
	}
	if len(info.Episodes) > 0 {
		parsed.Episode = info.Episodes[0]
	}

	return parsed
}

// ResolutionGroup returns the resolution group (4k, 1080p, 720p, sd) from parsed metadata.
func (p *ParsedRelease) ResolutionGroup() string {
	if p == nil {
		return "sd"
	}
	res := strings.ToLower(p.Resolution)
	if strings.Contains(res, "2160") || strings.Contains(res, "4k") {
		return "4k"
	}
	if strings.Contains(res, "1080") {
		return "1080p"
	}
	if strings.Contains(res, "720") {
		return "720p"
	}
	return "sd"
}
