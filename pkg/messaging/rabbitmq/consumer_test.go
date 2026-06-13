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

// ── mock channel ──────────────────────────────────────────────────────────────

// mockConsumeChan implements consumeChan with per-call error control.
type mockConsumeChan struct {
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

func newMockChan() *mockConsumeChan {
	return &mockConsumeChan{deliveries: make(chan amqp.Delivery, 8)}
}

func (m *mockConsumeChan) next(slice []error, n *int) error {
	i := *n
	*n++
	if i < len(slice) {
		return slice[i]
	}
	return nil
}

func (m *mockConsumeChan) ExchangeDeclare(_, _ string, _, _, _, _ bool, _ amqp.Table) error {
	return m.next(m.exDeclareReturns, &m.exDeclareN)
}
func (m *mockConsumeChan) QueueDeclare(_ string, _, _, _, _ bool, _ amqp.Table) (amqp.Queue, error) {
	return amqp.Queue{}, m.next(m.qDeclareReturns, &m.qDeclareN)
}
func (m *mockConsumeChan) QueueBind(_, _, _ string, _ bool, _ amqp.Table) error {
	return m.next(m.qBindReturns, &m.qBindN)
}
func (m *mockConsumeChan) Consume(_, _ string, _, _, _, _ bool, _ amqp.Table) (<-chan amqp.Delivery, error) {
	if m.consumeErr != nil {
		return nil, m.consumeErr
	}
	return m.deliveries, nil
}
func (m *mockConsumeChan) Close() error { return m.closeErr }

// mockAck tracks Ack/Nack on a delivery.
type mockAck struct {
	acked  bool
	nacked bool
	doneCh chan struct{}
}

func newMockAck() *mockAck { return &mockAck{doneCh: make(chan struct{}, 1)} }

func (a *mockAck) Ack(_ uint64, _ bool) error {
	a.acked = true
	a.doneCh <- struct{}{}
	return nil
}
func (a *mockAck) Nack(_ uint64, _, _ bool) error {
	a.nacked = true
	a.doneCh <- struct{}{}
	return nil
}
func (a *mockAck) Reject(_ uint64, _ bool) error { return nil }

// newTestConsumer returns a Consumer wired to a mock channel for unit testing
// internal functions (declareExchanges, declareQueues, handle).
func newTestConsumer(ch consumeChan, exchange string) *Consumer {
	return &Consumer{exchange: exchange, dlx: exchange + ".dlx", log: zerolog.Nop()}
}

// ── declareExchanges ──────────────────────────────────────────────────────────

func TestDeclareExchanges_MainFails(t *testing.T) {
	ch := newMockChan()
	ch.exDeclareReturns = []error{errors.New("main declare failed")}
	c := newTestConsumer(ch, "ex")
	require.Error(t, c.declareExchanges(ch))
}

func TestDeclareExchanges_DLXFails(t *testing.T) {
	ch := newMockChan()
	ch.exDeclareReturns = []error{nil, errors.New("dlx failed")}
	c := newTestConsumer(ch, "ex")
	err := c.declareExchanges(ch)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dlx declare")
}

func TestDeclareExchanges_Success(t *testing.T) {
	ch := newMockChan()
	c := newTestConsumer(ch, "ex")
	assert.NoError(t, c.declareExchanges(ch))
}

// ── declareQueues ─────────────────────────────────────────────────────────────

func TestDeclareQueues_DLQDeclareError(t *testing.T) {
	ch := newMockChan()
	ch.qDeclareReturns = []error{errors.New("dlq declare failed")}
	c := newTestConsumer(ch, "ex")
	err := c.declareQueues(ch, "q", "rk")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dlq declare")
}

func TestDeclareQueues_DLQBindError(t *testing.T) {
	ch := newMockChan()
	ch.qBindReturns = []error{errors.New("dlq bind failed")}
	c := newTestConsumer(ch, "ex")
	err := c.declareQueues(ch, "q", "rk")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dlq bind")
}

func TestDeclareQueues_MainQueueDeclareError(t *testing.T) {
	ch := newMockChan()
	ch.qDeclareReturns = []error{nil, errors.New("main queue declare failed")}
	c := newTestConsumer(ch, "ex")
	err := c.declareQueues(ch, "q", "rk")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "queue declare")
}

