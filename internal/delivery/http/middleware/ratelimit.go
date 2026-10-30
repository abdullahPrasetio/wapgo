package middleware

import (
	"os"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"

	"github.com/abdullahPrasetio/wapgo/pkg/ratelimit"
)

// RateLimiter limits each IP to 100 requests per minute by default.
// Override with RATE_LIMIT_MAX (requests) and RATE_LIMIT_WINDOW (Go duration).
//
// It uses the in-process MemoryLimiter: limits are per instance and reset on
// restart. For a quota shared across replicas use RateLimiterWith and pass a
// ratelimit.NewRedis limiter.
func RateLimiter() fiber.Handler {
	return RateLimiterWith(defaultLimiter())
}

// RateLimiterWith applies the given limiter. A nil limiter falls back to the
// default in-memory limiter. Storage errors fail open (the request is allowed
// and a warning is logged) so a Redis outage cannot take the API down.
func RateLimiterWith(l ratelimit.Limiter) fiber.Handler {
	if l == nil {
		l = defaultLimiter()
	}
	return func(c *fiber.Ctx) error {
		res, err := l.Allow(c.UserContext(), c.IP())
		if err != nil {
			log.Warn().Err(err).Str("ip", c.IP()).Msg("rate limiter unavailable, failing open")
			return c.Next()
		}

		c.Set("X-RateLimit-Remaining", strconv.Itoa(maxInt(res.Remaining, 0)))
		if !res.Allowed {
			retryAfter := int(res.RetryAfter.Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}
			c.Set(fiber.HeaderRetryAfter, strconv.Itoa(retryAfter))
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"status":  false,
				"code":    "ERR_RATE_LIMIT",
				"message": "too many requests, please slow down",
			})
		}
		return c.Next()
	}
}

// StrictRateLimiter limits each IP to max requests per window in process memory.
// Use on write/mutation endpoints (e.g. POST /users, POST /auth/login).
func StrictRateLimiter(max int, window time.Duration) fiber.Handler {
	return RateLimiterWith(ratelimit.NewMemory(max, window))
}

func defaultLimiter() ratelimit.Limiter {
	max := 100
	if v := os.Getenv("RATE_LIMIT_MAX"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			max = n
		}
	}
	window := time.Minute
	if v := os.Getenv("RATE_LIMIT_WINDOW"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			window = d
		}
	}
	return ratelimit.NewMemory(max, window)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
