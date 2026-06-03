//go:build ignore

package route

import (
	"github.com/gofiber/fiber/v2"
	fiberSwagger "github.com/gofiber/swagger"

	"github.com/abdullahPrasetio/wapgo/internal/delivery/http/handler"
	"github.com/abdullahPrasetio/wapgo/pkg/observability"
)

func Setup(app *fiber.App, userHandler *handler.UserHandler, healthHandler *handler.HealthHandler, appEnv string) {
	app.Get("/", welcomeHandler(appEnv))
	app.Get("/health", healthHandler.Check)
	app.Get("/metrics", prodGuard(appEnv), observability.MetricsHandler())
	app.Get("/docs/*", prodGuard(appEnv), fiberSwagger.HandlerDefault)

	v1 := app.Group("/api/v1")
	RegisterUserRoutes(v1, userHandler)
}

// prodGuard returns 404 for internal endpoints when APP_ENV=production.
func prodGuard(appEnv string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if appEnv == "production" {
			return fiber.ErrNotFound
		}
		return c.Next()
	}
}

func welcomeHandler(appEnv string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		links := fiber.Map{"health": "/health"}
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
