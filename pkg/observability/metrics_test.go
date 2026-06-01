package observability_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abdullahPrasetio/wapgo/pkg/observability"
)

func newMetricsApp() *fiber.App {
	app := fiber.New()
	app.Use(observability.MetricsMiddleware())
	app.Get("/ping", func(c *fiber.Ctx) error { return c.SendStatus(200) })
	app.Get("/metrics", observability.MetricsHandler())
	return app
}

func TestMetricsMiddleware_RecordsRequests(t *testing.T) {
	app := newMetricsApp()

	// fire a request to populate counters
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// verify /metrics endpoint returns 200 with Prometheus text format
	req2 := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	resp2, err := app.Test(req2)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	body, _ := io.ReadAll(resp2.Body)
	bodyStr := string(body)
	assert.Contains(t, bodyStr, "wapgo_http_requests_total")
	assert.Contains(t, bodyStr, "wapgo_http_request_duration_seconds")
}

func TestMetricsHandler_ContentType(t *testing.T) {
	app := newMetricsApp()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	ct := resp.Header.Get("Content-Type")
	assert.Contains(t, ct, "text/plain")
}

func ExampleMetricsHandler() {
	app := fiber.New()
	app.Use(observability.MetricsMiddleware())
	app.Get("/metrics", observability.MetricsHandler())
	// In production, guard /metrics so it is not publicly accessible:
	// app.Get("/metrics", prodGuard, observability.MetricsHandler())
	_ = app
}
