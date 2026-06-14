package kafka

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"
	kafkago "github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"

	applogger "github.com/abdullahPrasetio/wapgo/pkg/logger"
)

// writer is the subset of kafkago.Writer used by Producer (enables mocking in tests).
type writer interface {
	WriteMessages(ctx context.Context, msgs ...kafkago.Message) error
	Close() error
}

// Message is the unit of data published to a Kafka topic.
type Message struct {
	Topic     string
	Key       []byte
	Value     []byte
	RequestID string // propagated as x-request-id header; falls back to ctx value if empty
}

// Producer sends messages to Kafka topics.
type Producer struct {
	w   writer
	log zerolog.Logger
}

// NewProducer creates a Producer connected to the given brokers.
// brokers is a comma-separated list of host:port addresses.
func NewProducer(brokers string, log zerolog.Logger) *Producer {
	addrs := strings.Split(brokers, ",")
	w := &kafkago.Writer{
		Addr:         kafkago.TCP(addrs...),
		Balancer:     &kafkago.LeastBytes{},
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
		RequiredAcks: kafkago.RequireOne,
		MaxAttempts:  3,
	}
	return &Producer{w: w, log: log}
}

// Publish writes a single message to Kafka.
// The request-id is attached as an x-request-id header; if msg.RequestID is
// empty it falls back to the value stored in ctx by the logger package.
func (p *Producer) Publish(ctx context.Context, msg Message) error {
	rid := msg.RequestID
	if rid == "" {
		rid = applogger.RequestIDFromContext(ctx)
	}
	headers := []kafkago.Header{
		{Key: "x-request-id", Value: []byte(rid)},
		{Key: "content-type", Value: []byte("application/json")},
	}
	// Inject OTel trace context so consumers can continue the distributed trace.
	carrier := &kafkaHeaderCarrier{headers: &headers}
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	km := kafkago.Message{
		Topic:   msg.Topic,
		Key:     msg.Key,
		Value:   msg.Value,
		Time:    time.Now(),
		Headers: headers,
	}
	if err := p.w.WriteMessages(ctx, km); err != nil {
		return fmt.Errorf("kafka publish topic=%s: %w", msg.Topic, err)
	}
	return nil
}

// Close shuts down the underlying writer and flushes any pending messages.
func (p *Producer) Close() error {
	return p.w.Close()
}
