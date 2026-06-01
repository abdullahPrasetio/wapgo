package observability

import (
	"context"

	"github.com/gofiber/fiber/v2"
)

// TraceContext returns the trace-carrying context stored by the active provider's
// HTTPMiddleware. Works with both OTel (stored in Locals "otel_ctx") and Elastic APM
// (stored in c.UserContext()). Falls back to context.Background() on routes without
// the middleware.
func TraceContext(c *fiber.Ctx) context.Context {
	if ctx, ok := c.Locals("otel_ctx").(context.Context); ok {
		return ctx
	}
	if ctx := c.UserContext(); ctx != nil {
		return ctx
	}
	return context.Background()
}

// fiberHeaderCarrier adapts Fiber request headers to propagation.TextMapCarrier.
// Used by otelProvider to extract/inject W3C TraceContext headers.
type fiberHeaderCarrier struct{ c *fiber.Ctx }

func (h *fiberHeaderCarrier) Get(key string) string { return h.c.Get(key) }
func (h *fiberHeaderCarrier) Set(key, val string)   { h.c.Set(key, val) }
func (h *fiberHeaderCarrier) Keys() []string {
	var keys []string
	h.c.Request().Header.VisitAll(func(k, _ []byte) {
		keys = append(keys, string(k))
	})
	return keys
}
