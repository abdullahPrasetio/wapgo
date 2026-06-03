package route

import (
	"github.com/gofiber/fiber/v2"
	fiberSwagger "github.com/gofiber/swagger"

	"github.com/abdullahPrasetio/wapgo/internal/delivery/http/handler"
	"github.com/abdullahPrasetio/wapgo/pkg/observability"
)

// Setup registers all application routes on the Fiber app.
// appEnv is the value of APP_ENV ("development", "production", …).
func Setup(app *fiber.App, userHandler *handler.UserHandler, healthHandler *handler.HealthHandler, appEnv string) {
	// Welcome — service info landing page
	app.Get("/", welcomeHandler(appEnv))

	// Health check — no auth, no additional rate limit per-route
	app.Get("/health", healthHandler.Check)

	// Dev-only endpoints: return 404 in production so they cannot be discovered.
	app.Get("/metrics", prodGuard(appEnv), observability.MetricsHandler())
	app.Get("/docs/*", prodGuard(appEnv), fiberSwagger.HandlerDefault)

	// API v1 group
	v1 := app.Group("/api/v1")
	RegisterUserRoutes(v1, userHandler)
}

// prodGuard returns a middleware that responds 404 when the app is running in
// production. Use it in front of internal-only endpoints (/metrics, /docs, /debug/pprof).
func prodGuard(appEnv string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if appEnv == "production" {
			return fiber.ErrNotFound
		}
		return c.Next()
	}
}

// welcomeHandler returns a lightweight landing response with service info.
func welcomeHandler(appEnv string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		links := fiber.Map{
			"health": "/health",
		}
		if appEnv != "production" {
			links["docs"] = "/docs/index.html"
			links["metrics"] = "/metrics"
		}
		return c.JSON(fiber.Map{
			"service": "wapgo",
			"version": "1.0.0",
			"env":     appEnv,
			"links":   links,
		})
	}
}
