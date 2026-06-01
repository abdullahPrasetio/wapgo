package handler

import (
	"context"
	"database/sql"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

// HealthHandler checks the liveness of downstream dependencies.
type HealthHandler struct {
	db          *sql.DB
	redisClient *redis.Client
	startTime   time.Time
	version     string
}

// NewHealthHandler creates a HealthHandler.
func NewHealthHandler(db *sql.DB, rc *redis.Client, startTime time.Time, version string) *HealthHandler {
	return &HealthHandler{
		db:          db,
		redisClient: rc,
		startTime:   startTime,
		version:     version,
	}
}

// Check handles GET /health.
// Returns HTTP 200 when all services are up, HTTP 503 when any is down.
func (h *HealthHandler) Check(c *fiber.Ctx) error {
	type serviceStatus struct {
		Database string `json:"database"`
		Redis    string `json:"redis"`
		Kafka    string `json:"kafka"`
		RabbitMQ string `json:"rabbitmq"`
	}

	status := serviceStatus{
		Kafka:    "not_configured",
		RabbitMQ: "not_configured",
	}
	overall := "ok"
	httpCode := fiber.StatusOK

	// DB ping (2-second budget)
	dbCtx, dbCancel := context.WithTimeout(c.UserContext(), 2*time.Second)
	defer dbCancel()
	if err := h.db.PingContext(dbCtx); err != nil {
		status.Database = "down"
		overall = "degraded"
		httpCode = fiber.StatusServiceUnavailable
	} else {
		status.Database = "ok"
	}

	// Redis ping (2-second budget)
	redisCtx, redisCancel := context.WithTimeout(c.UserContext(), 2*time.Second)
	defer redisCancel()
	if err := h.redisClient.Ping(redisCtx).Err(); err != nil {
		status.Redis = "down"
		overall = "degraded"
		httpCode = fiber.StatusServiceUnavailable
	} else {
		status.Redis = "ok"
	}

	return c.Status(httpCode).JSON(fiber.Map{
		"status":   overall,
		"services": status,
		"version":  h.version,
		"uptime":   time.Since(h.startTime).Round(time.Second).String(),
	})
}
