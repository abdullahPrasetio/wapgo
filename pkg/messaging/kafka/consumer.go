package kafka

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"
	kafkago "github.com/segmentio/kafka-go"

	applogger "github.com/abdullahPrasetio/wapgo/pkg/logger"
)

// reader is the subset of kafkago.Reader used by Consumer (enables mocking in tests).
type reader interface {
	FetchMessage(ctx context.Context) (kafkago.Message, error)
	CommitMessages(ctx context.Context, msgs ...kafkago.Message) error
	Close() error
}

// HandlerFunc processes a single Kafka message.
// Return non-nil to skip committing the offset (message will be re-delivered).
type HandlerFunc func(ctx context.Context, msg Message) error

// Consumer reads messages from a single Kafka topic in a consumer group.
type Consumer struct {
	r   reader
	log zerolog.Logger
}

// NewConsumer creates a Consumer for one topic within a consumer group.
// brokers is a comma-separated list of host:port addresses.
func NewConsumer(brokers, groupID, topic string, log zerolog.Logger) *Consumer {
	addrs := strings.Split(brokers, ",")
	r := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:        addrs,
		GroupID:        groupID,
		Topic:          topic,
		MinBytes:       10e3,
		MaxBytes:       10e6,
		MaxWait:        500 * time.Millisecond,
		StartOffset:    kafkago.LastOffset,
		CommitInterval: 0, // manual commit after handler succeeds
	})
	return &Consumer{r: r, log: log}
}

// Start blocks and calls handler for every message.
// Stops gracefully when ctx is cancelled.
// A handler error causes the message offset to be skipped (not committed).
func (c *Consumer) Start(ctx context.Context, handler HandlerFunc) error {
	for {
		m, err := c.r.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil // graceful shutdown
			}
			c.log.Error().Err(err).Msg("kafka fetch message failed")
			continue
		}

		msgCtx := applogger.WithRequestID(ctx, extractRequestID(m))

		if err := handler(msgCtx, Message{
			Topic:     m.Topic,
			Key:       m.Key,
			Value:     m.Value,
			RequestID: applogger.RequestIDFromContext(msgCtx),
		}); err != nil {
			c.log.Error().Err(err).Str("topic", m.Topic).Msg("kafka handler error, skipping commit")
			continue
		}

		if err := c.r.CommitMessages(ctx, m); err != nil {
			c.log.Error().Err(err).Msg("kafka commit messages failed")
		}
	}
}

// Close shuts down the consumer reader.
func (c *Consumer) Close() error {
	return c.r.Close()
}

// kafkaConnCloser is the subset of kafkago.Conn used by the health check (enables mocking).
type kafkaConnCloser interface {
	Close() error
}

// HealthCheck returns a probe function that dials the first broker and closes immediately.
// Compatible with handler.Checker (returns "ok", "down", or "not_configured").
func HealthCheck(brokers string) func(ctx context.Context) string {
	return healthCheckWithDialer(brokers, func(ctx context.Context, addr string) (kafkaConnCloser, error) {
		return kafkago.DialContext(ctx, "tcp", addr)
	})
}

func healthCheckWithDialer(brokers string, dial func(context.Context, string) (kafkaConnCloser, error)) func(ctx context.Context) string {
	return func(ctx context.Context) string {
		if strings.TrimSpace(brokers) == "" {
			return "not_configured"
		}
		addr := strings.TrimSpace(strings.SplitN(brokers, ",", 2)[0])
		conn, err := dial(ctx, addr)
		if err != nil {
			return fmt.Sprintf("down: %v", err)
		}
		conn.Close()
		return "ok"
	}
}

func extractRequestID(m kafkago.Message) string {
	for _, h := range m.Headers {
		if h.Key == "x-request-id" {
			return string(h.Value)
		}
	}
	return ""
}
