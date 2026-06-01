package httpclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/sony/gobreaker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTransport is a function-backed http.RoundTripper used in tests.
type mockTransport func(*http.Request) (*http.Response, error)

func (f mockTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func okResp() *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}
}

func statusResp(code int) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(""))}
}

// ─── retryTransport ───────────────────────────────────────────────────────────

func TestRetryTransport_SuccessOnFirstAttempt(t *testing.T) {
	calls := 0
	rt := &retryTransport{
		inner:      mockTransport(func(*http.Request) (*http.Response, error) { calls++; return okResp(), nil }),
		maxRetries: 3,
		baseDelay:  time.Millisecond,
	}
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 1, calls)
}

func TestRetryTransport_RetriesOn5xx(t *testing.T) {
	calls := 0
	rt := &retryTransport{
		inner: mockTransport(func(*http.Request) (*http.Response, error) {
			calls++
			if calls < 3 {
				return statusResp(http.StatusServiceUnavailable), nil
			}
			return okResp(), nil
		}),
		maxRetries: 3,
		baseDelay:  time.Millisecond,
	}
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 3, calls)
}

func TestRetryTransport_ExhaustsRetries(t *testing.T) {
	calls := 0
	rt := &retryTransport{
		inner: mockTransport(func(*http.Request) (*http.Response, error) {
			calls++
			return statusResp(http.StatusInternalServerError), nil
		}),
		maxRetries: 3,
		baseDelay:  time.Millisecond,
	}
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	_, err := rt.RoundTrip(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
	assert.Equal(t, 4, calls) // 1 initial + 3 retries
}

func TestRetryTransport_SkipsRetryOnNonRetryableError(t *testing.T) {
	calls := 0
	rt := &retryTransport{
		inner: mockTransport(func(*http.Request) (*http.Response, error) {
			calls++
			return nil, errors.New("TLS: certificate signed by unknown authority")
		}),
		maxRetries: 3,
		baseDelay:  time.Millisecond,
	}
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	_, err := rt.RoundTrip(req)
	require.Error(t, err)
	assert.Equal(t, 1, calls) // no retry for non-retryable error
}

func TestRetryTransport_Does4xxWithoutRetry(t *testing.T) {
	calls := 0
	rt := &retryTransport{
		inner: mockTransport(func(*http.Request) (*http.Response, error) {
			calls++
			return statusResp(http.StatusNotFound), nil
		}),
		maxRetries: 3,
		baseDelay:  time.Millisecond,
	}
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, 1, calls)
}

