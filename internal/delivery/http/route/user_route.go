package route

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/abdullahPrasetio/wapgo/internal/delivery/http/handler"
	mw "github.com/abdullahPrasetio/wapgo/internal/delivery/http/middleware"
	"github.com/abdullahPrasetio/wapgo/pkg/auth"
)

// RegisterUserRoutes binds user CRUD endpoints to the given router group.
// All routes require a valid Bearer JWT; DELETE additionally requires the "admin" role.
func RegisterUserRoutes(router fiber.Router, h *handler.UserHandler, jwtCfg *auth.Config, bl auth.Blacklist) {
	authMW := auth.Middleware(jwtCfg, bl)
	writeLimiter := mw.StrictRateLimiter(20, time.Minute)

	users := router.Group("/users", authMW)
	users.Get("/", h.ListUsers)
	users.Get("/:id", h.GetUser)
	users.Post("/", writeLimiter, h.CreateUser)
	users.Put("/:id", writeLimiter, h.UpdateUser)
	users.Delete("/:id", writeLimiter, auth.RequireRole("admin"), h.DeleteUser)
}
