package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abdullahPrasetio/wapgo/internal/domain/entity"
	applogger "github.com/abdullahPrasetio/wapgo/pkg/logger"
)

// newTestUserClient builds a UserClient backed by a test HTTP server.
// The client bypasses the SSRF guard so it can reach 127.0.0.1.
func newTestUserClient(t *testing.T, handler http.HandlerFunc) (*UserClient, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	uc := &UserClient{
		// Use the test server's own client (no SSRF guard) pointing at srv.URL.
		client:  &Client{http: srv.Client(), opts: applyDefaults(Options{})},
		baseURL: srv.URL,
	}
	return uc, srv
}

func TestUserClient_GetUser_Success(t *testing.T) {
	want := entity.User{Name: "Alice", Email: "alice@example.com"}
	uc, _ := newTestUserClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/users/uid-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": true, "data": want}) //nolint:errcheck
	})
	got, err := uc.GetUser(context.Background(), "uid-1")
	require.NoError(t, err)
	assert.Equal(t, want.Name, got.Name)
	assert.Equal(t, want.Email, got.Email)
}

func TestUserClient_GetUser_NotFound(t *testing.T) {
	uc, _ := newTestUserClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	_, err := uc.GetUser(context.Background(), "no-such-user")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestUserClient_GetUser_ServerError(t *testing.T) {
	uc, _ := newTestUserClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	_, err := uc.GetUser(context.Background(), "uid-1")
	require.Error(t, err)
	assert.False(t, errors.Is(err, ErrUserNotFound))
}

func TestUserClient_GetUser_InvalidJSON(t *testing.T) {
	uc, _ := newTestUserClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not-json")) //nolint:errcheck
	})
	_, err := uc.GetUser(context.Background(), "uid-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}

func TestUserClient_GetUser_PropagatesRequestID(t *testing.T) {
	var got string
	uc, _ := newTestUserClient(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Request-ID")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": true, "data": entity.User{}}) //nolint:errcheck
	})

	ctx := applogger.WithRequestID(context.Background(), "trace-99")
	_, err := uc.GetUser(ctx, "uid-1")
	require.NoError(t, err)
	assert.Equal(t, "trace-99", got)
}

func TestNewUserClient_ReturnsClient(t *testing.T) {
	uc := NewUserClient("http://user-service.example.com", Options{})
	require.NotNil(t, uc)
	assert.Equal(t, "http://user-service.example.com", uc.baseURL)
}

func ExampleUserClient_GetUser() {
	// NewUserClient is typically wired in cmd/api/main.go:
	//   uc := httpclient.NewUserClient(cfg.Services.UserURL, httpclient.Options{})
	//   user, err := uc.GetUser(ctx, userID)
	_ = struct{}{}
}
