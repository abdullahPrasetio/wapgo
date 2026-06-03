package handler_test

import (
	"context"
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

// openBrokenDB/RC open connections to ports guaranteed to refuse connections.
func openBrokenDB(port string) *sql.DB {
	db, _ := sql.Open("pgx",
		"postgresql://user:pass@127.0.0.1:"+port+"/test?sslmode=disable&connect_timeout=1",
	)
	return db
}

func openBrokenRC(port string) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:" + port,
		DialTimeout: 300 * time.Millisecond,
	})
}

// TestHealth_Degraded_BothDown verifies 503 when neither DB nor Redis is reachable.
func TestHealth_Degraded_BothDown(t *testing.T) {
	db := openBrokenDB("59997")
	defer db.Close()
	rc := openBrokenRC("59996")
	defer rc.Close()

	h := handler.NewHealthHandler(db, rc, time.Now(), "v0.2.0", 2*time.Second)
	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := newHealthApp(h).Test(req, 8000)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)
}

// TestHealth_ResponseFields verifies that the response body contains expected top-level keys.
func TestHealth_ResponseFields(t *testing.T) {
	db := openBrokenDB("59995")
	defer db.Close()
	rc := openBrokenRC("59994")
	defer rc.Close()

	h := handler.NewHealthHandler(db, rc, time.Now(), "v0.2.0", 2*time.Second)
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
	rc := openBrokenRC("59986")
	defer rc.Close()

	h := handler.NewHealthHandler(db, rc, time.Now(), "v0.2.0", 2*time.Second)
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
	rc := openBrokenRC("59984")
	defer rc.Close()

	h := handler.NewHealthHandler(db, rc, time.Now(), "v0.2.0", 2*time.Second)
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
	rc := openBrokenRC("59982")
	defer rc.Close()

	h := handler.NewHealthHandler(db, rc, time.Now(), "v0.2.0", 2*time.Second).
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
