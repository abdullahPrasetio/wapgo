package rabbitmq

import (
	"context"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
)

// Connection is a single shared, auto-reconnecting AMQP connection.
//
// Create ONE Connection per application process and pass it to all Consumer
// and Publisher instances. This guarantees a single TCP socket to the broker
// regardless of how many consumers or publishers you have, preventing the
// connection-explosion problem that crashes brokers under load.
//
// The underlying connection is re-dialed automatically when Channel() is called
// after a broker restart or network interruption.
type Connection struct {
	dsn  string
	mu   sync.RWMutex
	conn *amqp.Connection
	log  zerolog.Logger
}

// NewConnection dials the AMQP broker with a 5-second timeout and returns a
// shared Connection ready to be passed to NewConsumer and NewPublisher.
func NewConnection(dsn string, log zerolog.Logger) (*Connection, error) {
	c := &Connection{dsn: dsn, log: log}
	if err := c.dial(); err != nil {
		return nil, err
	}
	return c, nil
}

// Channel opens a new AMQP channel on the shared connection.
// If the connection was dropped, it is re-dialed transparently before opening
// the channel. Each Consumer and Publisher should hold its own channel.
func (c *Connection) Channel() (*amqp.Channel, error) {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil || conn.IsClosed() {
		if err := c.reconnect(); err != nil {
			return nil, err
		}
		c.mu.RLock()
		conn = c.conn
		c.mu.RUnlock()
	}

	ch, err := conn.Channel()
	if err != nil {
		// Connection was valid but channel open failed — reconnect once and retry.
		if rerr := c.reconnect(); rerr != nil {
			return nil, fmt.Errorf("rabbitmq channel: %w", err)
		}
		c.mu.RLock()
		conn = c.conn
		c.mu.RUnlock()
		ch, err = conn.Channel()
	}
	return ch, err
}

// IsClosed returns true if the underlying AMQP connection is nil or closed.
func (c *Connection) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn == nil || c.conn.IsClosed()
}

// Close gracefully shuts down the shared AMQP connection.
// Call this once during application shutdown after all consumers and publishers
// have been stopped.
func (c *Connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil && !c.conn.IsClosed() {
		return c.conn.Close()
	}
	return nil
}

// HealthCheck returns a probe that dials the broker and immediately disconnects.
// It uses a fresh connection so the shared Connection is never affected.
func HealthCheck(dsn string) func(ctx context.Context) string {
	return healthCheckWithDialer(dsn, func(d string) (amqpConnCloser, error) {
		return amqp.Dial(d)
	})
}

// reconnect re-dials the broker under a write lock. It is a no-op if the
// connection has already been restored by another goroutine.
func (c *Connection) reconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Double-check — another goroutine may have reconnected already.
	if c.conn != nil && !c.conn.IsClosed() {
		return nil
	}
	c.log.Warn().Msg("rabbitmq reconnecting")
	return c.dial()
}

func (c *Connection) dial() error {
	cfg := amqp.Config{
		Heartbeat: 10 * time.Second,
		Locale:    "en_US",
		Dial:      amqp.DefaultDial(5 * time.Second),
	}
	conn, err := amqp.DialConfig(c.dsn, cfg)
	if err != nil {
		return fmt.Errorf("rabbitmq dial: %w", err)
	}
	c.conn = conn
	return nil
}
