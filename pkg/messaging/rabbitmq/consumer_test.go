package rabbitmq

import (
	"context"
	"errors"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── mock infrastructure ───────────────────────────────────────────────────────

// callConsumeChan is a flexible mock that returns per-call errors from slices.
// The Nth call takes from the slice at index N-1; if the slice is exhausted the
// zero value (nil / empty) is returned.
type callConsumeChan struct {
	exDeclareReturns []error
	qDeclareReturns  []error
	qBindReturns     []error
	consumeErr       error
	closeErr         error
	deliveries       chan amqp.Delivery

	exDeclareN int
	qDeclareN  int
	qBindN     int
}

func newCallChan() *callConsumeChan {
	return &callConsumeChan{deliveries: make(chan amqp.Delivery, 8)}
}

func (m *callConsumeChan) next(slice []error, n *int) error {
	i := *n
	*n++
	if i < len(slice) {
		return slice[i]
	}
	return nil
}

func (m *callConsumeChan) ExchangeDeclare(_, _ string, _, _, _, _ bool, _ amqp.Table) error {
	return m.next(m.exDeclareReturns, &m.exDeclareN)
}
func (m *callConsumeChan) QueueDeclare(_ string, _, _, _, _ bool, _ amqp.Table) (amqp.Queue, error) {
	return amqp.Queue{}, m.next(m.qDeclareReturns, &m.qDeclareN)
}
func (m *callConsumeChan) QueueBind(_, _, _ string, _ bool, _ amqp.Table) error {
	return m.next(m.qBindReturns, &m.qBindN)
}
func (m *callConsumeChan) Consume(_, _ string, _, _, _, _ bool, _ amqp.Table) (<-chan amqp.Delivery, error) {
	if m.consumeErr != nil {
		return nil, m.consumeErr
	}
	return m.deliveries, nil
}
func (m *callConsumeChan) Close() error { return m.closeErr }

// mockAck tracks Ack/Nack calls on a delivery.
// doneCh (optional) is closed/sent-to after Ack or Nack so goroutine-based tests
// can synchronize without a data race on the acked/nacked fields.
type mockAck struct {
	acked  bool
	nacked bool
	doneCh chan struct{}
}

func newMockAck() *mockAck { return &mockAck{doneCh: make(chan struct{}, 1)} }

func (a *mockAck) Ack(_ uint64, _ bool) error {
	a.acked = true
	if a.doneCh != nil {
		a.doneCh <- struct{}{}
	}
	return nil
}
func (a *mockAck) Nack(_ uint64, _, _ bool) error {
	a.nacked = true
	if a.doneCh != nil {
		a.doneCh <- struct{}{}
	}
	return nil
}
func (a *mockAck) Reject(_ uint64, _ bool) error { return nil }

// mockCloser tracks Close calls.
type mockCloser struct {
	closed   bool
	closeErr error
}

func (m *mockCloser) Close() error { m.closed = true; return m.closeErr }

// ── declareExchanges ──────────────────────────────────────────────────────────

func TestDeclareExchanges_MainFails(t *testing.T) {
	ch := newCallChan()
	ch.exDeclareReturns = []error{errors.New("main declare failed")}
	c := newConsumerFrom(ch, nil, "ex", zerolog.Nop())
	err := c.declareExchanges()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "main exchange declare")
}

func TestDeclareExchanges_DLXFails(t *testing.T) {
	ch := newCallChan()
	// First call (main exchange) succeeds, second call (DLX) fails.
	ch.exDeclareReturns = []error{nil, errors.New("dlx declare failed")}
	c := newConsumerFrom(ch, nil, "ex", zerolog.Nop())
	err := c.declareExchanges()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dlx declare")
}

func TestDeclareExchanges_Success(t *testing.T) {
	ch := newCallChan()
	c := newConsumerFrom(ch, nil, "ex", zerolog.Nop())
	assert.NoError(t, c.declareExchanges())
}

// ── Subscribe ─────────────────────────────────────────────────────────────────

