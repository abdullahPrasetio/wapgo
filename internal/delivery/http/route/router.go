package route

import (
	"github.com/gofiber/fiber/v2"

	"github.com/abdullahPrasetio/wapgo/internal/delivery/http/handler"
	"github.com/abdullahPrasetio/wapgo/pkg/observability"
)

// Setup registers all application routes on the Fiber app.
// appEnv is the value of APP_ENV ("development", "production", …).
func Setup(app *fiber.App, userHandler *handler.UserHandler, healthHandler *handler.HealthHandler, appEnv string) {
	// Health check — no auth, no additional rate limit per-route
	app.Get("/health", healthHandler.Check)

	// /metrics — exposed only outside production to prevent public scraping.
	// In production this endpoint returns 404 so it cannot be discovered.
	app.Get("/metrics", prodGuard(appEnv), observability.MetricsHandler())

	// API v1 group
	v1 := app.Group("/api/v1")
	RegisterUserRoutes(v1, userHandler)
}

// prodGuard returns a middleware that responds 404 when the app is running in
// production. Use it in front of internal-only endpoints (/metrics, /debug/pprof).
func prodGuard(appEnv string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if appEnv == "production" {
			return fiber.ErrNotFound
		}
		return c.Next()
	}
}
