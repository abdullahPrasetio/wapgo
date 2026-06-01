package httpclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	applogger "github.com/abdullahPrasetio/wapgo/pkg/logger"
)

// testClient returns a Client whose http.Client uses the given transport,
// bypassing the full SSRF+retry+CB chain for unit tests.
func testClient(transport http.RoundTripper) *Client {
	return &Client{
		http: &http.Client{Transport: transport},
		opts: applyDefaults(Options{}),
	}
}

func TestClient_Do_InjectsRequestID(t *testing.T) {
	var got string
	c := testClient(mockTransport(func(r *http.Request) (*http.Response, error) {
		got = r.Header.Get("X-Request-ID")
		return okResp(), nil
	}))
	ctx := applogger.WithRequestID(context.Background(), "req-abc")
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com", nil)
	_, err := c.Do(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, "req-abc", got)
}

func TestClient_Do_InjectsAuthorization(t *testing.T) {
	var got string
	c := testClient(mockTransport(func(r *http.Request) (*http.Response, error) {
		got = r.Header.Get("Authorization")
		return okResp(), nil
	}))
	ctx := WithAuthorization(context.Background(), "Bearer token-xyz")
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com", nil)
	_, err := c.Do(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, "Bearer token-xyz", got)
}

func TestClient_Do_SkipsEmptyHeaders(t *testing.T) {
	var rid, auth string
	c := testClient(mockTransport(func(r *http.Request) (*http.Response, error) {
		rid = r.Header.Get("X-Request-ID")
		auth = r.Header.Get("Authorization")
		return okResp(), nil
	}))
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com", nil)
	_, err := c.Do(context.Background(), req)
	require.NoError(t, err)
	assert.Empty(t, rid)
	assert.Empty(t, auth)
}

func TestClient_Do_LimitsResponseBody(t *testing.T) {
	c := testClient(mockTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("1234567890")),
		}, nil
	}))
	c.opts.MaxResponseBytes = 5

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com", nil)
	resp, err := c.Do(context.Background(), req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "12345", string(body))
}

func TestClient_Get_ReadsBody(t *testing.T) {
	c := testClient(mockTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
		}, nil
	}))
	resp, body, err := c.Get(context.Background(), "http://example.com/data")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.JSONEq(t, `{"ok":true}`, string(body))
}

func TestWithAuthorization_RoundTrip(t *testing.T) {
	ctx := WithAuthorization(context.Background(), "Bearer xyz")
	assert.Equal(t, "Bearer xyz", AuthorizationFromContext(ctx))
	assert.Empty(t, AuthorizationFromContext(context.Background()))
}

// errReader is an io.ReadCloser whose Read always returns an error.
type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("forced read error") }
func (errReader) Close() error             { return nil }

func TestNew_DefaultOptions(t *testing.T) {
	c := New(Options{})
	require.NotNil(t, c)
	assert.Equal(t, 5*time.Second, c.opts.Timeout)
	assert.Equal(t, 3, c.opts.MaxRetries)
	assert.Equal(t, uint32(5), c.opts.CBConsecutiveFailures)
}

func TestNew_CustomOptions(t *testing.T) {
	c := New(Options{
		Timeout:    10 * time.Second,
		MaxRetries: 5,
		AllowedHosts: []string{"api.example.com"},
	})
	require.NotNil(t, c)
	assert.Equal(t, 10*time.Second, c.opts.Timeout)
	assert.Equal(t, 5, c.opts.MaxRetries)
}

func TestClient_Get_InvalidURL(t *testing.T) {
	c := testClient(mockTransport(func(*http.Request) (*http.Response, error) { return okResp(), nil }))
	_, _, err := c.Get(context.Background(), "://invalid-url")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "build request")
}

func TestClient_Get_ErrorOnBodyRead(t *testing.T) {
	c := testClient(mockTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: errReader{}}, nil
	}))
	_, _, err := c.Get(context.Background(), "http://example.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read body")
}

func ExampleNew() {
	client := New(Options{
		AllowedHosts: []string{"api.payment-service.com"},
		Timeout:      10 * time.Second,
	})
	// Use client.Get or client.Do for resilient inter-service calls.
	_ = client
}

func ExampleWithAuthorization() {
	ctx := WithAuthorization(context.Background(), "Bearer eyJhbGc...")
	// Pass ctx to client.Do — Authorization header is injected automatically.
	_ = ctx
}
