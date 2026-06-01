//go:build ignore

package httpclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sony/gobreaker"

	applogger "github.com/abdullahPrasetio/wapgo/pkg/logger"
)

type ctxKey int

const authKey ctxKey = iota

func WithAuthorization(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, authKey, token)
}

func AuthorizationFromContext(ctx context.Context) string {
	v, _ := ctx.Value(authKey).(string)
	return v
}

type Options struct {
	Timeout               time.Duration
	AllowedHosts          []string
	MaxResponseBytes      int64
	MaxRetries            int
	CBConsecutiveFailures uint32
	CBTimeout             time.Duration
}

func applyDefaults(o Options) Options {
	if o.Timeout == 0 {
		o.Timeout = 5 * time.Second
	}
	if o.MaxResponseBytes == 0 {
		o.MaxResponseBytes = 10 << 20
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

type Client struct {
	http *http.Client
	opts Options
}

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
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
		MaxIdleConns:    100,
		IdleConnTimeout: 90 * time.Second,
	}
	ssrf := &ssrfGuardTransport{inner: base, allowedHosts: opts.AllowedHosts}
	retry := &retryTransport{inner: ssrf, maxRetries: opts.MaxRetries, baseDelay: 500 * time.Millisecond}
	cb := newCBTransport(retry, cbSettings)

	hc := &http.Client{
		Transport:     cb,
		Timeout:       opts.Timeout,
		CheckRedirect: ssrfCheckRedirect(opts.AllowedHosts),
	}
	return &Client{http: hc, opts: opts}
}

func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	req = req.WithContext(ctx)

	if rid := applogger.RequestIDFromContext(ctx); rid != "" {
		req.Header.Set("X-Request-ID", rid)
	}
	if auth := AuthorizationFromContext(ctx); auth != "" {
		req.Header.Set("Authorization", auth)
	}

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

type limitReadCloser struct {
	io.Reader
	io.Closer
}
