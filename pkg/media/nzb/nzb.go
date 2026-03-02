package nzb

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"io"
	"path/filepath"
	"strings"

	"github.com/MunifTanjim/go-ptt"
	"golang.org/x/net/html/charset"

	"streamnzb/pkg/core/logger"
	"streamnzb/pkg/media/fileutil"
)

type NZB struct {
	XMLName xml.Name `xml:"nzb"`
	Head    Head     `xml:"head"`
	Files   []File   `xml:"file"`
}

type Head struct {
	Meta []Meta `xml:"meta"`
}

type Meta struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

type File struct {
	Poster   string    `xml:"poster,attr"`
	Date     int64     `xml:"date,attr"`
	Subject  string    `xml:"subject,attr"`
	Groups   []string  `xml:"groups>group"`
	Segments []Segment `xml:"segments>segment"`
}

type Segment struct {
	Bytes  int64  `xml:"bytes,attr"`
	Number int    `xml:"number,attr"`
	ID     string `xml:",chardata"`
}

type FileInfo struct {
	File       *File
	Filename   string
	Extension  string
	Size       int64
	IsVideo    bool
	IsSample   bool
	IsExtra    bool
	ParsedInfo *ptt.Result
}

func Parse(r io.Reader) (*NZB, error) {
	var nzb NZB
	decoder := xml.NewDecoder(r)
	decoder.CharsetReader = charset.NewReaderLabel
	if err := decoder.Decode(&nzb); err != nil {
		return nil, err
	}
	return &nzb, nil
}

func (n *NZB) Password() string {
	for _, m := range n.Head.Meta {
		if strings.EqualFold(m.Type, "password") {
			return strings.TrimSpace(m.Value)
		}
	}
	return ""
}

