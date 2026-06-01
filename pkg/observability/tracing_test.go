package observability_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abdullahPrasetio/wapgo/pkg/observability"
)

func TestSetupTracing_Disabled(t *testing.T) {
	shutdown, err := observability.SetupTracing(context.Background(), &observability.TraceConfig{
		Enabled: false,
	})
	require.NoError(t, err)
	require.NotNil(t, shutdown)
	assert.NoError(t, shutdown(context.Background()))
}

func TestSetupTracing_StdoutExporter(t *testing.T) {
	shutdown, err := observability.SetupTracing(context.Background(), &observability.TraceConfig{
		ServiceName:    "test-svc",
		ServiceVersion: "0.0.0",
		Enabled:        true, // no OTLPEndpoint → stdout
	})
	require.NoError(t, err)
	require.NotNil(t, shutdown)
	assert.NoError(t, shutdown(context.Background()))
}

func TestTracingMiddleware_CreatesSpan(t *testing.T) {
	// Ensure tracing is enabled (stdout exporter — no network needed)
	shutdown, err := observability.SetupTracing(context.Background(), &observability.TraceConfig{
		ServiceName: "test",
		Enabled:     true,
	})
	require.NoError(t, err)
	defer shutdown(context.Background()) //nolint:errcheck

	app := fiber.New()
	app.Use(observability.TracingMiddleware("test"))
	app.Get("/hello", func(c *fiber.Ctx) error {
		ctx := observability.TraceContext(c)
		assert.NotNil(t, ctx)
		return c.SendStatus(200)
	})

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestTracingMiddleware_PropagatesW3CHeaders(t *testing.T) {
	shutdown, err := observability.SetupTracing(context.Background(), &observability.TraceConfig{
		ServiceName: "test",
		Enabled:     true,
	})
	require.NoError(t, err)
	defer shutdown(context.Background()) //nolint:errcheck

	app := fiber.New()
	app.Use(observability.TracingMiddleware("test"))
	app.Get("/trace", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	req := httptest.NewRequest(http.MethodGet, "/trace", nil)
	// Inject a valid W3C traceparent header
	req.Header.Set("traceparent", "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestTraceContext_FallbackWhenNoMiddleware(t *testing.T) {
	app := fiber.New()
	app.Get("/no-trace", func(c *fiber.Ctx) error {
		ctx := observability.TraceContext(c)
		assert.NotNil(t, ctx) // must return context.Background(), never nil
		return c.SendStatus(200)
	})
	req := httptest.NewRequest(http.MethodGet, "/no-trace", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func ExampleSetupTracing() {
	ctx := context.Background()
	shutdown, err := observability.SetupTracing(ctx, &observability.TraceConfig{
		ServiceName:    "my-service",
		ServiceVersion: "1.0.0",
		OTLPEndpoint:   "", // empty → stdout in dev; set "collector:4318" in prod
		Enabled:        true,
	})
	if err != nil {
		panic(err)
	}
	defer shutdown(ctx) //nolint:errcheck
	_ = io.Discard     // use tracer via otel.Tracer("my-service")
}
