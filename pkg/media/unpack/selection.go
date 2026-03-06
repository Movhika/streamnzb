package unpack

import (
	"path/filepath"

	searchparser "streamnzb/pkg/search/parser"
)

type EpisodeTarget struct {
	Season  int
	Episode int
}

func (t EpisodeTarget) Valid() bool {
	return t.Season > 0 && t.Episode > 0
}

type namedEpisodeCandidate struct {
	Name  string
	Size  int64
	Index int
	Order int
}

func selectEpisodeCandidate(candidates []namedEpisodeCandidate, target EpisodeTarget) (namedEpisodeCandidate, bool) {
	if !target.Valid() {
		return namedEpisodeCandidate{}, false
	}
	var best namedEpisodeCandidate
	bestRank := 0
	found := false
	for _, candidate := range candidates {
		rank := episodeNameMatchRank(candidate.Name, target)
		if rank == 0 {
			continue
		}
		if !found || rank > bestRank ||
			(rank == bestRank && (candidate.Size > best.Size ||
				(candidate.Size == best.Size && candidate.Order < best.Order))) {
			best = candidate
			bestRank = rank
			found = true
		}
	}
	return best, found
}

func episodeNameMatchRank(name string, target EpisodeTarget) int {
	if !target.Valid() {
		return 0
	}
	parsed := searchparser.ParseReleaseTitle(filepath.Base(name))
	if parsed == nil {
		return 0
	}
	return parsed.EpisodeMatchRank(target.Season, target.Episode)
}

func selectDirectFileIndex(files []UnpackableFile, target EpisodeTarget) int {
	firstVideoIdx := -1
	candidates := make([]namedEpisodeCandidate, 0, len(files))
	for i, f := range files {
		name := ExtractFilename(f.Name())
		if !IsVideoFile(name) {
			continue
		}
		if firstVideoIdx == -1 {
			firstVideoIdx = i
		}
		candidates = append(candidates, namedEpisodeCandidate{Name: name, Size: f.Size(), Index: i, Order: len(candidates)})
	}
	if best, ok := selectEpisodeCandidate(candidates, target); ok {
		return best.Index
	}
	return firstVideoIdx
}
