package route

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/abdullahPrasetio/wapgo/internal/delivery/http/handler"
	mw "github.com/abdullahPrasetio/wapgo/internal/delivery/http/middleware"
)

// RegisterUserRoutes binds user CRUD endpoints to the given router group.
func RegisterUserRoutes(router fiber.Router, h *handler.UserHandler) {
	// Stricter rate limit for write operations: 20 req/min per IP.
	writeLimiter := mw.StrictRateLimiter(20, time.Minute)

	users := router.Group("/users")
	users.Get("/", h.ListUsers)
	users.Get("/:id", h.GetUser)
	users.Post("/", writeLimiter, h.CreateUser)
	users.Put("/:id", writeLimiter, h.UpdateUser)
	users.Delete("/:id", writeLimiter, h.DeleteUser)
}
