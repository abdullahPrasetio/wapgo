package route

import (
	"github.com/gofiber/fiber/v2"

	"github.com/abdullahPrasetio/wapgo/internal/delivery/http/handler"
)

// Setup registers all application routes on the Fiber app.
func Setup(app *fiber.App, userHandler *handler.UserHandler, healthHandler *handler.HealthHandler) {
	// Health check — no auth, no rate limit applied per-route (global middleware is sufficient)
	app.Get("/health", healthHandler.Check)

	// API v1 group (add more domain routes here as the app grows)
	v1 := app.Group("/api/v1")
	RegisterUserRoutes(v1, userHandler)
}
