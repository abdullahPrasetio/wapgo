package rabbitmq

import (
	"context"
	"fmt"
	"io"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"

	"github.com/abdullahPrasetio/wapgo/pkg/journal"
	applogger "github.com/abdullahPrasetio/wapgo/pkg/logger"
	"github.com/abdullahPrasetio/wapgo/pkg/observability"
)

// consumeChan is the subset of amqp.Channel used by Consumer (enables mocking).
type consumeChan interface {
	ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error
	QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error)
	QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error
	Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
	Close() error
}

// amqpConnForCon wraps an AMQP connection for use in NewConsumer (enables mocking).
type amqpConnForCon interface {
	Channel() (consumeChan, error)
	Close() error
}

type realAMQPConConn struct{ c *amqp.Connection }

func (r *realAMQPConConn) Channel() (consumeChan, error) { return r.c.Channel() }
func (r *realAMQPConConn) Close() error                  { return r.c.Close() }

// HandlerFunc processes a single RabbitMQ delivery.
// Return non-nil to Nack the message (it will be routed to the DLQ).
type HandlerFunc func(ctx context.Context, msg Message) error

// Consumer receives messages from a RabbitMQ topic exchange with DLQ support.
type Consumer struct {
	ch       consumeChan
	conn     io.Closer
	exchange string
	dlx      string // dead-letter exchange
	log      zerolog.Logger
}

// NewConsumer dials the AMQP broker and declares the topic exchange and its DLX.
func NewConsumer(dsn, exchange string, log zerolog.Logger) (*Consumer, error) {
	conn, err := amqp.Dial(dsn)
	if err != nil {
		return nil, fmt.Errorf("rabbitmq dial: %w", err)
	}
	return newConsumerWithConn(&realAMQPConConn{conn}, exchange, log)
}

// newConsumerWithConn creates a Consumer from an injectable AMQP connection (used in tests).
func newConsumerWithConn(conn amqpConnForCon, exchange string, log zerolog.Logger) (*Consumer, error) {
	ch, err := conn.Channel()
	if err != nil {
		conn.Close() //nolint:errcheck
		return nil, fmt.Errorf("rabbitmq channel: %w", err)
	}
	c := newConsumerFrom(ch, conn, exchange, log)
	if err := c.declareExchanges(); err != nil {
		c.Close() //nolint:errcheck
		return nil, err
	}
	return c, nil
}

// newConsumerFrom creates a Consumer with a pre-built channel (used in tests).
func newConsumerFrom(ch consumeChan, conn io.Closer, exchange string, log zerolog.Logger) *Consumer {
	return &Consumer{
		ch:       ch,
		conn:     conn,
		exchange: exchange,
		dlx:      exchange + ".dlx",
		log:      log,
	}
}

func (c *Consumer) declareExchanges() error {
	if err := c.ch.ExchangeDeclare(c.exchange, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("rabbitmq main exchange declare: %w", err)
	}
	if err := c.ch.ExchangeDeclare(c.dlx, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("rabbitmq dlx declare: %w", err)
	}
	return nil
}

// Subscribe binds queueName to routingKey, sets up the matching DLQ, and
// starts consuming messages in a background goroutine.
// Failed deliveries are Nacked without requeue and routed to the DLQ via DLX.
func (c *Consumer) Subscribe(queueName, routingKey string, handler HandlerFunc) error {
	dlqName := queueName + ".dlq"

	if _, err := c.ch.QueueDeclare(dlqName, true, false, false, false, nil); err != nil {
		return fmt.Errorf("rabbitmq dlq declare: %w", err)
	}
	if err := c.ch.QueueBind(dlqName, routingKey, c.dlx, false, nil); err != nil {
		return fmt.Errorf("rabbitmq dlq bind: %w", err)
	}

	qArgs := amqp.Table{
		"x-dead-letter-exchange":    c.dlx,
		"x-dead-letter-routing-key": routingKey,
	}
	if _, err := c.ch.QueueDeclare(queueName, true, false, false, false, qArgs); err != nil {
		return fmt.Errorf("rabbitmq queue declare: %w", err)
	}
	if err := c.ch.QueueBind(queueName, routingKey, c.exchange, false, nil); err != nil {
		return fmt.Errorf("rabbitmq queue bind: %w", err)
	}

	deliveries, err := c.ch.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("rabbitmq consume: %w", err)
	}

	go c.drain(deliveries, handler)
	return nil
}

func (c *Consumer) drain(deliveries <-chan amqp.Delivery, handler HandlerFunc) {
	for d := range deliveries {
		c.handle(d, handler)
	}
}

func (c *Consumer) handle(d amqp.Delivery, handler HandlerFunc) {
	rid := ""
	if v, ok := d.Headers["x-request-id"]; ok {
		rid = fmt.Sprint(v)
	}

	// Continue any distributed trace propagated through the AMQP headers.
	msgCtx := otel.GetTextMapPropagator().Extract(
		applogger.WithRequestID(context.Background(), rid), amqpTableCarrier(d.Headers))
	jctx, j := journal.Start(msgCtx, journal.KindConsumer)
	j.SetRequestID(rid)
	spanCtx, endSpan := observability.StartSpan(jctx, "rabbitmq.consume "+d.RoutingKey)

	start := time.Now()
	herr := handler(spanCtx, Message{
		RoutingKey: d.RoutingKey,
		Body:       d.Body,
		RequestID:  rid,
	})
	endSpan()

	j.SetTraceID(observability.TraceID(spanCtx))
	c.logConsumed(d, rid, j, time.Since(start), herr)

	if herr != nil {
		c.log.Error().Err(herr).Str("routing_key", d.RoutingKey).Msg("rabbitmq handler failed, routing to DLQ")
		d.Nack(false, false) //nolint:errcheck
		return
	}
	d.Ack(false) //nolint:errcheck
}

// logConsumed writes one structured JSON line to consumer.log for a processed
// delivery, embedding the third-party calls and custom traces from the journal.
func (c *Consumer) logConsumed(d amqp.Delivery, rid string, j *journal.Journal, latency time.Duration, herr error) {
	status := "ok"
	if herr != nil {
		status = "error"
	}
	ev := applogger.Consumer().Info().
		Str("broker", "rabbitmq").
		Str("exchange", d.Exchange).
		Str("routing_key", d.RoutingKey).
		Str("request_id", rid).
		Str("trace_id", j.TraceID()).
		Dur("latency_ms", latency).
		Str("status", status).
		Interface("thirdparty", j.ThirdParties()).
		Interface("trace", j.Traces())
	if herr != nil {
		ev = ev.Str("error", herr.Error())
	}
	ev.Msg("consumed")
}

// Close shuts down the channel and connection.
func (c *Consumer) Close() error {
	if err := c.ch.Close(); err != nil {
		return err
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// amqpConnCloser is the subset of amqp.Connection used by the health check (enables mocking).
type amqpConnCloser interface {
	Close() error
}

// HealthCheck returns a probe that dials the AMQP broker and disconnects immediately.
func HealthCheck(dsn string) func(ctx context.Context) string {
	return healthCheckWithDialer(dsn, func(d string) (amqpConnCloser, error) {
		return amqp.Dial(d)
	})
}

func healthCheckWithDialer(dsn string, dial func(string) (amqpConnCloser, error)) func(ctx context.Context) string {
	return func(_ context.Context) string {
		if dsn == "" {
			return "not_configured"
		}
		conn, err := dial(dsn)
		if err != nil {
			return fmt.Sprintf("down: %v", err)
		}
		conn.Close() //nolint:errcheck
		return "ok"
	}
}
