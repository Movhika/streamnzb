package unpack

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"streamnzb/pkg/media/nzb"
)

func Identify7zParts(files []UnpackableFile) ([]UnpackableFile, error) {
	var candidates []UnpackableFile

	for _, f := range files {

		name := strings.ToLower(ExtractFilename(f.Name()))

		if !strings.Contains(name, ".7z") {
			continue
		}

		if strings.HasSuffix(name, ".par2") {
			continue
		}

		if strings.HasSuffix(name, ".nzb") || strings.HasSuffix(name, ".nfo") {
			continue
		}

		candidates = append(candidates, f)
	}

	if len(candidates) == 0 {
		return nil, errors.New("no 7z files found")
	}

	sort.Slice(candidates, func(i, j int) bool {
		nameI := strings.ToLower(ExtractFilename(candidates[i].Name()))
		nameJ := strings.ToLower(ExtractFilename(candidates[j].Name()))
		return nameI < nameJ
	})

	sets := make(map[string][]UnpackableFile)

	for _, f := range candidates {
		name := ExtractFilename(f.Name())
		lower := strings.ToLower(name)

		var key string
		if strings.Contains(lower, ".7z.") {

			idx := strings.Index(lower, ".7z")
			if idx != -1 {
				key = lower[:idx+3]
			} else {
				key = lower
			}
		} else if strings.HasSuffix(lower, ".7z") {

			key = lower
		} else {

			key = lower
		}

		sets[key] = append(sets[key], f)
	}

	var bestSet []UnpackableFile
	var bestSetScore int64

	for _, set := range sets {
		var size int64
		hasOne := false
		for _, f := range set {
			size += f.Size()
			lower := strings.ToLower(ExtractFilename(f.Name()))
			if strings.HasSuffix(lower, ".7z.001") || strings.HasSuffix(lower, ".7z") {
				hasOne = true
			}
		}

		if !hasOne && len(set) > 0 {

		}

		if bestSet == nil || size > bestSetScore {
			bestSetScore = size
			bestSet = set
		} else if size == bestSetScore {

			if hasOne {
				bestSet = set
			}
		}
	}

	if len(bestSet) == 0 {
		return nil, errors.New("no valid 7z sets found")
	}

	sort.Slice(bestSet, func(i, j int) bool {
		return Get7zVolumeNumber(bestSet[i].Name()) < Get7zVolumeNumber(bestSet[j].Name())
	})

	return bestSet, nil
}

func Validate7zArchive(files []nzb.File) error {
	var candidates []*nzb.File

	for i := range files {
		f := &files[i]

		name := strings.ToLower(ExtractFilename(f.Subject))

		if !strings.Contains(name, ".7z") {
			continue
		}

		if strings.HasSuffix(name, ".par2") || strings.HasSuffix(name, ".nzb") || strings.HasSuffix(name, ".nfo") {
			continue
		}

		candidates = append(candidates, f)
	}

	if len(candidates) == 0 {
		return nil
	}

	sets := make(map[string][]*nzb.File)
	for _, f := range candidates {
		name := ExtractFilename(f.Subject)
		lower := strings.ToLower(name)
		var key string
		if strings.Contains(lower, ".7z.") {

			idx := strings.Index(lower, ".7z")
			if idx != -1 {
				key = lower[:idx+3]
			} else {
				key = lower
			}
		} else if strings.HasSuffix(lower, ".7z") {
			key = lower
		} else {
			key = lower
		}
		sets[key] = append(sets[key], f)
	}

	var bestSet []*nzb.File
	var bestSetScore int64

	for _, set := range sets {
		var size int64
		hasOne := false
		for _, f := range set {

			fileSize := int64(0)
			for _, s := range f.Segments {
				fileSize += s.Bytes
			}
			size += fileSize

			lower := strings.ToLower(ExtractFilename(f.Subject))
			if strings.HasSuffix(lower, ".7z.001") || strings.HasSuffix(lower, ".7z") {
				hasOne = true
			}
		}

		if bestSet == nil || size > bestSetScore {
			bestSetScore = size
			bestSet = set
		} else if size == bestSetScore {
			if hasOne {
				bestSet = set
			}
		}
	}

	if len(bestSet) == 0 {
		return nil
	}

	sort.Slice(bestSet, func(i, j int) bool {
		nameI := strings.ToLower(ExtractFilename(bestSet[i].Subject))
		nameJ := strings.ToLower(ExtractFilename(bestSet[j].Subject))
		return nameI < nameJ
	})

	firstRawSubject := bestSet[0].Subject
	first := strings.ToLower(ExtractFilename(firstRawSubject))

	isSplit := false
	for _, f := range bestSet {
		if strings.Contains(strings.ToLower(ExtractFilename(f.Subject)), ".7z.") {
			isSplit = true
			break
		}
	}

	if !isSplit {

		return nil
	}

	if !strings.HasSuffix(first, ".001") {
		return fmt.Errorf("split 7z archive missing part .001 (first found: %s)", first)
	}

	for i, f := range bestSet {
		expectedSuffix := fmt.Sprintf(".%03d", i+1)
		name := strings.ToLower(ExtractFilename(f.Subject))
		if !strings.HasSuffix(name, expectedSuffix) {
			return fmt.Errorf("7z archive sequence error: expected part %s, found %s", expectedSuffix, name)
		}
	}

	totalParts := parseTotalParts(firstRawSubject)
	if totalParts > 0 && len(bestSet) != totalParts {
		return fmt.Errorf("7z archive missing parts: found %d, expected %d", len(bestSet), totalParts)
	}

	return nil
}

func parseTotalParts(subject string) int {

	s := strings.ToLower(subject)

	idx := strings.Index(s, "1/")
	if idx == -1 {

		idx = strings.Index(s, "01/")
	}
	if idx == -1 {

		idx = strings.Index(s, "001/")
	}

	if idx != -1 {

		slashIdx := strings.Index(s[idx:], "/") + idx
		rest := s[slashIdx+1:]

		end := 0
		for end < len(rest) && isDigit(rest[end]) {
			end++
		}

		if end > 0 {

			var total int
			if _, err := fmt.Sscanf(rest[:end], "%d", &total); err == nil {
				return total
			}
		}
	}

	return 0
}
