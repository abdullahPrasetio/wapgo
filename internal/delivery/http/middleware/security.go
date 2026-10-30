package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/helmet"
)

// SecurityHeaders adds HTTP security headers (HSTS, CSP, X-Frame-Options, etc.).
func SecurityHeaders() fiber.Handler {
	return helmet.New(helmet.Config{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "DENY",
		HSTSMaxAge:            31536000, // 1 year
		HSTSExcludeSubdomains: false,    // include subdomains
		HSTSPreloadEnabled:    false,
		ContentSecurityPolicy: "default-src 'self'",
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		PermissionPolicy:      "geolocation=(), microphone=(), camera=()",
	})
}

// WithBodyLimit returns a middleware that enforces a per-route body size limit.
// Use this on endpoints that handle small payloads (e.g. login, password reset)
// to override the global 4 MB Fiber limit:
//
//	router.Post("/auth/login", mw.WithBodyLimit(64*1024), handler.Login)
//
// Note: Fiber buffers the full request body before this middleware runs (up to the
// global BodyLimit, default 4 MB). This middleware enforces a tighter semantic limit
// post-buffer — it does not prevent large bodies from being read into memory.
func WithBodyLimit(maxBytes int) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if len(c.Body()) > maxBytes {
			return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
				"status":  false,
				"code":    "ERR_PAYLOAD_TOO_LARGE",
				"message": "request body too large",
			})
		}
		return c.Next()
	}
}
