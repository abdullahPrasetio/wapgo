package middleware

import (
	"os"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// SecurityHeaders adds HTTP security headers (HSTS, CSP, X-Frame-Options, etc.).
func SecurityHeaders() fiber.Handler {
	return helmet.New(helmet.Config{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "DENY",
		HSTSMaxAge:            31536000, // 1 year
		HSTSExcludeSubdomains: false, // include subdomains
		HSTSPreloadEnabled:    false,
		ContentSecurityPolicy: "default-src 'self'",
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		PermissionPolicy:      "geolocation=(), microphone=(), camera=()",
	})
}

// RateLimiter limits each IP to 100 requests per minute by default.
// Override with RATE_LIMIT_MAX env var (e.g. RATE_LIMIT_MAX=1000000 for load testing).
func RateLimiter() fiber.Handler {
	max := 100
	if v := os.Getenv("RATE_LIMIT_MAX"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			max = n
		}
	}
	return newLimiter(max, time.Minute)
}


// StrictRateLimiter limits each IP to max requests per window.
// Use on write/mutation endpoints (e.g. POST /users, POST /auth/login).
func StrictRateLimiter(max int, window time.Duration) fiber.Handler {
	return newLimiter(max, window)
}

func newLimiter(max int, window time.Duration) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        max,
		Expiration: window,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"status":  false,
				"code":    "ERR_RATE_LIMIT",
				"message": "too many requests, please slow down",
			})
		},
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
