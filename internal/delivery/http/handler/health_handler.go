package handler

import (
	"context"
	"database/sql"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

// Checker is a health probe function.
// It must return "ok", "down", or "not_configured".
type Checker func(ctx context.Context) string

// HealthHandler checks the liveness of downstream dependencies.
type HealthHandler struct {
	db           *sql.DB
	redisClient  *redis.Client
	extras       map[string]Checker // additional probes (kafka, rabbitmq, …)
	startTime    time.Time
	version      string
	probeTimeout time.Duration
}

// NewHealthHandler creates a HealthHandler with DB and Redis probes.
// probeTimeout sets the per-dependency ping budget (0 → defaults to 2s).
// Additional probes can be registered afterwards via AddChecker.
func NewHealthHandler(db *sql.DB, rc *redis.Client, startTime time.Time, version string, probeTimeout time.Duration) *HealthHandler {
	if probeTimeout <= 0 {
		probeTimeout = 2 * time.Second
	}
	return &HealthHandler{
		db:           db,
		redisClient:  rc,
		startTime:    startTime,
		version:      version,
		probeTimeout: probeTimeout,
	}
}

// AddChecker registers a named health probe. Returns h for chaining.
func (h *HealthHandler) AddChecker(name string, fn Checker) *HealthHandler {
	if h.extras == nil {
		h.extras = make(map[string]Checker)
	}
	h.extras[name] = fn
	return h
}

// Check godoc
// @Summary      Health check
// @Description  Returns status of all downstream dependencies (DB, Redis, Kafka, RabbitMQ).
// @Tags         system
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      503  {object}  map[string]interface{}
// @Router       /health [get]
func (h *HealthHandler) Check(c *fiber.Ctx) error {
	services := make(map[string]string)
	overall := "ok"
	httpCode := fiber.StatusOK

	// DB ping
	dbCtx, dbCancel := context.WithTimeout(c.UserContext(), h.probeTimeout)
	defer dbCancel()
	if err := h.db.PingContext(dbCtx); err != nil {
		services["database"] = "down"
		overall = "degraded"
		httpCode = fiber.StatusServiceUnavailable
	} else {
		services["database"] = "ok"
	}

	// Redis ping
	redisCtx, redisCancel := context.WithTimeout(c.UserContext(), h.probeTimeout)
	defer redisCancel()
	if err := h.redisClient.Ping(redisCtx).Err(); err != nil {
		services["redis"] = "down"
		overall = "degraded"
		httpCode = fiber.StatusServiceUnavailable
	} else {
		services["redis"] = "ok"
	}

	// Extra probes (kafka, rabbitmq, …)
	for name, check := range h.extras {
		result := check(c.UserContext())
		services[name] = result
		if result == "down" {
			overall = "degraded"
			httpCode = fiber.StatusServiceUnavailable
		}
	}

	return c.Status(httpCode).JSON(fiber.Map{
		"status":   overall,
		"services": services,
		"version":  h.version,
		"uptime":   time.Since(h.startTime).Round(time.Second).String(),
	})
}
