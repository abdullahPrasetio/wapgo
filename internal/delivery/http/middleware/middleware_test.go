package middleware_test

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abdullahPrasetio/wapgo/internal/delivery/http/middleware"
)

func newApp(mw ...fiber.Handler) *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	for _, m := range mw {
		app.Use(m)
	}
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
	app.Options("/", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) })
	return app
}

// ── SecurityHeaders ───────────────────────────────────────────────────────────

func TestSecurityHeaders_XFrameOptions(t *testing.T) {
	app := newApp(middleware.SecurityHeaders())
	resp, err := app.Test(httptest.NewRequest("GET", "/", nil), -1)
	require.NoError(t, err)
	assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"))
}

func TestSecurityHeaders_ContentTypeNosniff(t *testing.T) {
	app := newApp(middleware.SecurityHeaders())
	resp, err := app.Test(httptest.NewRequest("GET", "/", nil), -1)
	require.NoError(t, err)
	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
}

func TestSecurityHeaders_CSP(t *testing.T) {
	app := newApp(middleware.SecurityHeaders())
	resp, err := app.Test(httptest.NewRequest("GET", "/", nil), -1)
	require.NoError(t, err)
	assert.Contains(t, resp.Header.Get("Content-Security-Policy"), "default-src")
}

func TestSecurityHeaders_ReferrerPolicy(t *testing.T) {
	app := newApp(middleware.SecurityHeaders())
	resp, err := app.Test(httptest.NewRequest("GET", "/", nil), -1)
	require.NoError(t, err)
	assert.Equal(t, "strict-origin-when-cross-origin", resp.Header.Get("Referrer-Policy"))
}

// ── CORS ─────────────────────────────────────────────────────────────────────

func TestCORS_AllowsConfiguredOrigin(t *testing.T) {
	app := newApp(middleware.CORS("https://example.com"))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://example.com")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", resp.Header.Get("Access-Control-Allow-Origin"))
}

func TestCORS_BlocksUnknownOrigin(t *testing.T) {
	app := newApp(middleware.CORS("https://example.com"))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://evil.com")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.NotEqual(t, "https://evil.com", resp.Header.Get("Access-Control-Allow-Origin"))
}

func TestCORS_DefaultOrigin_WhenEmpty(t *testing.T) {
	app := newApp(middleware.CORS(""))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:3000", resp.Header.Get("Access-Control-Allow-Origin"))
}

func TestCORS_PreflightReturns204(t *testing.T) {
	app := newApp(middleware.CORS("https://example.com"))
	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestCORS_ExposesRequestIDHeader(t *testing.T) {
	app := newApp(middleware.CORS("https://example.com"))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://example.com")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Contains(t, resp.Header.Get("Access-Control-Expose-Headers"), "X-Request-ID")
}

// ── RequestID ────────────────────────────────────────────────────────────────

func TestRequestID_SetsHeaderWhenAbsent(t *testing.T) {
	app := newApp(middleware.RequestID())
	resp, err := app.Test(httptest.NewRequest("GET", "/", nil), -1)
	require.NoError(t, err)
	id := resp.Header.Get("X-Request-Id")
	assert.NotEmpty(t, id)
}

func TestRequestID_PreservesExistingHeader(t *testing.T) {
	app := newApp(middleware.RequestID())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-Id", "my-custom-id")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, "my-custom-id", resp.Header.Get("X-Request-Id"))
}

// ── RateLimiter ───────────────────────────────────────────────────────────────

func TestRateLimiter_Returns429AfterLimit(t *testing.T) {
	app := newApp(middleware.RateLimiter())

	// Send 101 requests — the 101st must be rate-limited
	var lastCode int
	for i := 0; i <= 100; i++ {
		resp, err := app.Test(httptest.NewRequest("GET", "/", nil), -1)
		require.NoError(t, err)
		lastCode = resp.StatusCode
	}
	assert.Equal(t, fiber.StatusTooManyRequests, lastCode)
}

// ── Recover ──────────────────────────────────────────────────────────────────

func TestRecover_CatchesPanic(t *testing.T) {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(middleware.Recover())
	app.Get("/", func(c *fiber.Ctx) error {
		panic("something went wrong")
	})

	resp, err := app.Test(httptest.NewRequest("GET", "/", nil), -1)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}
