package handler_test

import (
	"database/sql"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	_ "github.com/jackc/pgx/v5/stdlib" // register "pgx" driver
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abdullahPrasetio/wapgo/internal/delivery/http/handler"
)

func newHealthApp(h *handler.HealthHandler) *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/health", h.Check)
	return app
}

// TestHealth_Degraded_BothDown verifies that the endpoint returns 503
// when neither DB nor Redis is reachable (connection-refused ports).
func TestHealth_Degraded_BothDown(t *testing.T) {
	// sql.Open does not dial until Ping — use a connection-refused port so
	// PingContext fails immediately (ECONNREFUSED).
	db, err := sql.Open("pgx",
		"postgresql://user:pass@127.0.0.1:59997/test?sslmode=disable&connect_timeout=1",
	)
	require.NoError(t, err)
	defer db.Close()

	rc := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:59996",
		DialTimeout: 300 * time.Millisecond,
	})
	defer rc.Close()

	h := handler.NewHealthHandler(db, rc, time.Now(), "v0.1.0")
	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := newHealthApp(h).Test(req, 8000) // generous timeout for CI
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)
}

// TestHealth_Fields verifies that the response body contains expected keys.
func TestHealth_ResponseFields(t *testing.T) {
	db, _ := sql.Open("pgx",
		"postgresql://user:pass@127.0.0.1:59995/test?sslmode=disable&connect_timeout=1",
	)
	defer db.Close()

	rc := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:59994",
		DialTimeout: 300 * time.Millisecond,
	})
	defer rc.Close()

	h := handler.NewHealthHandler(db, rc, time.Now(), "v0.1.0")
	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := newHealthApp(h).Test(req, 8000)
	require.NoError(t, err)

	body := parseBody(t, resp)
	assert.Contains(t, body, "status")
	assert.Contains(t, body, "services")
	assert.Contains(t, body, "version")
	assert.Contains(t, body, "uptime")
}
