package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
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

	ts := time.Now().Unix()
	logFileName := fmt.Sprintf("streamnzb-%d.log", ts)
	logFilePath := filepath.Join(dataDir, logFileName)

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create log directory: %v\n", err)
	} else {
		logFileMu.Lock()
		if logFile == nil {
			var err error
			logFile, err = os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to open log file %s: %v\n", logFilePath, err)
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