func (n *NZB) Hash() string {
	if len(n.Files) == 0 {
		return ""
	}

	subject := n.Files[0].Subject
	if subject == "" {
		subject = n.Files[0].Poster
	}
	h := sha256.New()
	h.Write([]byte(subject))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func (n *NZB) CalculateID() string {
	if len(n.Files) == 0 || len(n.Files[0].Segments) == 0 {
		return ""
	}

	msgID := n.Files[0].Segments[0].ID
	msgID = strings.Trim(msgID, "<>")
	h := sha1.New()
	h.Write([]byte(msgID))
	return hex.EncodeToString(h.Sum(nil))
}

func (n *NZB) TotalSize() int64 {
	var total int64
	for _, file := range n.Files {
		for _, seg := range file.Segments {
			total += seg.Bytes
		}
	}
	return total
}

func (n *NZB) GetFileInfo() []*FileInfo {
	infos := make([]*FileInfo, 0, len(n.Files))

	for i := range n.Files {
		file := &n.Files[i]
		info := analyzeFile(file)
		infos = append(infos, info)
	}

	return infos
}

func (n *NZB) GetLargestContentFile() *FileInfo {
	infos := n.GetFileInfo()
	var largest *FileInfo
	var maxSize int64
	for _, info := range infos {
		if info.IsSample || info.IsExtra {
			continue
		}
		if info.Size <= maxSize {
			continue
		}
		if info.IsVideo || info.Extension == ".rar" || info.Extension == ".7z" ||
			isArchivePart(info.Extension) || isRarVolume(info.Extension) ||
			isSplitArchivePart(info.Extension) || isRarSplitPart(info.Extension, info.Filename) {
			maxSize = info.Size
			largest = info
		}
	}
	return largest
}

func (n *NZB) GetPlaybackFile() *FileInfo {
	ct := n.CompressionType()
	if ct == "rar" {
		for _, info := range n.GetContentFiles() {
			lower := strings.ToLower(info.Filename)
			if (strings.HasSuffix(lower, ".rar") && !strings.Contains(lower, ".part")) ||
				strings.Contains(lower, ".part01.") || strings.Contains(lower, ".part1.") ||
				strings.Contains(lower, ".part001.") {
				return info
			}
		}
	}
	return n.GetLargestContentFile()
}

func (n *NZB) GetContentFiles() []*FileInfo {
	infos := n.GetFileInfo()

	var mainPattern string
	var maxSize int64

	for _, info := range infos {
		if info.IsSample || info.IsExtra {
			continue
		}

		if info.Size > maxSize {

			if info.IsVideo || info.Extension == ".rar" || info.Extension == ".7z" ||
				isArchivePart(info.Extension) || isRarVolume(info.Extension) ||
				isSplitArchivePart(info.Extension) || isRarSplitPart(info.Extension, info.Filename) {
				maxSize = info.Size
				mainPattern = getFilePattern(info.Filename)
			}
		}
	}

	if mainPattern == "" {
		for _, info := range infos {
			if info.IsSample || info.IsExtra {
				continue
			}
			if info.Size > maxSize {
				maxSize = info.Size
				mainPattern = getFilePattern(info.Filename)
			}
		}
	}

	var contentFiles []*FileInfo
	if mainPattern != "" {
		for _, info := range infos {
			if getFilePattern(info.Filename) == mainPattern {
				contentFiles = append(contentFiles, info)
			}
		}
	}

	if len(contentFiles) == 0 {
		logGetContentFilesEmpty(infos, mainPattern)
	}

	return contentFiles
}

func logGetContentFilesEmpty(infos []*FileInfo, mainPattern string) {
	total := len(infos)
	samples := 0
	extras := 0
	subjects := make([]string, 0, 8)
	for _, info := range infos {
		if info.IsSample {
			samples++
		}
		if info.IsExtra {
			extras++
		}
		if len(subjects) < 8 {
			subjects = append(subjects, info.Filename)
		}
	}
	logger.Debug("GetContentFiles returned empty",
		"total_files", total,
		"samples", samples,
		"extras", extras,
		"main_pattern", mainPattern,
		"sample_filenames", subjects)
}

func getFilePattern(filename string) string {

	s := strings.ToLower(filename)

	ext := filepath.Ext(s)
	s = strings.TrimSuffix(s, ext)

	if idx := strings.LastIndex(s, ".part"); idx != -1 {
		s = s[:idx]
	}
	if idx := strings.LastIndex(s, ".vol"); idx != -1 {
		s = s[:idx]
	}

	s = strings.TrimSuffix(s, ".7z")

	return strings.Trim(s, " .-_")
}

func (n *NZB) IsRARRelease() bool {
	return n.CompressionType() == "rar"
}

func (n *NZB) CompressionType() string {
	contentFiles := n.GetContentFiles()
	if len(contentFiles) == 0 {
		return "direct"
	}

	for _, info := range contentFiles {
		if info.Extension == ".7z" || strings.Contains(strings.ToLower(info.Filename), ".7z.001") {
			return "7z"
		}
	}

	hasRarFiles := false
	for _, info := range contentFiles {
		ext := strings.ToLower(info.Extension)
		if ext == ".rar" || isRarVolume(ext) {
			hasRarFiles = true
			break
		}
	}

	if hasRarFiles {
		return "rar"
	}

	largest := n.GetLargestContentFile()
	if largest == nil {
		return "direct"
	}
	ct, _ := compressionTypeFromFileWithReason(largest.Filename, largest.Extension)
	return ct
}

func compressionTypeFromFileWithReason(filename, ext string) (string, string) {
	ext = strings.ToLower(ext)
	filenameLower := strings.ToLower(filename)

	if ext == ".7z" || strings.Contains(filenameLower, ".7z.001") {
		return "7z", "ext=.7z or contains .7z.001"
	}
	if strings.HasSuffix(filenameLower, ".7z.001") || strings.HasSuffix(filenameLower, ".7z.0001") {
		return "7z", "suffix .7z.001/.7z.0001"
	}

	if ext == ".rar" {
		return "rar", "ext=.rar"
	}
	if isRarVolume(ext) {
		return "rar", "isRarVolume(ext)"
	}

	return "direct", ""
}

func isRarVolume(ext string) bool {
	if len(ext) < 4 || !strings.HasPrefix(ext, ".r") {
		return false
	}
	for _, c := range ext[2:] {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func isRarSplitPart(ext, filename string) bool {
	if len(ext) < 3 || ext[0] != '.' {
		return false
	}
	for _, c := range ext[1:] {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func (n *NZB) GetMainVideoFile() *FileInfo {
	files := n.GetContentFiles()
	if len(files) > 0 {
		return files[0]
	}
	return nil
}

func analyzeFile(file *File) *FileInfo {

	subject := file.Subject
	if subject == "" {
		subject = file.Poster
	}
	filename := fileutil.ExtractFilename(subject)

	var size int64
	for _, seg := range file.Segments {
		size += seg.Bytes
	}

	ext := strings.ToLower(filepath.Ext(filename))

	parsed := ptt.Parse(filename)

	info := &FileInfo{
		File:       file,
		Filename:   filename,
		Extension:  ext,
		Size:       size,
		ParsedInfo: parsed,
	}

	info.IsVideo = fileutil.IsVideoOrArchiveExtension(ext)
	info.IsSample = isSampleFile(filename)
	info.IsExtra = isExtraFile(filename, ext)

	return info
}

func isArchivePart(ext string) bool {
	if len(ext) == 4 && strings.HasPrefix(ext, ".r") {
		for _, c := range ext[2:] {
			if c < '0' || c > '9' {
				return false
			}
		}
		return true
	}
	return false
}

func isSplitArchivePart(ext string) bool {
	if len(ext) != 4 {
		return false
	}
	return ext[0] == '.' &&
		ext[1] >= '0' && ext[1] <= '9' &&
		ext[2] >= '0' && ext[2] <= '9' &&
		ext[3] >= '0' && ext[3] <= '9'
}

func isSampleFile(filename string) bool {
	lower := strings.ToLower(filename)
	return strings.Contains(lower, "sample") ||
		strings.Contains(lower, "preview")
}

func isExtraFile(filename string, ext string) bool {
	extraExts := map[string]bool{
		".nfo": true, ".txt": true, ".srt": true, ".sub": true,
		".idx": true, ".ass": true, ".ssa": true, ".vtt": true,
		".jpg": true, ".png": true, ".gif": true,

		".par2": true,
	}

	if extraExts[ext] {
		return true
	}

	lower := strings.ToLower(filename)
	return strings.Contains(lower, "proof") ||
		strings.Contains(lower, "cover")
}
