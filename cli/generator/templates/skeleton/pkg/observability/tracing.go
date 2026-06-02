//go:build ignore

package observability

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
)

// StartSpan begins a span named name as a child of whatever trace is active in ctx,
// and returns the derived context plus an end func to defer.
//
// It uses the global OTel tracer, so it works for every backend wired by New():
//   - "otel"        — spans go to the OTel SDK exporter
//   - "elastic_apm" — the apmotel bridge forwards spans to Elastic APM
//   - "none"        — the global tracer is a no-op, so this is essentially free
//
// Usage:
//
//	ctx, end := observability.StartSpan(ctx, "ScoreRisk")
//	defer end()
func StartSpan(ctx context.Context, name string) (context.Context, func()) {
	ctx, span := otel.Tracer("wapgo").Start(ctx, name)
	return ctx, func() { span.End() }
}

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
