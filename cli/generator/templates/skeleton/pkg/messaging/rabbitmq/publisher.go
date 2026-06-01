//go:build ignore

package rabbitmq

import (
	"context"
	"fmt"
	"io"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"

	applogger "github.com/abdullahPrasetio/wapgo/pkg/logger"
)

type Message struct {
	RoutingKey string
	Body       []byte
	RequestID  string
}

type publishChan interface {
	ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error
	PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
	Close() error
}

type amqpConnForPub interface {
	Channel() (publishChan, error)
	Close() error
}

type realAMQPPubConn struct{ c *amqp.Connection }

func (r *realAMQPPubConn) Channel() (publishChan, error) { return r.c.Channel() }
func (r *realAMQPPubConn) Close() error                  { return r.c.Close() }

type Publisher struct {
	ch       publishChan
	conn     io.Closer
	exchange string
	log      zerolog.Logger
}

func NewPublisher(dsn, exchange string, log zerolog.Logger) (*Publisher, error) {
	conn, err := amqp.Dial(dsn)
	if err != nil {
		return nil, fmt.Errorf("rabbitmq dial: %w", err)
	}
	return newPublisherWithConn(&realAMQPPubConn{conn}, exchange, log)
}

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

func newPublisherFrom(ch publishChan, conn io.Closer, exchange string, log zerolog.Logger) *Publisher {
	return &Publisher{ch: ch, conn: conn, exchange: exchange, log: log}
}

func (p *Publisher) Publish(ctx context.Context, msg Message) error {
	rid := msg.RequestID
	if rid == "" {
		rid = applogger.RequestIDFromContext(ctx)
	}
	err := p.ch.PublishWithContext(ctx, p.exchange, msg.RoutingKey, false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			Body:         msg.Body,
			Headers: amqp.Table{
				"x-request-id": rid,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("rabbitmq publish exchange=%s routing_key=%s: %w", p.exchange, msg.RoutingKey, err)
	}
	return nil
}

func (p *Publisher) Close() error {
	if err := p.ch.Close(); err != nil {
		return err
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}
