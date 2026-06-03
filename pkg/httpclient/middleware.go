package httpclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/sony/gobreaker"
)

// ─── Retry transport ──────────────────────────────────────────────────────────

// retryTransport retries failed requests up to maxRetries times using
// exponential backoff. Only 5xx responses and network timeouts trigger a retry.
// Requests with non-seekable bodies are attempted once only.
type retryTransport struct {
	inner      http.RoundTripper
	maxRetries int
	baseDelay  time.Duration
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var (
		lastErr    error
		lastStatus int
	)

	for attempt := 0; attempt <= t.maxRetries; attempt++ {
		if attempt > 0 {
			// Non-replayable body: skip further retries.
			if req.Body != nil && req.GetBody == nil {
				break
			}
			if req.GetBody != nil {
				body, err := req.GetBody()
				if err != nil {
					return nil, fmt.Errorf("httpclient: retry reset body: %w", err)
				}
				req.Body = body
			}

			delay := t.baseDelay * time.Duration(math.Pow(2, float64(attempt-1)))
			timer := time.NewTimer(delay)
			select {
			case <-req.Context().Done():
				timer.Stop()
				return nil, req.Context().Err()
			case <-timer.C:
			}
		}

		resp, err := t.inner.RoundTrip(req)
		if err != nil {
			if isRetryableErr(err) {
				lastErr = err
				continue
			}
			return nil, err
		}
		if resp.StatusCode < 500 {
			return resp, nil
		}
		// 5xx — drain and close body, then retry.
		io.Copy(io.Discard, resp.Body) //nolint:errcheck
		resp.Body.Close() //nolint:errcheck
		lastStatus = resp.StatusCode
	}

	if lastErr != nil {
		return nil, fmt.Errorf("httpclient: transport error after %d retries: %w", t.maxRetries, lastErr)
	}
	return nil, fmt.Errorf("httpclient: server returned %d after %d retries", lastStatus, t.maxRetries)
}

// isRetryableErr returns true only for network-level timeouts.
// Context cancellation and deadline exceeded from the caller are not retried.
func isRetryableErr(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

// ─── Circuit-breaker transport ────────────────────────────────────────────────

// cbTransport wraps a RoundTripper with a gobreaker circuit breaker.
// The breaker opens after CBConsecutiveFailures consecutive errors and
// enters half-open after CBTimeout.
type cbTransport struct {
	inner http.RoundTripper
	cb    *gobreaker.CircuitBreaker
}

func newCBTransport(inner http.RoundTripper, settings gobreaker.Settings) *cbTransport {
	return &cbTransport{inner: inner, cb: gobreaker.NewCircuitBreaker(settings)}
}

func (t *cbTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	result, err := t.cb.Execute(func() (any, error) {
		return t.inner.RoundTrip(req)
	})
	if err != nil {
		return nil, err
	}
	return result.(*http.Response), nil //nolint:forcetypeassert
}

// State returns the current circuit breaker state name for health reporting.
func (t *cbTransport) State() string {
	return t.cb.State().String()
}

// ─── SSRF-guard transport ─────────────────────────────────────────────────────

// ssrfGuardTransport validates the target host before forwarding every request
// (including each redirect hop, since the transport is invoked per hop).
// It always blocks loopback, private, and link-local addresses, and enforces
// an optional allowlist of explicitly permitted hosts.
type ssrfGuardTransport struct {
	inner        http.RoundTripper
	allowedHosts []string
}

func (t *ssrfGuardTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := validateHost(req.URL.Hostname(), t.allowedHosts); err != nil {
		return nil, err
	}
	return t.inner.RoundTrip(req)
}

// ssrfCheckRedirect is used as http.Client.CheckRedirect to validate redirect
// targets as a second line of defence and cap the redirect chain at 10 hops.
func ssrfCheckRedirect(allowedHosts []string) func(*http.Request, []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return fmt.Errorf("httpclient: stopped after 10 redirects")
		}
		return validateHost(req.URL.Hostname(), allowedHosts)
	}
}

// validateHost returns an error when host is internal/loopback/link-local or
// is absent from allowedHosts (when the allowlist is non-empty).
func validateHost(host string, allowedHosts []string) error {
	if host == "" {
		return fmt.Errorf("httpclient: SSRF protection: empty host")
	}
	if isInternalHost(host) {
		return fmt.Errorf("httpclient: SSRF protection: host %q is internal/loopback/link-local", host)
	}
	if len(allowedHosts) == 0 {
		return nil
	}
	for _, h := range allowedHosts {
		if strings.EqualFold(host, h) {
			return nil
		}
	}
	return fmt.Errorf("httpclient: SSRF protection: host %q not in allowlist", host)
}

// privateRanges covers loopback, RFC-1918 private, link-local, and unique-local networks.
var privateRanges = func() []*net.IPNet {
	cidrs := []string{
		"127.0.0.0/8",    // loopback IPv4
		"10.0.0.0/8",     // private class A
		"172.16.0.0/12",  // private class B
		"192.168.0.0/16", // private class C
		"169.254.0.0/16", // link-local IPv4 (includes AWS metadata 169.254.169.254)
		"100.64.0.0/10",  // shared address space (CGNAT)
		"::1/128",        // loopback IPv6
		"fc00::/7",       // unique-local IPv6
		"fe80::/10",      // link-local IPv6
	}
	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		_, n, _ := net.ParseCIDR(cidr)
		nets = append(nets, n)
	}
	return nets
}()

// isInternalHost returns true for loopback, private, and link-local addresses.
// Hostname-based checks cover well-known aliases; IP-based checks cover numeric
// addresses. DNS-based SSRF (hostname resolves to internal IP) is mitigated by
// requiring hosts to be in the allowlist when one is configured.
func isInternalHost(host string) bool {
	switch strings.ToLower(host) {
	case "localhost", "ip6-localhost", "ip6-loopback":
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	for _, r := range privateRanges {
		if r.Contains(ip) {
			return true
		}
	}
	return false
}
