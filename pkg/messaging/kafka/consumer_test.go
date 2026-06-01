package kafka

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rs/zerolog"
	kafkago "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// chanReader is a mock reader backed by a channel — naturally context-aware.
type chanReader struct {
	msgs      chan kafkago.Message
	commitErr error
	committed []kafkago.Message
	closed    bool
}

func newChanReader(msgs ...kafkago.Message) *chanReader {
	ch := make(chan kafkago.Message, len(msgs))
	for _, m := range msgs {
		ch <- m
	}
	return &chanReader{msgs: ch}
}

func (r *chanReader) FetchMessage(ctx context.Context) (kafkago.Message, error) {
	select {
	case msg, ok := <-r.msgs:
		if !ok {
			<-ctx.Done()
			return kafkago.Message{}, ctx.Err()
		}
		return msg, nil
	case <-ctx.Done():
		return kafkago.Message{}, ctx.Err()
	}
}

func (r *chanReader) CommitMessages(_ context.Context, msgs ...kafkago.Message) error {
	r.committed = append(r.committed, msgs...)
	return r.commitErr
}

func (r *chanReader) Close() error {
	r.closed = true
	return nil
}

// errReader returns a single non-fatal error then blocks until ctx is done.
type errReader struct {
	err  error
	once bool
}

func (e *errReader) FetchMessage(ctx context.Context) (kafkago.Message, error) {
	if !e.once {
		e.once = true
		return kafkago.Message{}, e.err
	}
	<-ctx.Done()
	return kafkago.Message{}, ctx.Err()
}

func (e *errReader) CommitMessages(_ context.Context, _ ...kafkago.Message) error { return nil }
func (e *errReader) Close() error                                                  { return nil }

// ─────────────────────────────────────────────────────────────────────────────

func TestConsumer_GracefulShutdown(t *testing.T) {
	r := newChanReader() // no messages — will block until ctx done
	c := &Consumer{r: r, log: zerolog.Nop()}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := c.Start(ctx, func(_ context.Context, _ Message) error { return nil })
	assert.NoError(t, err)
}

func TestConsumer_HandlerCalled_MessageCommitted(t *testing.T) {
	msg := kafkago.Message{
		Topic: "events",
		Key:   []byte("k"),
		Value: []byte(`{"x":1}`),
		Headers: []kafkago.Header{
			{Key: "x-request-id", Value: []byte("rid-123")},
		},
	}
	r := newChanReader(msg)
	c := &Consumer{r: r, log: zerolog.Nop()}

	var got Message
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	c.Start(ctx, func(_ context.Context, m Message) error { //nolint:errcheck
		got = m
		cancel()
		return nil
	})

	assert.Equal(t, "events", got.Topic)
	assert.Equal(t, []byte(`{"x":1}`), got.Value)
	assert.Equal(t, "rid-123", got.RequestID)
	assert.Len(t, r.committed, 1)
}

func TestConsumer_HandlerError_NoCommit(t *testing.T) {
	r := newChanReader(kafkago.Message{Topic: "t", Value: []byte("{}")})
	c := &Consumer{r: r, log: zerolog.Nop()}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	c.Start(ctx, func(_ context.Context, _ Message) error { //nolint:errcheck
		cancel()
		return errors.New("handler failed")
	})

	assert.Empty(t, r.committed)
}

func TestConsumer_CommitError_ContinuesRunning(t *testing.T) {
	r := newChanReader(kafkago.Message{Topic: "t", Value: []byte("{}")})
	r.commitErr = errors.New("commit failed")
	c := &Consumer{r: r, log: zerolog.Nop()}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// Should not return an error even if commit fails
	c.Start(ctx, func(_ context.Context, _ Message) error { //nolint:errcheck
		cancel()
		return nil
	})
}

func TestConsumer_FetchError_LogsAndContinues(t *testing.T) {
	r := &errReader{err: errors.New("network reset")}
	c := &Consumer{r: r, log: zerolog.Nop()}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := c.Start(ctx, func(_ context.Context, _ Message) error { return nil })
	assert.NoError(t, err) // non-ctx error should be logged, not returned
}

func TestConsumer_Close(t *testing.T) {
	r := newChanReader()
	c := &Consumer{r: r, log: zerolog.Nop()}
	require.NoError(t, c.Close())
	assert.True(t, r.closed)
}

func TestExtractRequestID_Present(t *testing.T) {
	m := kafkago.Message{
		Headers: []kafkago.Header{{Key: "x-request-id", Value: []byte("abc")}},
	}
	assert.Equal(t, "abc", extractRequestID(m))
}

func TestExtractRequestID_Missing(t *testing.T) {
	assert.Empty(t, extractRequestID(kafkago.Message{}))
}

func TestHealthCheck_EmptyBrokers(t *testing.T) {
	fn := HealthCheck("")
	assert.Equal(t, "not_configured", fn(context.Background()))
}

func TestHealthCheck_WhitespaceBrokers(t *testing.T) {
	fn := HealthCheck("   ")
	assert.Equal(t, "not_configured", fn(context.Background()))
}

func TestHealthCheck_UnreachableBroker(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	fn := HealthCheck("127.0.0.1:59990") // nothing listening
	result := fn(ctx)
	assert.NotEqual(t, "ok", result)
}

func TestHealthCheck_Ok(t *testing.T) {
	conn := &fakeConn{}
	fn := healthCheckWithDialer("localhost:9092", func(_ context.Context, _ string) (kafkaConnCloser, error) {
		return conn, nil
	})
	assert.Equal(t, "ok", fn(context.Background()))
	assert.True(t, conn.closed)
}

func TestHealthCheck_DialError(t *testing.T) {
	fn := healthCheckWithDialer("localhost:9092", func(_ context.Context, _ string) (kafkaConnCloser, error) {
		return nil, errors.New("refused")
	})
	assert.Contains(t, fn(context.Background()), "down")
}

type fakeConn struct{ closed bool }

func (f *fakeConn) Close() error { f.closed = true; return nil }
