//go:build ignore

package route

import (
	"github.com/gofiber/fiber/v2"

	"github.com/abdullahPrasetio/wapgo/internal/delivery/http/handler"
)

func Setup(app *fiber.App, userHandler *handler.UserHandler, healthHandler *handler.HealthHandler) {
	app.Get("/health", healthHandler.Check)

	v1 := app.Group("/api/v1")
	RegisterUserRoutes(v1, userHandler)
}