func TestSubscribe_DLQDeclareError(t *testing.T) {
	ch := newCallChan()
	ch.qDeclareReturns = []error{errors.New("dlq declare failed")}
	c := newConsumerFrom(ch, nil, "ex", zerolog.Nop())
	err := c.Subscribe("q", "rk", func(_ context.Context, _ Message) error { return nil })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dlq declare")
}

func TestSubscribe_DLQBindError(t *testing.T) {
	ch := newCallChan()
	ch.qBindReturns = []error{errors.New("dlq bind failed")}
	c := newConsumerFrom(ch, nil, "ex", zerolog.Nop())
	err := c.Subscribe("q", "rk", func(_ context.Context, _ Message) error { return nil })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dlq bind")
}

func TestSubscribe_MainQueueDeclareError(t *testing.T) {
	ch := newCallChan()
	// DLQ declare (1st call) succeeds; main queue declare (2nd call) fails.
	ch.qDeclareReturns = []error{nil, errors.New("main queue declare failed")}
	c := newConsumerFrom(ch, nil, "ex", zerolog.Nop())
	err := c.Subscribe("q", "rk", func(_ context.Context, _ Message) error { return nil })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "queue declare")
}

func TestSubscribe_MainQueueBindError(t *testing.T) {
	ch := newCallChan()
	// DLQ bind (1st call) succeeds; main queue bind (2nd call) fails.
	ch.qBindReturns = []error{nil, errors.New("main queue bind failed")}
	c := newConsumerFrom(ch, nil, "ex", zerolog.Nop())
	err := c.Subscribe("q", "rk", func(_ context.Context, _ Message) error { return nil })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "queue bind")
}

func TestSubscribe_ConsumeError(t *testing.T) {
	ch := newCallChan()
	ch.consumeErr = errors.New("consume failed")
	c := newConsumerFrom(ch, nil, "ex", zerolog.Nop())
	err := c.Subscribe("q", "rk", func(_ context.Context, _ Message) error { return nil })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rabbitmq consume")
}

func TestSubscribe_Success(t *testing.T) {
	ch := newCallChan()
	c := newConsumerFrom(ch, nil, "ex", zerolog.Nop())
	assert.NoError(t, c.Subscribe("q", "rk", func(_ context.Context, _ Message) error { return nil }))
}

// ── drain ─────────────────────────────────────────────────────────────────────

// TestDrain_WithMessage verifies the drain goroutine processes a real delivery.
// We synchronize via mockAck.doneCh (written before the channel send returns)
// rather than a separate handler channel, so the race detector is satisfied.
func TestDrain_WithMessage(t *testing.T) {
	ch := newCallChan()
	c := newConsumerFrom(ch, nil, "ex", zerolog.Nop())
	require.NoError(t, c.Subscribe("q", "rk", func(_ context.Context, _ Message) error { return nil }))

	ack := newMockAck()
	ch.deliveries <- amqp.Delivery{
		Acknowledger: ack,
		RoutingKey:   "rk",
		Body:         []byte(`{"x":1}`),
	}

	select {
	case <-ack.doneCh:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("drain goroutine did not process message in time")
	}
	assert.True(t, ack.acked)
}

// ── handle ────────────────────────────────────────────────────────────────────

func TestHandle_Ack(t *testing.T) {
	ack := newMockAck()
	d := amqp.Delivery{
		Acknowledger: ack,
		RoutingKey:   "user.created",
		Body:         []byte(`{"id":"1"}`),
		Headers:      amqp.Table{"x-request-id": "rid-001"},
	}
	c := newConsumerFrom(newCallChan(), nil, "ex", zerolog.Nop())
	c.handle(d, func(_ context.Context, msg Message) error {
		assert.Equal(t, "user.created", msg.RoutingKey)
		assert.Equal(t, "rid-001", msg.RequestID)
		return nil
	})
	assert.True(t, ack.acked)
	assert.False(t, ack.nacked)
}

func TestHandle_Nack_OnError(t *testing.T) {
	ack := newMockAck()
	d := amqp.Delivery{
		Acknowledger: ack,
		RoutingKey:   "rk",
		Body:         []byte("{}"),
	}
	c := newConsumerFrom(newCallChan(), nil, "ex", zerolog.Nop())
	c.handle(d, func(_ context.Context, _ Message) error {
		return errors.New("handler failed")
	})
	assert.False(t, ack.acked)
	assert.True(t, ack.nacked)
}

