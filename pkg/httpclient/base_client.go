package httpclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sony/gobreaker"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	applogger "github.com/abdullahPrasetio/wapgo/pkg/logger"
)

type ctxKey int

const authKey ctxKey = iota

// WithAuthorization stores a Bearer token (or any Authorization header value)
// in ctx. The JWT middleware (v0.5) will call this; Client.Do reads it.
func WithAuthorization(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, authKey, token)
}

// AuthorizationFromContext retrieves the Authorization value set by WithAuthorization.
func AuthorizationFromContext(ctx context.Context) string {
	v, _ := ctx.Value(authKey).(string)
	return v
}

// Options configures a Client. Zero values use secure defaults.
type Options struct {
	Timeout               time.Duration // HTTP request timeout (default 5s)
	AllowedHosts          []string      // SSRF host allowlist; empty = block only internal addresses
	MaxResponseBytes      int64         // cap on response body size (default 10 MB)
	MaxRetries            int           // retry attempts on 5xx/network timeout (default 3)
	CBConsecutiveFailures uint32        // consecutive failures before circuit opens (default 5)
	CBTimeout             time.Duration // open→half-open after this duration (default 30s)
	// TransportWrapper, when set, wraps the final assembled transport chain.
	// Use observability.Provider.WrapTransport to add distributed tracing spans
	// for outgoing requests (OTel otelhttp.NewTransport or Elastic APM apmhttp.WrapRoundTripper).
	TransportWrapper func(http.RoundTripper) http.RoundTripper
}

func applyDefaults(o Options) Options {
	if o.Timeout == 0 {
		o.Timeout = 5 * time.Second
	}
	if o.MaxResponseBytes == 0 {
		o.MaxResponseBytes = 10 << 20 // 10 MB
	}
	if o.MaxRetries == 0 {
		o.MaxRetries = 3
	}
	if o.CBConsecutiveFailures == 0 {
		o.CBConsecutiveFailures = 5
	}
	if o.CBTimeout == 0 {
		o.CBTimeout = 30 * time.Second
	}
	return o
}

// Client is a resilient inter-service HTTP client.
//
// Security posture (all ON by default, no opt-out):
//   - TLS certificate verification enabled (InsecureSkipVerify=false, TLS 1.2+).
//   - SSRF protection: loopback/private/link-local hosts are always blocked;
//     an allowlist further restricts the set of reachable hosts.
//   - Response body capped at MaxResponseBytes to prevent resource exhaustion.
//   - Retry with exponential back-off on 5xx and network timeouts.
//   - Circuit breaker opens after CBConsecutiveFailures consecutive failures.
type Client struct {
	http *http.Client
	opts Options
}

// New builds a Client with the full resilience transport chain wired up:
//
//	http.Transport (TLS verify ON) → SSRF guard → retry → circuit breaker
func New(opts Options) *Client {
	opts = applyDefaults(opts)

	cbSettings := gobreaker.Settings{
		Name:    "httpclient",
		Timeout: opts.CBTimeout,
		ReadyToTrip: func(c gobreaker.Counts) bool {
			return c.ConsecutiveFailures >= opts.CBConsecutiveFailures
		},
	}

	base := &http.Transport{
		// TLS certificate verification is ON (InsecureSkipVerify defaults to false).
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
		MaxIdleConns:    100,
		IdleConnTimeout: 90 * time.Second,
	}
	ssrf := &ssrfGuardTransport{inner: base, allowedHosts: opts.AllowedHosts}
	retry := &retryTransport{inner: ssrf, maxRetries: opts.MaxRetries, baseDelay: 500 * time.Millisecond}
	cb := newCBTransport(retry, cbSettings)

	// Outermost layer: observability transport (tracing spans for outgoing requests).
	var finalTransport http.RoundTripper = cb
	if opts.TransportWrapper != nil {
		finalTransport = opts.TransportWrapper(cb)
	}

	hc := &http.Client{
		Transport:     finalTransport,
		Timeout:       opts.Timeout,
		CheckRedirect: ssrfCheckRedirect(opts.AllowedHosts),
	}
	return &Client{http: hc, opts: opts}
}

// Do executes req, injecting X-Request-ID and Authorization from ctx.
// The response body is capped at MaxResponseBytes.
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	req = req.WithContext(ctx)

	if rid := applogger.RequestIDFromContext(ctx); rid != "" {
		req.Header.Set("X-Request-ID", rid)
	}
	if auth := AuthorizationFromContext(ctx); auth != "" {
		req.Header.Set("Authorization", auth)
	}
	// Propagate OTel trace context (W3C TraceContext + Baggage) into outgoing headers.
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	resp.Body = &limitReadCloser{
		Reader: io.LimitReader(resp.Body, c.opts.MaxResponseBytes),
		Closer: resp.Body,
	}
	return resp, nil
}

// Get is a convenience wrapper: performs a GET, reads the body, and returns
// the response, body bytes, and any error.
func (c *Client) Get(ctx context.Context, url string) (*http.Response, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("httpclient: build request: %w", err)
	}
	resp, err := c.Do(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, fmt.Errorf("httpclient: read body: %w", err)
	}
	return resp, body, nil
}

// limitReadCloser couples an io.LimitReader with the original body's Close.
type limitReadCloser struct {
	io.Reader
	io.Closer
}
