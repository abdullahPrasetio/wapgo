package kafka

import (
	"context"
	"errors"
	"testing"

	"github.com/rs/zerolog"
	kafkago "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	applogger "github.com/abdullahPrasetio/wapgo/pkg/logger"
)

// mockWriter implements the writer interface.
type mockWriter struct {
	err      error
	received []kafkago.Message
}

func (m *mockWriter) WriteMessages(_ context.Context, msgs ...kafkago.Message) error {
	m.received = append(m.received, msgs...)
	return m.err
}
func (m *mockWriter) Close() error { return m.err }

// ─────────────────────────────────────────────────────────────────────────────

func TestPublish_Success(t *testing.T) {
	mw := &mockWriter{}
	p := &Producer{w: mw, log: zerolog.Nop()}

	err := p.Publish(context.Background(), Message{
		Topic:     "orders",
		Key:       []byte("k1"),
		Value:     []byte(`{"id":1}`),
		RequestID: "req-abc",
	})
	require.NoError(t, err)
	require.Len(t, mw.received, 1)

	msg := mw.received[0]
	assert.Equal(t, "orders", msg.Topic)
	assert.Equal(t, []byte("k1"), msg.Key)
	assert.Equal(t, []byte(`{"id":1}`), msg.Value)

	// x-request-id header must match
	var gotRID string
	for _, h := range msg.Headers {
		if h.Key == "x-request-id" {
			gotRID = string(h.Value)
		}
	}
	assert.Equal(t, "req-abc", gotRID)
}

func TestPublish_RequestIDFallsBackToContext(t *testing.T) {
	mw := &mockWriter{}
	p := &Producer{w: mw, log: zerolog.Nop()}

	ctx := applogger.WithRequestID(context.Background(), "ctx-rid")
	require.NoError(t, p.Publish(ctx, Message{Topic: "t", Value: []byte("{}")}))

	var gotRID string
	for _, h := range mw.received[0].Headers {
		if h.Key == "x-request-id" {
			gotRID = string(h.Value)
		}
	}
	assert.Equal(t, "ctx-rid", gotRID)
}

func TestPublish_EmptyRequestID(t *testing.T) {
	mw := &mockWriter{}
	p := &Producer{w: mw, log: zerolog.Nop()}

	require.NoError(t, p.Publish(context.Background(), Message{Topic: "t", Value: []byte("{}")}))
	// x-request-id header should be empty string (not missing)
	var gotRID string
	found := false
	for _, h := range mw.received[0].Headers {
		if h.Key == "x-request-id" {
			gotRID = string(h.Value)
			found = true
		}
	}
	assert.True(t, found)
	assert.Empty(t, gotRID)
}

func TestPublish_WriterError(t *testing.T) {
	mw := &mockWriter{err: errors.New("broker unavailable")}
	p := &Producer{w: mw, log: zerolog.Nop()}

	err := p.Publish(context.Background(), Message{Topic: "t", Value: []byte("{}")})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "kafka publish topic=t")
}

func TestPublish_ContentTypeHeader(t *testing.T) {
	mw := &mockWriter{}
	p := &Producer{w: mw, log: zerolog.Nop()}
	require.NoError(t, p.Publish(context.Background(), Message{Topic: "t", Value: []byte("{}")}))

	var ct string
	for _, h := range mw.received[0].Headers {
		if h.Key == "content-type" {
			ct = string(h.Value)
		}
	}
	assert.Equal(t, "application/json", ct)
}

func TestProducerClose(t *testing.T) {
	p := &Producer{w: &mockWriter{}, log: zerolog.Nop()}
	assert.NoError(t, p.Close())
}

func TestProducerClose_Error(t *testing.T) {
	p := &Producer{w: &mockWriter{err: errors.New("close failed")}, log: zerolog.Nop()}
	assert.Error(t, p.Close())
}
