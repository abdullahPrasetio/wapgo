// Package auditlog records who changed what and when.
// It writes structured JSON entries to a zerolog logger; production deployments
// can redirect those entries to a dedicated sink (audit.log, Kafka topic, etc.)
// by wiring a separate zerolog writer in the logger setup.
package auditlog

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Action describes the type of mutation being audited.
type Action string

const (
	ActionCreate Action = "create"
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"
	ActionLogin  Action = "login"
	ActionLogout Action = "logout"
)

// Entry is a single audit record.
type Entry struct {
	Timestamp  time.Time   `json:"timestamp"`
	ActorID    string      `json:"actor_id"`    // user UUID performing the action
	ActorEmail string      `json:"actor_email"` // optional — helps human review
	Action     Action      `json:"action"`
	Entity     string      `json:"entity"`      // e.g. "user", "order"
	EntityID   string      `json:"entity_id"`
	RequestID  string      `json:"request_id,omitempty"`
	Detail     interface{} `json:"detail,omitempty"` // optional extra context
}

type contextKey struct{}

// Logger is the audit trail writer.
// It is intentionally separate from the application logger so audit entries
// can be directed to a different sink without changing the logger setup.
type Logger struct {
	zlog zerolog.Logger
}

// New creates a Logger that writes to the given zerolog.Logger.
// Pass log.Logger to reuse the global logger, or a dedicated instance for a
// separate "audit.log" file.
func New(zlog zerolog.Logger) *Logger {
	return &Logger{zlog: zlog}
}

// NewDefault creates a Logger backed by the global zerolog logger.
func NewDefault() *Logger {
	return &Logger{zlog: log.Logger}
}

// Log writes an audit entry to the underlying logger.
func (l *Logger) Log(ctx context.Context, e Entry) {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}

	ev := l.zlog.Info().
		Str("audit", "true").
		Time("timestamp", e.Timestamp).
		Str("actor_id", e.ActorID).
		Str("action", string(e.Action)).
		Str("entity", e.Entity).
		Str("entity_id", e.EntityID)

	if e.ActorEmail != "" {
		ev = ev.Str("actor_email", e.ActorEmail)
	}
	if e.RequestID != "" {
		ev = ev.Str("request_id", e.RequestID)
	}
	if e.Detail != nil {
		ev = ev.Interface("detail", e.Detail)
	}

	ev.Msg("audit")
}

// WithAuditLogger stores an audit Logger in context for use across layers.
func WithAuditLogger(ctx context.Context, l *Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, l)
}

// FromContext retrieves the audit Logger from context.
// Returns nil when none is set — callers must guard: if l := auditlog.FromContext(ctx); l != nil { ... }
func FromContext(ctx context.Context) *Logger {
	v := ctx.Value(contextKey{})
	if v == nil {
		return nil
	}
	l, _ := v.(*Logger)
	return l
}
