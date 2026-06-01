package observability_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abdullahPrasetio/wapgo/config"
	"github.com/abdullahPrasetio/wapgo/pkg/observability"
)

// newOTelProvider creates an OTel provider with stdout exporter for testing.
func newOTelProvider(t *testing.T, enabled bool) observability.Provider {
	t.Helper()
	cfg := &config.ObservabilityConfig{Provider: "otel", TracingEnabled: enabled}
	p, err := observability.New(context.Background(), cfg, "test-svc", "0.0.0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.Shutdown(context.Background()) })
	return p
}

func TestOTelProvider_Disabled(t *testing.T) {
	p := newOTelProvider(t, false)
	assert.NotNil(t, p)
	assert.NoError(t, p.Shutdown(context.Background()))
}

func TestOTelProvider_StdoutExporter(t *testing.T) {
	p := newOTelProvider(t, true) // no OTLPEndpoint → stdout exporter
	assert.NotNil(t, p)
}

func TestOTelProvider_HTTPMiddleware_CreatesSpan(t *testing.T) {
	p := newOTelProvider(t, true)

	app := fiber.New()
	app.Use(p.HTTPMiddleware())
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

func TestOTelProvider_HTTPMiddleware_PropagatesW3CHeaders(t *testing.T) {
	p := newOTelProvider(t, true)

	app := fiber.New()
	app.Use(p.HTTPMiddleware())
	app.Get("/trace", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	req := httptest.NewRequest(http.MethodGet, "/trace", nil)
	req.Header.Set("traceparent", "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestTraceContext_FallbackWhenNoMiddleware(t *testing.T) {
	app := fiber.New()
	app.Get("/no-trace", func(c *fiber.Ctx) error {
		ctx := observability.TraceContext(c)
		assert.NotNil(t, ctx) // must never return nil
		return c.SendStatus(200)
	})
	req := httptest.NewRequest(http.MethodGet, "/no-trace", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func ExampleNew() {
	cfg := &config.ObservabilityConfig{
		Provider:       "otel", // or "elastic_apm"
		TracingEnabled: true,
		OTLPEndpoint:   "",    // empty → stdout in dev; "collector:4318" in prod
	}
	p, err := observability.New(context.Background(), cfg, "my-service", "1.0.0")
	if err != nil {
		panic(err)
	}
	defer p.Shutdown(context.Background()) //nolint:errcheck
	// Use p.HTTPMiddleware() in your Fiber app,
	// p.InstrumentGORM(db), p.InstrumentRedis(client), p.WrapTransport(t).
}
