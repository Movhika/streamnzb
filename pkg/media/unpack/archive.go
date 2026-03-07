package unpack

import (
	"context"
	"errors"
	"io"
	"strings"

	"streamnzb/pkg/core/logger"
)

var ErrTooManyZeroFills = errors.New("too many failed segments")

type ReadSeekCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

type DirectBlueprint struct {
	FileName  string
	FileIndex int
}

type FailedBlueprint struct {
	Err error
}

func GetMediaStream(ctx context.Context, files []UnpackableFile, cachedBP interface{}, password string) (ReadSeekCloser, string, int64, interface{}, error) {
	return GetMediaStreamForEpisode(ctx, files, cachedBP, password, EpisodeTarget{})
}

func GetMediaStreamForEpisode(ctx context.Context, files []UnpackableFile, cachedBP interface{}, password string, target EpisodeTarget) (ReadSeekCloser, string, int64, interface{}, error) {
	if cachedBP != nil {
		switch bp := cachedBP.(type) {
		case *ArchiveBlueprint:
			logger.Debug("Using cached RAR blueprint", "file", bp.MainFileName)
			s, name, size, err := StreamFromBlueprint(ctx, bp, password)
			return s, name, size, bp, err
		case *SevenZipBlueprint:
			logger.Debug("Using cached 7z blueprint", "file", bp.MainFileName)
			if len(bp.Files) == 0 {
				return nil, "", 0, nil, errors.New("7z set empty for cached blueprint")
			}
			s, n, sz, err := Open7zStreamFromBlueprint(ctx, bp, password)
			return s, n, sz, bp, err
		case *DirectBlueprint:
			if bp.FileIndex < len(files) {
				f := files[bp.FileIndex]
				stream, err := f.OpenStreamCtx(ctx)
				if err != nil {
					return nil, "", 0, nil, err
				}
				return stream, bp.FileName, f.Size(), bp, nil
			}
		case *FailedBlueprint:
			logger.Debug("Using cached scan failure", "err", bp.Err)
			return nil, "", 0, bp, bp.Err
		}
	}

	rarFiles := filterRarFiles(files)
	var rarScanFailed bool
	if len(rarFiles) > 0 {
		logger.Trace("Detected RAR archive", "volumes", len(rarFiles))
		unpackables := make([]UnpackableFile, len(files))
		copy(unpackables, files)
		bp, err := ScanArchive(unpackables, password, target)
		if err != nil {
			rarScanFailed = true
			logger.Warn("ScanArchive failed, falling back to other methods", "err", err)
		} else {
			s, name, size, err := StreamFromBlueprint(ctx, bp, password)
			if err != nil {
				return nil, "", 0, nil, err
			}
			return s, name, size, bp, nil
		}
	}

	archiveFiles, err := Identify7zParts(files)
	if err == nil && len(archiveFiles) > 0 {
		firstVolName := ExtractFilename(archiveFiles[0].Name())
		logger.Info("Detected 7z archive", "name", firstVolName, "parts", len(archiveFiles))
		newBp, err := CreateSevenZipBlueprint(archiveFiles, firstVolName, password, target)
		if err != nil {
			return nil, "", 0, nil, err
		}
			s, n, sz, err := Open7zStreamFromBlueprint(ctx, newBp, password)
		return s, n, sz, newBp, err
	}

	if directIdx := selectDirectFileIndex(files, target); directIdx >= 0 {
		f := files[directIdx]
		name := ExtractFilename(f.Name())
		stream, err := f.OpenStreamCtx(ctx)
		if err != nil {
			return nil, "", 0, nil, err
		}
		return stream, name, f.Size(), &DirectBlueprint{FileName: name, FileIndex: directIdx}, nil
	}

	var largestFile UnpackableFile
	var largestIdx int
	for i, f := range files {
		name := strings.ToLower(ExtractFilename(f.Name()))
		if strings.HasSuffix(name, ExtRar) || strings.Contains(name, ".part") || IsRarPart(name) || IsSplitArchivePart(name) {
			continue
		}
		if strings.HasSuffix(name, ExtPar2) || strings.HasSuffix(name, ExtNzb) || strings.HasSuffix(name, ExtNfo) {
			continue
		}
		if largestFile == nil || f.Size() > largestFile.Size() {
			largestFile = f
			largestIdx = i
		}
	}

	if largestFile != nil && largestFile.Size() > 50*1024*1024 {
		if !rarScanFailed {
			logger.Warn("No clear media found, probing largest file", "name", largestFile.Name(), "size", largestFile.Size())
			unpackables := make([]UnpackableFile, len(files))
			copy(unpackables, files)
			logger.Info("Attempting heuristic RAR scan on unknown files")
			bp, err := ScanArchive(unpackables, password, target)
			if err == nil {
				s, name, size, err := StreamFromBlueprint(ctx, bp, password)
				if err == nil {
					logger.Info("Heuristic scan found RAR archive")
					return s, name, size, bp, nil
				}
			} else {
				logger.Warn("Heuristic RAR scan failed, falling back to direct stream", "err", err)
			}
		}
		extractedName := ExtractFilename(largestFile.Name())
		stream, err := largestFile.OpenStreamCtx(ctx)
		if err != nil {
			return nil, "", 0, nil, err
		}
		return stream, extractedName, largestFile.Size(), &DirectBlueprint{FileName: extractedName, FileIndex: largestIdx}, nil
	}

	logger.Warn("GetMediaStream found no suitable media", "files", len(files), "rar_candidates", len(rarFiles))
	return nil, "", 0, &FailedBlueprint{Err: io.EOF}, io.EOF
}