func TestHandle_NoRequestIDHeader(t *testing.T) {
	ack := newMockAck()
	d := amqp.Delivery{
		Acknowledger: ack,
		RoutingKey:   "rk",
		Body:         []byte("{}"),
		Headers:      amqp.Table{},
	}
	c := newConsumerFrom(newCallChan(), nil, "ex", zerolog.Nop())
	var gotRID string
	c.handle(d, func(_ context.Context, msg Message) error {
		gotRID = msg.RequestID
		return nil
	})
	assert.Empty(t, gotRID)
	assert.True(t, ack.acked)
}

// ── Close ─────────────────────────────────────────────────────────────────────

func TestConsumer_Close(t *testing.T) {
	c := newConsumerFrom(newCallChan(), nil, "ex", zerolog.Nop())
	assert.NoError(t, c.Close())
}

func TestConsumer_Close_Error(t *testing.T) {
	ch := newCallChan()
	ch.closeErr = errors.New("close failed")
	c := newConsumerFrom(ch, nil, "ex", zerolog.Nop())
	assert.Error(t, c.Close())
}

func TestConsumer_Close_WithConn(t *testing.T) {
	conn := &mockCloser{}
	c := newConsumerFrom(newCallChan(), conn, "ex", zerolog.Nop())
	require.NoError(t, c.Close())
	assert.True(t, conn.closed)
}

// ── HealthCheck ───────────────────────────────────────────────────────────────

func TestHealthCheck_Empty(t *testing.T) {
	fn := HealthCheck("")
	assert.Equal(t, "not_configured", fn(context.Background()))
}

func TestHealthCheck_Unreachable(t *testing.T) {
	fn := HealthCheck("amqp://user:pass@127.0.0.1:59989/")
	result := fn(context.Background())
	assert.Contains(t, result, "down")
}

func TestHealthCheck_Ok(t *testing.T) {
	conn := &mockCloser{}
	fn := healthCheckWithDialer("amqp://test", func(_ string) (amqpConnCloser, error) {
		return conn, nil
	})
	assert.Equal(t, "ok", fn(context.Background()))
	assert.True(t, conn.closed)
}

func TestHealthCheck_DialError(t *testing.T) {
	fn := healthCheckWithDialer("amqp://test", func(_ string) (amqpConnCloser, error) {
		return nil, errors.New("refused")
	})
	assert.Contains(t, fn(context.Background()), "down")
}

// ── newConsumerWithConn ───────────────────────────────────────────────────────

// mockConConn implements amqpConnForCon.
type mockConConn struct {
	ch      consumeChan
	chanErr error
	closed  bool
}

func (m *mockConConn) Channel() (consumeChan, error) { return m.ch, m.chanErr }
func (m *mockConConn) Close() error                  { m.closed = true; return nil }

func TestNewConsumerWithConn_ChannelError(t *testing.T) {
	conn := &mockConConn{chanErr: errors.New("channel failed")}
	_, err := newConsumerWithConn(conn, "ex", zerolog.Nop())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rabbitmq channel")
	assert.True(t, conn.closed)
}

func TestNewConsumerWithConn_DeclareError(t *testing.T) {
	ch := newCallChan()
	ch.exDeclareReturns = []error{errors.New("declare failed")}
	conn := &mockConConn{ch: ch}
	_, err := newConsumerWithConn(conn, "ex", zerolog.Nop())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "main exchange declare")
}

func TestNewConsumerWithConn_Success(t *testing.T) {
	conn := &mockConConn{ch: newCallChan()}
	c, err := newConsumerWithConn(conn, "ex", zerolog.Nop())
	require.NoError(t, err)
	assert.NotNil(t, c)
}

// ── NewConsumer ───────────────────────────────────────────────────────────────

func TestNewConsumer_BadDSN(t *testing.T) {
	_, err := NewConsumer("amqp://bad-host:5672/", "ex", zerolog.Nop())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rabbitmq dial")
}
