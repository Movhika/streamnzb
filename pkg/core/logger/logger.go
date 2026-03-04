package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"streamnzb/pkg/core/env"
	"streamnzb/pkg/core/paths"
)

var Log *slog.Logger

type BroadcastHandler struct {
	slog.Handler
	ch chan<- string
}

func (h *BroadcastHandler) Handle(ctx context.Context, r slog.Record) error {

	err := h.Handler.Handle(ctx, r)

	if h.ch != nil {

		locationMu.RLock()
		loc := logLocation
		locationMu.RUnlock()
		if loc == nil {
			loc = time.Local
		}

		utcTime := r.Time.UTC()
		formattedTime := utcTime.In(loc)

		msg := fmt.Sprintf("time=%s level=%s msg=%q", formattedTime.Format("2006-01-02T15:04:05.000-07:00"), r.Level, r.Message)
		r.Attrs(func(a slog.Attr) bool {
			msg += fmt.Sprintf(" %s=%v", a.Key, a.Value)
			return true
		})

		select {
		case h.ch <- msg:
		default:

		}
	}
	return err
}

var _ = time.Now

var broadcastCh chan<- string

func SetBroadcast(ch chan<- string) {
	broadcastCh = ch

}

func Init(levelStr string) {
	var level slog.Level
	switch strings.ToUpper(levelStr) {
	case "TRACE":
		level = slog.LevelDebug - 1
	case "DEBUG":
		level = slog.LevelDebug
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	tzEnv := env.TZ()
	var loc *time.Location
	locationMu.Lock()
	if tzEnv != "" {
		loadedLoc, err := time.LoadLocation(tzEnv)
		if err != nil {

			loc = time.Local
			logLocation = time.Local
		} else {
			loc = loadedLoc
			logLocation = loadedLoc
		}
	} else {

		loc = time.Local
		logLocation = time.Local
	}
	locationMu.Unlock()

	dataDir := paths.GetDataDir()
	const currentLogName = "streamnzb.log"
	currentPath := filepath.Join(dataDir, currentLogName)

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create log directory: %v\n", err)
	} else {
		if fi, err := os.Stat(currentPath); err == nil && fi.Mode().IsRegular() {
			ts := fi.ModTime().Format("20060102-150405")
			archivedName := fmt.Sprintf("streamnzb-%s.log", ts)
			archivedPath := filepath.Join(dataDir, archivedName)
			if renameErr := os.Rename(currentPath, archivedPath); renameErr != nil {
				fmt.Fprintf(os.Stderr, "Failed to rotate log file %s -> %s: %v\n", currentPath, archivedPath, renameErr)
			}
		}
		logFileMu.Lock()
		if logFile == nil {
			var err error
			logFile, err = os.OpenFile(currentPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to open log file %s: %v\n", currentPath, err)
			}
		}
		logFileMu.Unlock()
	}

	tzLoc := loc
	opts := &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {

			if a.Key == slog.TimeKey {

				t := a.Value.Time().In(tzLoc)
				return slog.String("time", t.Format("2006-01-02T15:04:05.000-07:00"))
			}
			return a
		},
	}

	baseHandler := slog.NewTextHandler(os.Stdout, opts)

	handler := &GlobalBroadcastHandler{
		Handler: baseHandler,
	}

	Log = slog.New(handler)
	slog.SetDefault(Log)

	locationMu.RLock()
	currentLoc := logLocation
	currentTZEnv := tzEnv
	locationMu.RUnlock()
	if currentLoc != nil {
		Log.Info("Logger initialized", "timezone", currentLoc.String(), "tz_env", currentTZEnv)
	}
}

type GlobalBroadcastHandler struct {
	slog.Handler
}

var (
	history     []string
	historyMu   sync.RWMutex
	maxHistory  = 500
	logFile     *os.File
	logFileMu   sync.Mutex
	logLocation *time.Location
	locationMu  sync.RWMutex
)

func (h *GlobalBroadcastHandler) Handle(ctx context.Context, r slog.Record) error {

	locationMu.RLock()
	loc := logLocation
	locationMu.RUnlock()

	if loc == nil {
		loc = time.Local
	}

	formattedTime := r.Time.In(loc)

	msg := fmt.Sprintf("time=%s level=%s msg=%q", formattedTime.Format("2006-01-02T15:04:05.000-07:00"), r.Level, r.Message)
	r.Attrs(func(a slog.Attr) bool {
		msg += fmt.Sprintf(" %s=%v", a.Key, a.Value)
		return true
	})

	historyMu.Lock()
	if len(history) >= maxHistory {
		history = history[1:]
	}
	history = append(history, msg)
	historyMu.Unlock()

	err := h.Handler.Handle(ctx, r)

	logFileMu.Lock()
	if logFile != nil {
		fmt.Fprintln(logFile, msg)
	}
	logFileMu.Unlock()

	if broadcastCh != nil {
		select {
		case broadcastCh <- msg:
		default:
		}
	}
	return err
}

func GetHistory() []string {
	historyMu.RLock()
	defer historyMu.RUnlock()

	cp := make([]string, len(history))
	copy(cp, history)
	return cp
}

// PurgeOldLogs removes older rotated log files so at most keepCount log files remain
// (the current streamnzb.log plus up to keepCount-1 archived streamnzb-*.log files).
// Only timestamped files (streamnzb-*.log) are considered for deletion; the current
// streamnzb.log is never removed. keepCount must be at least 1.
func PurgeOldLogs(keepCount int) {
	if keepCount < 1 {
		return
	}
	dataDir := paths.GetDataDir()
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return
	}
	type namedInfo struct {
		name string
		path string
		mod  time.Time
	}
	var archived []namedInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if name == "streamnzb.log" || !strings.HasPrefix(name, "streamnzb-") || !strings.HasSuffix(name, ".log") {
			continue
		}
		path := filepath.Join(dataDir, name)
		info, err := e.Info()
		if err != nil {
			continue
		}
		archived = append(archived, namedInfo{name: name, path: path, mod: info.ModTime()})
	}
	maxArchived := keepCount - 1
	if len(archived) <= maxArchived {
		return
	}
	sort.Slice(archived, func(i, j int) bool { return archived[i].mod.Before(archived[j].mod) })
	for i := 0; i < len(archived)-maxArchived; i++ {
		_ = os.Remove(archived[i].path)
	}
}

func SetLevel(levelStr string) {
	Init(levelStr)
}

func Trace(msg string, args ...any) {
	Log.Log(context.TODO(), slog.LevelDebug-1, msg, args...)
}

func Debug(msg string, args ...any) {
	Log.Debug(msg, args...)
}

func Info(msg string, args ...any) {
	Log.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	Log.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	Log.Error(msg, args...)
}
