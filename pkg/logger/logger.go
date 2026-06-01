package logger

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

// ctxKey is an unexported type for context keys in this package.
type ctxKey int

const requestIDKey ctxKey = iota

// Setup configures the global zerolog logger.
// JSON output in production, pretty console in development.
// Optionally writes to a rotating file.
func Setup(env, level, filePath string, toFile bool, serviceName string) {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)
	zerolog.TimeFieldFormat = time.RFC3339

	var writers []io.Writer

	if env == "production" {
		writers = append(writers, os.Stdout)
	} else {
		writers = append(writers, zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		})
	}

	if toFile && filePath != "" {
		if err := os.MkdirAll(filepath.Dir(filePath), 0750); err == nil {
			writers = append(writers, &lumberjack.Logger{
				Filename:   filePath,
				MaxSize:    100, // MB per file
				MaxBackups: 7,
				MaxAge:     30, // days
				Compress:   true,
			})
		}
	}

	mw := io.MultiWriter(writers...)
	log.Logger = zerolog.New(mw).
		With().
		Timestamp().
		Str("service", serviceName).
		Logger()
}

// WithRequestID returns a new context carrying the given request ID.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// RequestIDFromContext extracts the request ID from context, or returns empty string.
func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// FromContext returns a zerolog.Logger enriched with the request-id from context.
func FromContext(ctx context.Context) zerolog.Logger {
	if rid := RequestIDFromContext(ctx); rid != "" {
		return log.With().Str("request_id", rid).Logger()
	}
	return log.Logger
}
