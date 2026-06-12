package auditlog

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func captureLogger(buf *bytes.Buffer) *Logger {
	zl := zerolog.New(buf).With().Timestamp().Logger()
	return New(zl)
}

func TestLog_WritesEntry(t *testing.T) {
	var buf bytes.Buffer
	l := captureLogger(&buf)

	l.Log(context.Background(), Entry{
		ActorID:  "user-1",
		Action:   ActionCreate,
		Entity:   "user",
		EntityID: "entity-1",
	})

	var m map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("invalid JSON output: %v\nraw: %s", err, buf.String())
	}
	if m["actor_id"] != "user-1" {
		t.Errorf("actor_id: want %q got %q", "user-1", m["actor_id"])
	}
	if m["action"] != "create" {
		t.Errorf("action: want %q got %q", "create", m["action"])
	}
}

func TestLog_TimestampAutoFilled(t *testing.T) {
	var buf bytes.Buffer
	l := captureLogger(&buf)

	before := time.Now().UTC().Truncate(time.Second)
	l.Log(context.Background(), Entry{ActorID: "x", Action: ActionLogin, Entity: "session", EntityID: "s1"})
	after := time.Now().UTC().Add(time.Second)

	var m map[string]interface{}
	_ = json.Unmarshal(buf.Bytes(), &m)

	tsStr, ok := m["timestamp"].(string)
	if !ok {
		t.Fatalf("timestamp missing or not a string: %v", m["timestamp"])
	}
	// zerolog uses RFC3339 (second precision) by default.
	ts, err := time.Parse(time.RFC3339, tsStr)
	if err != nil {
		// Try nanosecond precision as fallback.
		ts, err = time.Parse(time.RFC3339Nano, tsStr)
		if err != nil {
			t.Fatalf("parse timestamp %q: %v", tsStr, err)
		}
	}
	if ts.Before(before) || ts.After(after) {
		t.Errorf("timestamp %v not in [%v, %v]", ts, before, after)
	}
}

func TestLog_WithRequestID(t *testing.T) {
	var buf bytes.Buffer
	l := captureLogger(&buf)

	l.Log(context.Background(), Entry{
		ActorID:   "u1",
		Action:    ActionDelete,
		Entity:    "order",
		EntityID:  "o99",
		RequestID: "req-abc",
	})

	var m map[string]interface{}
	_ = json.Unmarshal(buf.Bytes(), &m)
	if m["request_id"] != "req-abc" {
		t.Errorf("request_id: want %q got %q", "req-abc", m["request_id"])
	}
}

func TestWithAuditLogger_FromContext(t *testing.T) {
	l := NewDefault()
	ctx := WithAuditLogger(context.Background(), l)
	got := FromContext(ctx)
	if got != l {
		t.Error("expected same Logger from context")
	}
}

func TestFromContext_Nil(t *testing.T) {
	got := FromContext(context.Background())
	if got != nil {
		t.Error("expected nil when no logger in context")
	}
}

func TestLog_AllActions(t *testing.T) {
	actions := []Action{ActionCreate, ActionUpdate, ActionDelete, ActionLogin, ActionLogout}
	for _, a := range actions {
		var buf bytes.Buffer
		l := captureLogger(&buf)
		l.Log(context.Background(), Entry{ActorID: "u", Action: a, Entity: "e", EntityID: "1"})
		if buf.Len() == 0 {
			t.Errorf("no output for action %q", a)
		}
	}
}
