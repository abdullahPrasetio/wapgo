//go:build ignore

package logger

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

// SinkConfig configures the four per-category structured log files written to Dir:
//
//	api.log         — one JSON line per HTTP request (full request + response)
//	consumer.log    — one JSON line per consumed message
//	thirdparty.log  — one JSON line per outbound third-party call
//	trace.log       — one JSON line per custom trace point
//
// Each file rotates either by size (lumberjack) or daily (date-stamped filename).
type SinkConfig struct {
	Dir        string // directory for log files (default "logs")
	Rotation   string // "size" | "daily" (default "size")
	MaxAgeDays int    // retention in days (default 30)
	MaxSizeMB  int    // size mode: MB per file before rotation (default 100)
	Console    bool   // also echo each sink to stdout (dev convenience)
}

// The four category loggers. Until SetupSinks runs they are no-ops, so code that
// logs to a sink before configuration simply discards the line instead of panicking.
var (
	apiLogger        = zerolog.Nop()
	consumerLogger   = zerolog.Nop()
	thirdPartyLogger = zerolog.Nop()
	traceLogger      = zerolog.Nop()
)

// SetupSinks initializes the four structured log files. Call once at startup.
func SetupSinks(cfg SinkConfig) error {
	if cfg.Dir == "" {
		cfg.Dir = "logs"
	}
	if cfg.Rotation == "" {
		cfg.Rotation = "size"
	}
	if cfg.MaxAgeDays == 0 {
		cfg.MaxAgeDays = 30
	}
	if cfg.MaxSizeMB == 0 {
		cfg.MaxSizeMB = 100
	}
	if err := os.MkdirAll(cfg.Dir, 0o750); err != nil {
		return err
	}

	apiLogger = newSinkLogger("api", cfg)
	consumerLogger = newSinkLogger("consumer", cfg)
	thirdPartyLogger = newSinkLogger("thirdparty", cfg)
	traceLogger = newSinkLogger("trace", cfg)
	return nil
}

func newSinkLogger(name string, cfg SinkConfig) zerolog.Logger {
	var w io.Writer
	switch cfg.Rotation {
	case "daily":
		w = newDailyWriter(cfg.Dir, name, cfg.MaxAgeDays)
	default: // "size"
		w = &lumberjack.Logger{
			Filename: filepath.Join(cfg.Dir, name+".log"),
			MaxSize:  cfg.MaxSizeMB,
			MaxAge:   cfg.MaxAgeDays,
			Compress: true,
		}
	}
	if cfg.Console {
		w = io.MultiWriter(w, os.Stdout)
	}
	return zerolog.New(w).With().Timestamp().Str("log", name).Logger()
}

// API returns the logger that writes to api.log. The pointer targets the package
// variable, so it stays valid (and live) across SetupSinks reconfiguration.
func API() *zerolog.Logger { return &apiLogger }

// Consumer returns the logger that writes to consumer.log.
func Consumer() *zerolog.Logger { return &consumerLogger }

// ThirdParty returns the logger that writes to thirdparty.log.
func ThirdParty() *zerolog.Logger { return &thirdPartyLogger }

// Trace returns the logger that writes to trace.log.
func Trace() *zerolog.Logger { return &traceLogger }

// ── daily rotating writer ─────────────────────────────────────────────────────

// dailyWriter writes to "<dir>/<base>-YYYY-MM-DD.log", switching files at midnight
// and deleting files older than maxAge days on each rotation.
type dailyWriter struct {
	mu      sync.Mutex
	dir     string
	base    string
	maxAge  int
	curDate string
	f       *os.File
}

func newDailyWriter(dir, base string, maxAge int) *dailyWriter {
	return &dailyWriter{dir: dir, base: base, maxAge: maxAge}
}

func (w *dailyWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	today := time.Now().Format("2006-01-02")
	if w.f == nil || today != w.curDate {
		if err := w.rotate(today); err != nil {
			return 0, err
		}
	}
	return w.f.Write(p)
}

func (w *dailyWriter) rotate(today string) error {
	if w.f != nil {
		w.f.Close() //nolint:errcheck
	}
	path := filepath.Join(w.dir, w.base+"-"+today+".log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600) // #nosec G304 -- path is dir+datestamp, no user input
	if err != nil {
		return err
	}
	w.f = f
	w.curDate = today
	w.cleanup()
	return nil
}

// cleanup removes "<base>-*.log" files older than maxAge days. Best-effort.
func (w *dailyWriter) cleanup() {
	if w.maxAge <= 0 {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -w.maxAge)
	entries, _ := filepath.Glob(filepath.Join(w.dir, w.base+"-*.log"))
	for _, e := range entries {
		base := filepath.Base(e)
		ds := strings.TrimSuffix(strings.TrimPrefix(base, w.base+"-"), ".log")
		if t, err := time.Parse("2006-01-02", ds); err == nil && t.Before(cutoff) {
			os.Remove(e) //nolint:errcheck
		}
	}
}
