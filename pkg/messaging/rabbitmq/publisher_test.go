package rabbitmq

import (
	"context"
	"errors"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	applogger "github.com/abdullahPrasetio/wapgo/pkg/logger"
)

// mockPublishChan implements publishChan.
type mockPublishChan struct {
	publishErr    error
	declareErr    error
	closeErr      error
	publishedMsgs []amqp.Publishing
	publishedKeys []string
}

func (m *mockPublishChan) ExchangeDeclare(_, _ string, _, _, _, _ bool, _ amqp.Table) error {
	return m.declareErr
}
func (m *mockPublishChan) PublishWithContext(_ context.Context, _, key string, _, _ bool, msg amqp.Publishing) error {
	m.publishedKeys = append(m.publishedKeys, key)
	m.publishedMsgs = append(m.publishedMsgs, msg)
	return m.publishErr
}
func (m *mockPublishChan) Close() error { return m.closeErr }

// ── Publish ───────────────────────────────────────────────────────────────────

func TestPublisher_Publish_Success(t *testing.T) {
	ch := &mockPublishChan{}
	p := newPublisherFrom(ch, nil, "myexchange", zerolog.Nop())

	err := p.Publish(context.Background(), Message{
		RoutingKey: "user.created",
		Body:       []byte(`{"id":"1"}`),
		RequestID:  "req-xyz",
	})
	require.NoError(t, err)
	require.Len(t, ch.publishedMsgs, 1)

	msg := ch.publishedMsgs[0]
	assert.Equal(t, []byte(`{"id":"1"}`), msg.Body)
	assert.Equal(t, "application/json", msg.ContentType)
	assert.Equal(t, "user.created", ch.publishedKeys[0])
	assert.Equal(t, "req-xyz", msg.Headers["x-request-id"])
}

func TestPublisher_Publish_RequestIDFromContext(t *testing.T) {
	ch := &mockPublishChan{}
	p := newPublisherFrom(ch, nil, "ex", zerolog.Nop())

	ctx := applogger.WithRequestID(context.Background(), "ctx-rid")
	require.NoError(t, p.Publish(ctx, Message{RoutingKey: "rk", Body: []byte("{}")}))

	assert.Equal(t, "ctx-rid", ch.publishedMsgs[0].Headers["x-request-id"])
}

func TestPublisher_Publish_PersistentDelivery(t *testing.T) {
	ch := &mockPublishChan{}
	p := newPublisherFrom(ch, nil, "ex", zerolog.Nop())
	require.NoError(t, p.Publish(context.Background(), Message{RoutingKey: "rk", Body: []byte("{}")}))
	assert.Equal(t, uint8(amqp.Persistent), ch.publishedMsgs[0].DeliveryMode)
}

func TestPublisher_Publish_Error_NoConn(t *testing.T) {
	// Without a Connection, retry is skipped and the original error is returned.
	ch := &mockPublishChan{publishErr: errors.New("channel closed")}
	p := newPublisherFrom(ch, nil, "ex", zerolog.Nop())
	err := p.Publish(context.Background(), Message{RoutingKey: "rk", Body: []byte("{}")})
	require.Error(t, err)
}

// ── Close ─────────────────────────────────────────────────────────────────────

func TestPublisher_Close(t *testing.T) {
	p := newPublisherFrom(&mockPublishChan{}, nil, "ex", zerolog.Nop())
	assert.NoError(t, p.Close())
}

func TestPublisher_Close_ChanError(t *testing.T) {
	ch := &mockPublishChan{closeErr: errors.New("close failed")}
	p := newPublisherFrom(ch, nil, "ex", zerolog.Nop())
	assert.Error(t, p.Close())
}

// ── NewConnection bad DSN ─────────────────────────────────────────────────────

func TestNewConnection_BadDSN(t *testing.T) {
	_, err := NewConnection("amqp://bad-host:5672/", zerolog.Nop())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rabbitmq dial")
}
