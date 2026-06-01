package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// CORS configures Cross-Origin Resource Sharing with a strict allowlist.
// Pass a comma-separated list of allowed origins (no wildcards when credentials
// are involved). Defaults to localhost:3000 when allowedOrigins is empty.
func CORS(allowedOrigins string) fiber.Handler {
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:3000"
	}
	return cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization,X-Request-ID",
		ExposeHeaders:    "X-Request-ID",
		AllowCredentials: false, // must be false with wildcard; set true + explicit origin in production
		MaxAge:           300,   // seconds to cache preflight
	})
}
