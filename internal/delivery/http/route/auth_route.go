package route

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/abdullahPrasetio/wapgo/internal/delivery/http/handler"
	mw "github.com/abdullahPrasetio/wapgo/internal/delivery/http/middleware"
	"github.com/abdullahPrasetio/wapgo/pkg/auth"
)

// RegisterAuthRoutes binds authentication endpoints to the given router group.
func RegisterAuthRoutes(router fiber.Router, h *handler.AuthHandler, jwtCfg *auth.Config, bl auth.Blacklist) {
	// Tighter rate limit on auth endpoints to slow credential stuffing.
	loginLimiter := mw.StrictRateLimiter(10, time.Minute)

	a := router.Group("/auth")
	a.Post("/login", loginLimiter, h.Login)
	a.Post("/refresh", loginLimiter, h.Refresh)
	a.Post("/logout", auth.Middleware(jwtCfg, bl), h.Logout)
	a.Post("/forgot-password", loginLimiter, h.ForgotPassword)
	a.Post("/reset-password", loginLimiter, h.ResetPassword)
}
