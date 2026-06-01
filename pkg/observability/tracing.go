package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"

	"github.com/gofiber/fiber/v2"

	applogger "github.com/abdullahPrasetio/wapgo/pkg/logger"
)

// TraceConfig configures the OpenTelemetry TracerProvider.
type TraceConfig struct {
	ServiceName    string // used as service.name resource attribute
	ServiceVersion string // used as service.version resource attribute
	// OTLPEndpoint is the OTLP HTTP collector endpoint (e.g. "localhost:4318").
	// When empty the stdout exporter is used — suitable for development.
	OTLPEndpoint string
	Enabled      bool
}

// SetupTracing initialises the global OTel TracerProvider and TextMapPropagator.
// The returned shutdown function must be deferred by the caller.
//
// When cfg.Enabled is false a no-op provider is installed so instrumentation
// code never panics, but no spans are exported.
func SetupTracing(ctx context.Context, cfg *TraceConfig) (shutdown func(context.Context) error, err error) {
	if !cfg.Enabled {
		otel.SetTracerProvider(tracenoop.NewTracerProvider())
		return func(context.Context) error { return nil }, nil
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("otel resource: %w", err)
	}

	var exporter sdktrace.SpanExporter
	if cfg.OTLPEndpoint != "" {
		exporter, err = otlptracehttp.New(ctx,
			otlptracehttp.WithEndpoint(cfg.OTLPEndpoint),
			otlptracehttp.WithInsecure(),
		)
	} else {
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
	}
	if err != nil {
		return nil, fmt.Errorf("otel exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}

// TracingMiddleware creates a server-side span for every HTTP request.
// It extracts incoming W3C TraceContext headers and annotates spans with
// the X-Request-ID set by the request-id middleware.
// The active context.Context (with span) is stored in Fiber Locals
// so that handlers can retrieve it via TraceContext(c).
func TracingMiddleware(serviceName string) fiber.Handler {
	tracer := otel.Tracer(serviceName)
	return func(c *fiber.Ctx) error {
		carrier := &fiberHeaderCarrier{c: c}
		ctx := otel.GetTextMapPropagator().Extract(context.Background(), carrier)

		route := c.Path()
		ctx, span := tracer.Start(ctx, c.Method()+" "+route,
			trace.WithSpanKind(trace.SpanKindServer),
		)
		defer span.End()

		// Propagate request-id as a span attribute.
		if rid := applogger.RequestIDFromContext(c.Context()); rid != "" {
			span.SetAttributes(attribute.String("http.request_id", rid))
		}

		// Make the span-carrying context available to handlers.
		c.Locals("otel_ctx", ctx)

		err := c.Next()

		span.SetAttributes(
			semconv.HTTPRequestMethodKey.String(c.Method()),
			semconv.HTTPRouteKey.String(c.Route().Path),
			semconv.HTTPResponseStatusCode(c.Response().StatusCode()),
		)
		if err != nil {
			span.RecordError(err)
		}
		return err
	}
}

// TraceContext returns the OTel-instrumented context stored by TracingMiddleware.
// Falls back to context.Background() on routes without the middleware.
func TraceContext(c *fiber.Ctx) context.Context {
	if ctx, ok := c.Locals("otel_ctx").(context.Context); ok {
		return ctx
	}
	return context.Background()
}

// fiberHeaderCarrier adapts Fiber request headers to propagation.TextMapCarrier.
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