func TestDeclareQueues_MainQueueBindError(t *testing.T) {
	ch := newMockChan()
	ch.qBindReturns = []error{nil, errors.New("main bind failed")}
	c := newTestConsumer(ch, "ex")
	err := c.declareQueues(ch, "q", "rk")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "queue bind")
}

func TestDeclareQueues_Success(t *testing.T) {
	ch := newMockChan()
	c := newTestConsumer(ch, "ex")
	assert.NoError(t, c.declareQueues(ch, "q", "rk"))
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
	c := newTestConsumer(newMockChan(), "ex")
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
	d := amqp.Delivery{Acknowledger: ack, RoutingKey: "rk", Body: []byte("{}")}
	c := newTestConsumer(newMockChan(), "ex")
	c.handle(d, func(_ context.Context, _ Message) error {
		return errors.New("handler failed")
	})
	assert.False(t, ack.acked)
	assert.True(t, ack.nacked)
}

func TestHandle_NoRequestIDHeader(t *testing.T) {
	ack := newMockAck()
	d := amqp.Delivery{Acknowledger: ack, RoutingKey: "rk", Body: []byte("{}"), Headers: amqp.Table{}}
	c := newTestConsumer(newMockChan(), "ex")
	var gotRID string
	c.handle(d, func(_ context.Context, msg Message) error {
		gotRID = msg.RequestID
		return nil
	})
	assert.Empty(t, gotRID)
	assert.True(t, ack.acked)
}

// ── Close ─────────────────────────────────────────────────────────────────────

func TestConsumer_Close_IsNoop(t *testing.T) {
	c := newTestConsumer(newMockChan(), "ex")
	assert.NoError(t, c.Close())
}

// ── sleepBackoff ──────────────────────────────────────────────────────────────

func TestSleepBackoff_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	bo := time.Second
	assert.False(t, sleepBackoff(ctx, &bo))
}

func TestSleepBackoff_Doubles(t *testing.T) {
	ctx := context.Background()
	bo := 10 * time.Millisecond
	sleepBackoff(ctx, &bo)
	assert.Equal(t, 20*time.Millisecond, bo)
}

func TestSleepBackoff_CapsAt30s(t *testing.T) {
	ctx := context.Background()
	bo := 20 * time.Second
	sleepBackoff(ctx, &bo)
	assert.Equal(t, 30*time.Second, bo)
}

// ── HealthCheck ───────────────────────────────────────────────────────────────

func TestHealthCheck_Empty(t *testing.T) {
	fn := HealthCheck("")
	assert.Equal(t, "not_configured", fn(context.Background()))
}

func TestHealthCheck_Unreachable(t *testing.T) {
	fn := HealthCheck("amqp://user:pass@127.0.0.1:59989/")
	assert.Contains(t, fn(context.Background()), "down")
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

// ── misc ──────────────────────────────────────────────────────────────────────

// mockCloser is used by HealthCheck tests.
type mockCloser struct {
	closed bool
}

func (m *mockCloser) Close() error { m.closed = true; return nil }

// TestHandle_ProcessesMessage verifies handle dispatches and acks correctly.
func TestHandle_ProcessesMessage(t *testing.T) {
	ack := newMockAck()
	d := amqp.Delivery{
		Acknowledger: ack,
		RoutingKey:   "order.created",
		Body:         []byte(`{"order_id":"42"}`),
	}
	c := newTestConsumer(newMockChan(), "ex")
	c.handle(d, func(_ context.Context, msg Message) error {
		assert.Equal(t, "order.created", msg.RoutingKey)
		return nil
	})
	select {
	case <-ack.doneCh:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("handle did not ack in time")
	}
	assert.True(t, ack.acked)
}
