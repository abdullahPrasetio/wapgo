package rabbitmq

import (
	"context"
	"fmt"
	"io"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"

	applogger "github.com/abdullahPrasetio/wapgo/pkg/logger"
)

// Message is the unit of data published or received via RabbitMQ.
type Message struct {
	RoutingKey string
	Body       []byte
	RequestID  string
}

// publishChan is the subset of amqp.Channel used by Publisher (enables mocking).
type publishChan interface {
	ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error
	PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
	Close() error
}

// amqpConnForPub wraps an AMQP connection for use in NewPublisher (enables mocking).
type amqpConnForPub interface {
	Channel() (publishChan, error)
	Close() error
}

type realAMQPPubConn struct{ c *amqp.Connection }

func (r *realAMQPPubConn) Channel() (publishChan, error) { return r.c.Channel() }
func (r *realAMQPPubConn) Close() error                  { return r.c.Close() }

// Publisher sends messages to a RabbitMQ topic exchange.
type Publisher struct {
	ch       publishChan
	conn     io.Closer // nil when constructed via newPublisherFrom (test helper)
	exchange string
	log      zerolog.Logger
}

// NewPublisher dials the AMQP broker, declares the topic exchange, and returns a Publisher.
// dsn example: "amqp://guest:guest@localhost:5672/"
func NewPublisher(dsn, exchange string, log zerolog.Logger) (*Publisher, error) {
	conn, err := amqp.Dial(dsn)
	if err != nil {
		return nil, fmt.Errorf("rabbitmq dial: %w", err)
	}
	return newPublisherWithConn(&realAMQPPubConn{conn}, exchange, log)
}

// newPublisherWithConn creates a Publisher from an injectable AMQP connection (used in tests).
func newPublisherWithConn(conn amqpConnForPub, exchange string, log zerolog.Logger) (*Publisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		conn.Close() //nolint:errcheck
		return nil, fmt.Errorf("rabbitmq channel: %w", err)
	}
	p := newPublisherFrom(ch, conn, exchange, log)
	if err := ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		p.Close() //nolint:errcheck
		return nil, fmt.Errorf("rabbitmq exchange declare: %w", err)
	}
	return p, nil
}

// newPublisherFrom creates a Publisher with a pre-built channel (used in tests only).
func newPublisherFrom(ch publishChan, conn io.Closer, exchange string, log zerolog.Logger) *Publisher {
	return &Publisher{ch: ch, conn: conn, exchange: exchange, log: log}
}

// Publish sends msg to the exchange using the given routing key.
// The request-id from ctx is attached as a header.
func (p *Publisher) Publish(ctx context.Context, msg Message) error {
	rid := msg.RequestID
	if rid == "" {
		rid = applogger.RequestIDFromContext(ctx)
	}
	headers := amqp.Table{"x-request-id": rid}
	// Inject OTel trace context so consumers can continue the distributed trace.
	otel.GetTextMapPropagator().Inject(ctx, amqpTableCarrier(headers))

	err := p.ch.PublishWithContext(ctx, p.exchange, msg.RoutingKey, false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			Body:         msg.Body,
			Headers:      headers,
		},
	)
	if err != nil {
		return fmt.Errorf("rabbitmq publish exchange=%s routing_key=%s: %w", p.exchange, msg.RoutingKey, err)
	}
	return nil
}

// Close shuts down the channel and underlying connection.
func (p *Publisher) Close() error {
	if err := p.ch.Close(); err != nil {
		return err
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}
