package email

import (
	"context"
	"fmt"
	"net"
	"time"
)

// HealthCheck returns a probe that dials the SMTP server TCP socket.
// It does not send a real email — it verifies the host:port is reachable.
//
// Wire it into your health endpoint the same way as rabbitmq.HealthCheck:
//
//	health.Register("smtp", email.HealthCheck("smtp.example.com", 587))
func HealthCheck(host string, port int) func(ctx context.Context) string {
	addr := fmt.Sprintf("%s:%d", host, port)
	return func(ctx context.Context) string {
		timeout := 5 * time.Second
		if dl, ok := ctx.Deadline(); ok {
			timeout = time.Until(dl)
		}
		conn, err := net.DialTimeout("tcp", addr, timeout)
		if err != nil {
			return "error: " + err.Error()
		}
		conn.Close() //nolint:errcheck
		return "ok"
	}
}
