//go:build ignore

package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"github.com/abdullahPrasetio/wapgo/config"
	"github.com/abdullahPrasetio/wapgo/internal/delivery/http/handler"
	mw "github.com/abdullahPrasetio/wapgo/internal/delivery/http/middleware"
	"github.com/abdullahPrasetio/wapgo/internal/delivery/http/route"
	"github.com/abdullahPrasetio/wapgo/internal/domain/entity"
	pgRepo "github.com/abdullahPrasetio/wapgo/internal/repository/postgres"
	"github.com/abdullahPrasetio/wapgo/internal/usecase"
	"github.com/abdullahPrasetio/wapgo/pkg/database"
	applogger "github.com/abdullahPrasetio/wapgo/pkg/logger"
	kafkamsg "github.com/abdullahPrasetio/wapgo/pkg/messaging/kafka"
	rabbitmqmsg "github.com/abdullahPrasetio/wapgo/pkg/messaging/rabbitmq"
	"github.com/abdullahPrasetio/wapgo/pkg/validator"
)

const version = "0.1.0"

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	applogger.Setup(cfg.App.Env, cfg.Log.Level, cfg.Log.FilePath, cfg.Log.ToFile, cfg.App.Name)
	log.Info().Str("version", version).Str("env", cfg.App.Env).Msg("starting service")

	db, err := database.NewConnection(&cfg.DB)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get underlying sql.DB")
	}

	if cfg.DB.AutoMigrate {
		if err := db.AutoMigrate(&entity.User{}); err != nil {
			log.Fatal().Err(err).Msg("auto-migrate failed")
		}
	}

	redisClient := newRedisClient(&cfg.Redis)

	userRepo := pgRepo.NewUserRepository(db)
	userUC := usecase.NewUserUseCase(userRepo)
	val := validator.New()

	startTime := time.Now()
	userHandler := handler.NewUserHandler(userUC, val)
	healthHandler := handler.NewHealthHandler(sqlDB, redisClient, startTime, version)

	if cfg.Kafka.Brokers != "" {
		healthHandler.AddChecker("kafka", kafkamsg.HealthCheck(cfg.Kafka.Brokers))
	} else {
		healthHandler.AddChecker("kafka", func(_ context.Context) string { return "not_configured" })
	}

	if cfg.RabbitMQ.DSN != "" {
		healthHandler.AddChecker("rabbitmq", rabbitmqmsg.HealthCheck(cfg.RabbitMQ.DSN))
	} else {
		healthHandler.AddChecker("rabbitmq", func(_ context.Context) string { return "not_configured" })
	}

	app := fiber.New(fiber.Config{
		AppName:               cfg.App.Name,
		BodyLimit:             4 * 1024 * 1024,
		ReadTimeout:           10 * time.Second,
		WriteTimeout:          10 * time.Second,
		IdleTimeout:           120 * time.Second,
		DisableStartupMessage: true,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"status":  false,
				"message": http.StatusText(code),
			})
		},
	})

	app.Use(mw.Recover())
	app.Use(mw.RequestID())
	app.Use(mw.SecurityHeaders())
	app.Use(mw.RateLimiter())
	app.Use(mw.RequestLogger())
	app.Use(mw.CORS(cfg.App.CORSAllowedOrigins))

	route.Setup(app, userHandler, healthHandler)

	go func() {
		addr := ":" + cfg.App.Port
		log.Info().Str("addr", addr).Msg("http server listening")
		if err := app.Listen(addr); err != nil {
			log.Fatal().Err(err).Msg("http server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutdown signal received")

	shutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(shutCtx); err != nil {
		log.Error().Err(err).Msg("http server forced shutdown")
	}

	redisClient.Close() //nolint:errcheck
	sqlDB.Close()       //nolint:errcheck

	log.Info().Msg("shutdown complete")
}

func newRedisClient(cfg *config.RedisConfig) *redis.Client {
	opts, err := redis.ParseURL(cfg.URL)
	if err != nil {
		opts = &redis.Options{Addr: "localhost:6379"}
	}
	if cfg.Password != "" {
		opts.Password = cfg.Password
	}
	if cfg.DB > 0 {
		opts.DB = cfg.DB
	}
	return redis.NewClient(opts)
}