func TestRetryTransport_RespectsContextCancellation(t *testing.T) {
	rt := &retryTransport{
		inner: mockTransport(func(r *http.Request) (*http.Response, error) {
			return statusResp(http.StatusInternalServerError), nil
		}),
		maxRetries: 5,
		baseDelay:  200 * time.Millisecond,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Cancel mid-backoff after first 5xx triggers the delay.
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com", nil)
	_, err := rt.RoundTrip(req)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

// ─── cbTransport ──────────────────────────────────────────────────────────────

func TestCBTransport_PassesOnSuccess(t *testing.T) {
	inner := mockTransport(func(*http.Request) (*http.Response, error) { return okResp(), nil })
	cb := newCBTransport(inner, gobreaker.Settings{
		Name:        "test",
		ReadyToTrip: func(c gobreaker.Counts) bool { return c.ConsecutiveFailures >= 3 },
	})
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	resp, err := cb.RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestCBTransport_OpensAfterConsecutiveFailures(t *testing.T) {
	inner := mockTransport(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("connection refused")
	})
	cb := newCBTransport(inner, gobreaker.Settings{
		Name:    "test",
		Timeout: 30 * time.Second,
		ReadyToTrip: func(c gobreaker.Counts) bool {
			return c.ConsecutiveFailures >= 3
		},
	})
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	for i := 0; i < 3; i++ {
		_, err := cb.RoundTrip(req)
		require.Error(t, err)
	}
	// Circuit should now be open.
	_, err := cb.RoundTrip(req)
	require.Error(t, err)
	assert.ErrorIs(t, err, gobreaker.ErrOpenState)
}

func TestCBTransport_StateString(t *testing.T) {
	inner := mockTransport(func(*http.Request) (*http.Response, error) { return okResp(), nil })
	cb := newCBTransport(inner, gobreaker.Settings{Name: "test"})
	assert.NotEmpty(t, cb.State())
}

// ─── ssrfGuardTransport ───────────────────────────────────────────────────────

func TestSSRFGuard_BlocksLoopbackIPv4(t *testing.T) {
	g := &ssrfGuardTransport{inner: mockTransport(func(*http.Request) (*http.Response, error) { return okResp(), nil })}
	req, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1/api", nil)
	_, err := g.RoundTrip(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SSRF")
}

func TestSSRFGuard_BlocksLocalhost(t *testing.T) {
	g := &ssrfGuardTransport{inner: mockTransport(func(*http.Request) (*http.Response, error) { return okResp(), nil })}
	req, _ := http.NewRequest(http.MethodGet, "http://localhost/api", nil)
	_, err := g.RoundTrip(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SSRF")
}

func TestSSRFGuard_BlocksPrivateIPs(t *testing.T) {
	g := &ssrfGuardTransport{inner: mockTransport(func(*http.Request) (*http.Response, error) { return okResp(), nil })}
	for _, ip := range []string{"10.0.0.1", "172.16.0.1", "192.168.1.1", "169.254.169.254"} {
		req, _ := http.NewRequest(http.MethodGet, "http://"+ip+"/api", nil)
		_, err := g.RoundTrip(req)
		require.Error(t, err, "expected SSRF block for %s", ip)
		assert.Contains(t, err.Error(), "SSRF", "host %s should be blocked", ip)
	}
}

func TestSSRFGuard_AllowsPublicHostNoAllowlist(t *testing.T) {
	inner := mockTransport(func(*http.Request) (*http.Response, error) { return okResp(), nil })
	g := &ssrfGuardTransport{inner: inner}
	req, _ := http.NewRequest(http.MethodGet, "http://example.com/api", nil)
	resp, err := g.RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestSSRFGuard_AllowsListedHost(t *testing.T) {
	inner := mockTransport(func(*http.Request) (*http.Response, error) { return okResp(), nil })
	g := &ssrfGuardTransport{inner: inner, allowedHosts: []string{"api.example.com"}}
	req, _ := http.NewRequest(http.MethodGet, "http://api.example.com/users", nil)
	resp, err := g.RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestSSRFGuard_BlocksUnlistedHost(t *testing.T) {
	g := &ssrfGuardTransport{
		inner:        mockTransport(func(*http.Request) (*http.Response, error) { return okResp(), nil }),
		allowedHosts: []string{"api.example.com"},
	}
	req, _ := http.NewRequest(http.MethodGet, "http://evil.example.com/api", nil)
	_, err := g.RoundTrip(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not in allowlist")
}

func TestSSRFGuard_BlocksInternalWhenAllowlistSet(t *testing.T) {
	g := &ssrfGuardTransport{
		inner:        mockTransport(func(*http.Request) (*http.Response, error) { return okResp(), nil }),
		allowedHosts: []string{"api.example.com"},
	}
	req, _ := http.NewRequest(http.MethodGet, "http://192.168.1.1/api", nil)
	_, err := g.RoundTrip(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SSRF")
}

func TestValidateHost_EmptyHost(t *testing.T) {
	err := validateHost("", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty host")
}

func TestValidateHost_AllowlistCaseInsensitive(t *testing.T) {
	err := validateHost("API.EXAMPLE.COM", []string{"api.example.com"})
	assert.NoError(t, err)
}

// mockNetError is a net.Error with a configurable Timeout value.
type mockNetError struct{ timeout bool }

func (e *mockNetError) Error() string   { return "mock net error" }
func (e *mockNetError) Timeout() bool   { return e.timeout }
func (e *mockNetError) Temporary() bool { return false }

func TestIsRetryableErr_Timeout(t *testing.T) {
	assert.True(t, isRetryableErr(&mockNetError{timeout: true}))
}

func TestIsRetryableErr_NotTimeout(t *testing.T) {
	assert.False(t, isRetryableErr(&mockNetError{timeout: false}))
}

func TestIsRetryableErr_ContextCanceled(t *testing.T) {
	assert.False(t, isRetryableErr(context.Canceled))
}

func TestIsRetryableErr_ContextDeadline(t *testing.T) {
	assert.False(t, isRetryableErr(context.DeadlineExceeded))
}

func TestRetryTransport_NonReplayableBodyBreaks(t *testing.T) {
	calls := 0
	rt := &retryTransport{
		inner: mockTransport(func(*http.Request) (*http.Response, error) {
			calls++
			return statusResp(http.StatusInternalServerError), nil
		}),
		maxRetries: 3,
		baseDelay:  time.Millisecond,
	}
	// Body set but GetBody=nil — retry is skipped after first failure.
	req, _ := http.NewRequest(http.MethodPost, "http://example.com", strings.NewReader("payload"))
	req.GetBody = nil
	_, err := rt.RoundTrip(req)
	require.Error(t, err)
	assert.Equal(t, 1, calls) // only one attempt
}

func TestRetryTransport_ResetsBodyViaGetBody(t *testing.T) {
	calls := 0
	bodyCalls := 0
	rt := &retryTransport{
		inner: mockTransport(func(*http.Request) (*http.Response, error) {
			calls++
			if calls < 2 {
				return statusResp(http.StatusInternalServerError), nil
			}
			return okResp(), nil
		}),
		maxRetries: 3,
		baseDelay:  time.Millisecond,
	}
	req, _ := http.NewRequest(http.MethodPost, "http://example.com", strings.NewReader("payload"))
	req.GetBody = func() (io.ReadCloser, error) {
		bodyCalls++
		return io.NopCloser(strings.NewReader("payload")), nil
	}
	resp, err := rt.RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 1, bodyCalls)
}

func TestRetryTransport_GetBodyError(t *testing.T) {
	calls := 0
	rt := &retryTransport{
		inner: mockTransport(func(*http.Request) (*http.Response, error) {
			calls++
			return statusResp(http.StatusInternalServerError), nil
		}),
		maxRetries: 3,
		baseDelay:  time.Millisecond,
	}
	req, _ := http.NewRequest(http.MethodPost, "http://example.com", strings.NewReader("payload"))
	req.GetBody = func() (io.ReadCloser, error) {
		return nil, errors.New("body reset failed")
	}
	_, err := rt.RoundTrip(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retry reset body")
}

func TestSSRFCheckRedirect_TooManyRedirects(t *testing.T) {
	fn := ssrfCheckRedirect(nil)
	req, _ := http.NewRequest(http.MethodGet, "http://example.com/", nil)
	err := fn(req, make([]*http.Request, 10))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "10 redirects")
}

func TestSSRFCheckRedirect_BlocksInternal(t *testing.T) {
	fn := ssrfCheckRedirect(nil)
	req, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1/redirect", nil)
	err := fn(req, []*http.Request{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SSRF")
}

func TestSSRFCheckRedirect_AllowsPublic(t *testing.T) {
	fn := ssrfCheckRedirect(nil)
	req, _ := http.NewRequest(http.MethodGet, "http://example.com/redirect", nil)
	err := fn(req, []*http.Request{})
	assert.NoError(t, err)
}

func TestIsInternalHost(t *testing.T) {
	cases := []struct {
		host     string
		internal bool
	}{
		{"localhost", true},
		{"127.0.0.1", true},
		{"::1", true},
		{"10.1.2.3", true},
		{"172.20.0.5", true},
		{"192.168.0.1", true},
		{"169.254.169.254", true}, // AWS metadata
		{"fe80::1", true},
		{"example.com", false},
		{"8.8.8.8", false},
		{"203.0.113.1", false},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.internal, isInternalHost(tc.host), "host=%s", tc.host)
	}
}
