package google

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testCfg = Config{
	ClientID:     "client-id",
	ClientSecret: "client-secret",
	RedirectURL:  "https://example.com/callback",
}

// ── New ───────────────────────────────────────────────────────────────────────

func TestNew_ReturnsProvider(t *testing.T) {
	p := New(testCfg)
	assert.NotNil(t, p)
	assert.NotNil(t, p.cfg)
}

func TestNew_SetsClientID(t *testing.T) {
	p := New(testCfg)
	assert.Equal(t, "client-id", p.cfg.ClientID)
}

func TestNew_SetsRedirectURL(t *testing.T) {
	p := New(testCfg)
	assert.Equal(t, "https://example.com/callback", p.cfg.RedirectURL)
}

func TestNew_SetsScopes(t *testing.T) {
	p := New(testCfg)
	assert.Contains(t, p.cfg.Scopes, "email")
	assert.Contains(t, p.cfg.Scopes, "profile")
}

// ── AuthURL ───────────────────────────────────────────────────────────────────

func TestAuthURL_ContainsState(t *testing.T) {
	p := New(testCfg)
	url := p.AuthURL("my-csrf-state")
	assert.Contains(t, url, "my-csrf-state")
}

func TestAuthURL_ContainsClientID(t *testing.T) {
	p := New(testCfg)
	url := p.AuthURL("state")
	assert.Contains(t, url, "client-id")
}

func TestAuthURL_IsHTTPS(t *testing.T) {
	p := New(testCfg)
	url := p.AuthURL("state")
	assert.True(t, len(url) > 0)
	assert.Contains(t, url, "https://")
}

// ── Exchange ──────────────────────────────────────────────────────────────────

func TestExchange_ErrorOnInvalidCode(t *testing.T) {
	// Point the token endpoint at a local server that always returns an error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid_grant"}`)) //nolint:errcheck
	}))
	defer srv.Close()

	p := New(testCfg)
	// Override the token URL to hit our local server.
	p.cfg.Endpoint.TokenURL = srv.URL

	_, err := p.Exchange(context.Background(), "bad-code")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "google oauth exchange")
}
