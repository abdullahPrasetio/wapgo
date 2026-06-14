// Package email provides an optional SMTP mailer add-on for wapgo services.
//
// Usage:
//
//	mailer := email.NewSMTPMailer(email.Config{
//	    Host: "smtp.example.com", Port: 587,
//	    Username: "user", Password: "pass",
//	    From: "noreply@example.com",
//	}, logger)
//
//	err := mailer.Send(ctx, email.Message{
//	    To: []string{"user@example.com"}, Subject: "Hello", Body: "<b>Hi!</b>", IsHTML: true,
//	})
//
// Each Send records an OTel span ("notification.email.send") and adds a
// ThirdParty entry to the request journal (visible in thirdparty.log and
// embedded in api.log / consumer.log).
package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"

	"github.com/abdullahPrasetio/wapgo/pkg/journal"
)

// Mailer is the interface for sending emails.
// Depend on this interface, not on SMTPMailer, so implementations can be swapped or mocked in tests.
type Mailer interface {
	Send(ctx context.Context, msg Message) error
}

// Message is the email to be sent.
type Message struct {
	To      []string
	CC      []string
	Subject string
	Body    string
	IsHTML  bool
}

// Config holds SMTP connection parameters.
// Populate from config.SMTPConfig or directly in tests.
type Config struct {
	Host     string
	Port     int          // 587 = STARTTLS (recommended), 465 = implicit TLS, 25 = plain
	Username string
	Password string
	From     string
	Timeout  time.Duration // default 10s
}

type smtpMailer struct {
	cfg Config
	log zerolog.Logger
}

// NewSMTPMailer creates an SMTPMailer. It does not open a connection at
// construction time — each Send dials a fresh connection (stateless, safe for
// concurrent use without a pool).
func NewSMTPMailer(cfg Config, log zerolog.Logger) Mailer {
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}
	return &smtpMailer{cfg: cfg, log: log}
}

// Send delivers msg via SMTP, records an OTel span, and appends a ThirdParty
// entry to the journal stored in ctx.
func (m *smtpMailer) Send(ctx context.Context, msg Message) error {
	if len(msg.To) == 0 {
		return fmt.Errorf("email: at least one recipient required")
	}
	ctx, span := otel.Tracer("wapgo").Start(ctx, "notification.email.send")
	defer span.End()

	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)
	start := time.Now()
	err := m.dial(ctx, addr, msg)
	latency := time.Since(start).Milliseconds()

	status := 250 // SMTP success
	errStr := ""
	if err != nil {
		status = 0
		errStr = err.Error()
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	journal.FromContext(ctx).AddThirdParty(journal.ThirdParty{
		Name:      "smtp",
		Method:    "SEND",
		URL:       addr,
		Host:      m.cfg.Host,
		Status:    status,
		LatencyMS: latency,
		Error:     errStr,
		StartedAt: start,
	})

	m.log.Debug().
		Str("to", strings.Join(msg.To, ",")).
		Str("subject", msg.Subject).
		Int64("latency_ms", latency).
		Err(err).
		Msg("notification.email.send")

	return err
}

// dial opens a new TCP connection to addr and sends the message.
// Port 465 → implicit TLS; port 587/25 → STARTTLS if offered.
func (m *smtpMailer) dial(ctx context.Context, addr string, msg Message) error {
	dialer := net.Dialer{Timeout: m.cfg.Timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp dial %s: %w", addr, err)
	}

	if m.cfg.Port == 465 {
		conn = tls.Client(conn, &tls.Config{ServerName: m.cfg.Host}) //nolint:gosec
	}

	c, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		conn.Close() //nolint:errcheck
		return fmt.Errorf("smtp client: %w", err)
	}
	defer c.Close() //nolint:errcheck

	if m.cfg.Port != 465 {
		ok, _ := c.Extension("STARTTLS")
		if !ok && m.cfg.Port == 587 {
			return fmt.Errorf("smtp: server at port 587 did not advertise STARTTLS")
		}
		if ok {
			if err := c.StartTLS(&tls.Config{ServerName: m.cfg.Host}); err != nil { //nolint:gosec
				return fmt.Errorf("smtp starttls: %w", err)
			}
		}
	}

	if m.cfg.Username != "" {
		auth := smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := c.Mail(m.cfg.From); err != nil {
		return fmt.Errorf("smtp MAIL FROM: %w", err)
	}

	for _, r := range append(msg.To, msg.CC...) {
		if err := c.Rcpt(r); err != nil {
			return fmt.Errorf("smtp RCPT %s: %w", r, err)
		}
	}

	wc, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA: %w", err)
	}
	defer wc.Close() //nolint:errcheck

	_, err = fmt.Fprint(wc, m.buildRaw(msg))
	return err
}

func (m *smtpMailer) buildRaw(msg Message) string {
	ct := "text/plain"
	if msg.IsHTML {
		ct = "text/html"
	}
	return fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nCc: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: %s; charset=\"utf-8\"\r\n\r\n%s",
		sanitizeHeader(m.cfg.From),
		sanitizeHeader(strings.Join(msg.To, ", ")),
		sanitizeHeader(strings.Join(msg.CC, ", ")),
		sanitizeHeader(msg.Subject),
		ct,
		msg.Body,
	)
}

// sanitizeHeader strips CR and LF characters to prevent email header injection.
func sanitizeHeader(s string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(s)
}
