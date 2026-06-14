package email

import (
	"context"
	"net"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func noopLogger() zerolog.Logger { return zerolog.Nop() }

// ── sanitizeHeader ─────────────────────────────────────────────────────────────

func TestSanitizeHeader_StripsCR(t *testing.T) {
	assert.Equal(t, "helloworld", sanitizeHeader("hello\rworld"))
}

func TestSanitizeHeader_StripsLF(t *testing.T) {
	assert.Equal(t, "helloworld", sanitizeHeader("hello\nworld"))
}

func TestSanitizeHeader_StripsBoth(t *testing.T) {
	assert.Equal(t, "injected", sanitizeHeader("injected\r\n"))
}

func TestSanitizeHeader_PassesCleanString(t *testing.T) {
	assert.Equal(t, "noreply@example.com", sanitizeHeader("noreply@example.com"))
}

// ── NewSMTPMailer ──────────────────────────────────────────────────────────────

func TestNewSMTPMailer_DefaultTimeout(t *testing.T) {
	m := NewSMTPMailer(Config{Host: "smtp.example.com", Port: 587}, noopLogger())
	require.NotNil(t, m)
	impl := m.(*smtpMailer)
	assert.Equal(t, int64(10_000_000_000), int64(impl.cfg.Timeout)) // 10s default
}

func TestNewSMTPMailer_CustomTimeout(t *testing.T) {
	m := NewSMTPMailer(Config{Host: "smtp.example.com", Port: 587, Timeout: 5_000_000_000}, noopLogger())
	impl := m.(*smtpMailer)
	assert.Equal(t, int64(5_000_000_000), int64(impl.cfg.Timeout))
}

// ── buildRaw ───────────────────────────────────────────────────────────────────

func TestBuildRaw_PlainText(t *testing.T) {
	m := &smtpMailer{cfg: Config{From: "sender@example.com"}}
	msg := Message{To: []string{"a@b.com"}, Subject: "Hi", Body: "Hello"}
	raw := m.buildRaw(msg)
	assert.Contains(t, raw, "Content-Type: text/plain")
	assert.Contains(t, raw, "Subject: Hi")
	assert.Contains(t, raw, "Hello")
}

func TestBuildRaw_HTML(t *testing.T) {
	m := &smtpMailer{cfg: Config{From: "sender@example.com"}}
	msg := Message{To: []string{"a@b.com"}, Subject: "Hi", Body: "<b>Hi</b>", IsHTML: true}
	raw := m.buildRaw(msg)
	assert.Contains(t, raw, "Content-Type: text/html")
}

func TestBuildRaw_CCIncluded(t *testing.T) {
	m := &smtpMailer{cfg: Config{From: "sender@example.com"}}
	msg := Message{To: []string{"a@b.com"}, CC: []string{"c@d.com"}, Subject: "Sub", Body: "body"}
	raw := m.buildRaw(msg)
	assert.Contains(t, raw, "c@d.com")
}

func TestBuildRaw_SubjectSanitized(t *testing.T) {
	m := &smtpMailer{cfg: Config{From: "sender@example.com"}}
	msg := Message{To: []string{"a@b.com"}, Subject: "Injected\r\nBcc: evil@x.com", Body: "x"}
	raw := m.buildRaw(msg)
	assert.NotContains(t, raw, "\r\nBcc")
}

// ── Send ───────────────────────────────────────────────────────────────────────

func TestSend_ErrorOnEmptyRecipients(t *testing.T) {
	m := NewSMTPMailer(Config{Host: "smtp.example.com", Port: 587}, noopLogger())
	err := m.Send(context.Background(), Message{To: nil, Subject: "Hi", Body: "x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "recipient")
}

func TestSend_ErrorOnDialFailure(t *testing.T) {
	m := NewSMTPMailer(Config{Host: "127.0.0.1", Port: 19999, Timeout: 1e8}, noopLogger())
	err := m.Send(context.Background(), Message{To: []string{"a@b.com"}, Subject: "Hi", Body: "x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "smtp dial")
}

// ── HealthCheck ───────────────────────────────────────────────────────────────

func TestHealthCheck_ReturnsErrorOnClosedPort(t *testing.T) {
	probe := HealthCheck("127.0.0.1", 19998)
	result := probe(context.Background())
	assert.Contains(t, result, "error:")
}

func TestHealthCheck_ReturnsOKOnOpenPort(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skip("cannot open test listener")
	}
	defer ln.Close()
	addr := ln.Addr().(*net.TCPAddr)

	probe := HealthCheck("127.0.0.1", addr.Port)
	result := probe(context.Background())
	assert.Equal(t, "ok", result)
}
