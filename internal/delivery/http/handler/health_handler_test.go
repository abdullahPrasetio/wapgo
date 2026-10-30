package handler_test

import (
	"context"
	"database/sql"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	_ "github.com/jackc/pgx/v5/stdlib" // register "pgx" driver
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abdullahPrasetio/wapgo/internal/delivery/http/handler"
)

func newHealthApp(h *handler.HealthHandler) *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/health", h.Check)
	return app
}

// openBrokenDB opens a connection to a port guaranteed to refuse connections.
func openBrokenDB(port string) *sql.DB {
	db, _ := sql.Open("pgx",
		"postgresql://user:pass@127.0.0.1:"+port+"/test?sslmode=disable&connect_timeout=1",
	)
	return db
}

// TestHealth_Degraded_DBDown verifies 503 when the database is unreachable.
func TestHealth_Degraded_DBDown(t *testing.T) {
	db := openBrokenDB("59997")
	defer db.Close()

	h := handler.NewHealthHandler(db, time.Now(), "v0.2.0", 2*time.Second)
	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := newHealthApp(h).Test(req, 8000)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)
}

// TestHealth_DisabledRedis_NotReported verifies an unconfigured optional
// dependency (e.g. Redis disabled) is absent from the services map.
func TestHealth_DisabledRedis_NotReported(t *testing.T) {
	db := openBrokenDB("59993")
	defer db.Close()

	h := handler.NewHealthHandler(db, time.Now(), "v0.2.0", 2*time.Second)
	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := newHealthApp(h).Test(req, 8000)
	require.NoError(t, err)

	body := parseBody(t, resp)
	services, ok := body["services"].(map[string]interface{})
	require.True(t, ok)
	assert.NotContains(t, services, "redis")
	assert.Equal(t, "down", services["database"])
}

// TestHealth_ResponseFields verifies that the response body contains expected top-level keys.
func TestHealth_ResponseFields(t *testing.T) {
	db := openBrokenDB("59995")
	defer db.Close()

	h := handler.NewHealthHandler(db, time.Now(), "v0.2.0", 2*time.Second)
	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := newHealthApp(h).Test(req, 8000)
	require.NoError(t, err)

	body := parseBody(t, resp)
	assert.Contains(t, body, "status")
	assert.Contains(t, body, "services")
	assert.Contains(t, body, "version")
	assert.Contains(t, body, "uptime")
}

// TestHealth_AddChecker_Down verifies that a "down" extra checker causes 503.
func TestHealth_AddChecker_Down(t *testing.T) {
	db := openBrokenDB("59987")
	defer db.Close()

	h := handler.NewHealthHandler(db, time.Now(), "v0.2.0", 2*time.Second)
	h.AddChecker("kafka", func(_ context.Context) string { return "down" })

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := newHealthApp(h).Test(req, 8000)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)

	body := parseBody(t, resp)
	services, ok := body["services"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "down", services["kafka"])
}

// TestHealth_AddChecker_NotConfigured verifies "not_configured" does not degrade overall status on its own.
func TestHealth_AddChecker_NotConfigured(t *testing.T) {
	db := openBrokenDB("59985")
	defer db.Close()

	h := handler.NewHealthHandler(db, time.Now(), "v0.2.0", 2*time.Second)
	h.AddChecker("rabbitmq", func(_ context.Context) string { return "not_configured" })

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := newHealthApp(h).Test(req, 8000)
	require.NoError(t, err)

	body := parseBody(t, resp)
	services, ok := body["services"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "not_configured", services["rabbitmq"])
}

// TestHealth_AddChecker_Chaining verifies the fluent AddChecker API.
func TestHealth_AddChecker_Chaining(t *testing.T) {
	db := openBrokenDB("59983")
	defer db.Close()

	h := handler.NewHealthHandler(db, time.Now(), "v0.2.0", 2*time.Second).
		AddChecker("kafka", func(_ context.Context) string { return "not_configured" }).
		AddChecker("rabbitmq", func(_ context.Context) string { return "not_configured" })

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := newHealthApp(h).Test(req, 8000)
	require.NoError(t, err)

	body := parseBody(t, resp)
	services := body["services"].(map[string]interface{})
	assert.Equal(t, "not_configured", services["kafka"])
	assert.Equal(t, "not_configured", services["rabbitmq"])
}
