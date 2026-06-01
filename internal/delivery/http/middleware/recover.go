package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// Recover catches panics and returns HTTP 500 without leaking stack traces.
func Recover() fiber.Handler {
	return recover.New(recover.Config{
		EnableStackTrace: false, // never leak internals to the client
	})
}
