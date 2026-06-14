package route

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abdullahPrasetio/wapgo/internal/delivery/http/handler"
	"github.com/abdullahPrasetio/wapgo/pkg/auth"
)

func newTestApp() *fiber.App {
	return fiber.New(fiber.Config{DisableStartupMessage: true})
}

// ── prodGuard ─────────────────────────────────────────────────────────────────

func TestProdGuard_Returns404InProduction(t *testing.T) {
	app := newTestApp()
	app.Get("/metrics", prodGuard("production"), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	resp, err := app.Test(httptest.NewRequest("GET", "/metrics", nil), -1)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestProdGuard_AllowsInDevelopment(t *testing.T) {
	app := newTestApp()
	app.Get("/metrics", prodGuard("development"), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	resp, err := app.Test(httptest.NewRequest("GET", "/metrics", nil), -1)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestProdGuard_AllowsInStaging(t *testing.T) {
	app := newTestApp()
	app.Get("/x", prodGuard("staging"), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	resp, err := app.Test(httptest.NewRequest("GET", "/x", nil), -1)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

// ── welcomeHandler ────────────────────────────────────────────────────────────

func TestWelcomeHandler_DevEnv_HasDocLinks(t *testing.T) {
	app := newTestApp()
	app.Get("/", welcomeHandler("development"))
	resp, err := app.Test(httptest.NewRequest("GET", "/", nil), -1)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	links := body["links"].(map[string]any)
	assert.Contains(t, links, "docs")
	assert.Contains(t, links, "metrics")
}

func TestWelcomeHandler_ProdEnv_NoDocLinks(t *testing.T) {
	app := newTestApp()
	app.Get("/", welcomeHandler("production"))
	resp, err := app.Test(httptest.NewRequest("GET", "/", nil), -1)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	links := body["links"].(map[string]any)
	assert.NotContains(t, links, "docs")
	assert.NotContains(t, links, "metrics")
}

func TestWelcomeHandler_ReturnsServiceName(t *testing.T) {
	app := newTestApp()
	app.Get("/", welcomeHandler("test"))
	resp, err := app.Test(httptest.NewRequest("GET", "/", nil), -1)
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "wapgo", body["service"])
}

// ── Setup / RegisterRoutes ────────────────────────────────────────────────────

// minJWTCfg returns a JWT config with a 32-byte secret so auth middleware initialises.
func minJWTCfg() *auth.Config {
	return &auth.Config{Secret: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Issuer: "test", Audience: "test"}
}

func TestSetup_RegistersHealthRoute(t *testing.T) {
	app := newTestApp()
	// nil handlers are fine — Setup only registers routes, never calls them.
	Setup(app, (*handler.UserHandler)(nil), (*handler.AuthHandler)(nil), (*handler.HealthHandler)(nil),
		minJWTCfg(), nil, "development")

	// /health must be registered (handler will be called, but HealthHandler.Check is nil-safe for routing test)
	routes := app.GetRoutes()
	var paths []string
	for _, r := range routes {
		paths = append(paths, r.Path)
	}
	assert.Contains(t, paths, "/health")
	assert.Contains(t, paths, "/")
}

func TestSetup_ProdGuard_HidesMetrics(t *testing.T) {
	app := newTestApp()
	Setup(app, (*handler.UserHandler)(nil), (*handler.AuthHandler)(nil), (*handler.HealthHandler)(nil),
		minJWTCfg(), nil, "production")

	resp, err := app.Test(httptest.NewRequest("GET", "/metrics", nil), -1)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestSetup_ProdGuard_ExposesMetricsInDev(t *testing.T) {
	app := newTestApp()
	Setup(app, (*handler.UserHandler)(nil), (*handler.AuthHandler)(nil), (*handler.HealthHandler)(nil),
		minJWTCfg(), nil, "development")

	resp, err := app.Test(httptest.NewRequest("GET", "/metrics", nil), -1)
	require.NoError(t, err)
	assert.NotEqual(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestRegisterAuthRoutes_RegistersLoginRoute(t *testing.T) {
	app := newTestApp()
	v1 := app.Group("/api/v1")
	RegisterAuthRoutes(v1, (*handler.AuthHandler)(nil), minJWTCfg(), nil)

	routes := app.GetRoutes()
	var paths []string
	for _, r := range routes {
		paths = append(paths, r.Path)
	}
	assert.Contains(t, paths, "/api/v1/auth/login")
	assert.Contains(t, paths, "/api/v1/auth/refresh")
}

func TestRegisterUserRoutes_RegistersUsersRoute(t *testing.T) {
	app := newTestApp()
	v1 := app.Group("/api/v1")
	RegisterUserRoutes(v1, (*handler.UserHandler)(nil), minJWTCfg(), nil)

	routes := app.GetRoutes()
	var paths []string
	for _, r := range routes {
		paths = append(paths, r.Path)
	}
	assert.Contains(t, paths, "/api/v1/users")
}
