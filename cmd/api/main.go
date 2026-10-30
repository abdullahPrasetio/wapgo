// @title           wapgo API
// @version         1.0.0
// @description     Web API Platform for Go — production-ready microservice boilerplate.
// @contact.name    wapgo
// @contact.url     https://github.com/abdullahPrasetio/wapgo
// @license.name    MIT
// @license.url     https://opensource.org/licenses/MIT
// @BasePath        /api/v1
// @schemes         http https
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
// @description     Bearer <token>
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"github.com/abdullahPrasetio/wapgo/config"
	_ "github.com/abdullahPrasetio/wapgo/docs" // swagger generated docs
	"github.com/abdullahPrasetio/wapgo/internal/delivery/http/handler"
	mw "github.com/abdullahPrasetio/wapgo/internal/delivery/http/middleware"
	"github.com/abdullahPrasetio/wapgo/internal/delivery/http/route"
	"github.com/abdullahPrasetio/wapgo/internal/domain/entity"
	domainrepo "github.com/abdullahPrasetio/wapgo/internal/domain/repository"
	dbrepo "github.com/abdullahPrasetio/wapgo/internal/repository/db"
	memoryrepo "github.com/abdullahPrasetio/wapgo/internal/repository/memory"
	redisrepo "github.com/abdullahPrasetio/wapgo/internal/repository/redis"
	"github.com/abdullahPrasetio/wapgo/internal/usecase"
	"github.com/abdullahPrasetio/wapgo/pkg/auth"
	"github.com/abdullahPrasetio/wapgo/pkg/database"
	applogger "github.com/abdullahPrasetio/wapgo/pkg/logger"
	kafkamsg "github.com/abdullahPrasetio/wapgo/pkg/messaging/kafka"
	rabbitmqmsg "github.com/abdullahPrasetio/wapgo/pkg/messaging/rabbitmq"
	"github.com/abdullahPrasetio/wapgo/pkg/observability"
	"github.com/abdullahPrasetio/wapgo/pkg/ratelimit"
	"github.com/abdullahPrasetio/wapgo/pkg/validator"
)

const version = "1.0.0"

func main() {
	// ── Config ───────────────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	// ── Logger ───────────────────────────────────────────────────────────────
	applogger.Setup(cfg.App.Env, cfg.Log.Level, cfg.Log.FilePath, cfg.Log.ToFile, cfg.App.Name)
	log.Info().Str("version", version).Str("env", cfg.App.Env).
		Str("observability", cfg.Observability.Provider).Msg("starting wapgo")

	// ── Observability provider ────────────────────────────────────────────────
	obsProvider, err := observability.New(context.Background(), &cfg.Observability, cfg.App.Name, version, cfg.App.Env)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to setup observability provider")
	}

	// ── Database ─────────────────────────────────────────────────────────────
	db, err := database.NewConnection(&cfg.DB)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	if err := obsProvider.InstrumentGORM(db); err != nil {
		log.Warn().Err(err).Msg("gorm instrumentation failed")
	}
	// v1.3: enforce per-query deadline from config.
	queryTimeout, err := time.ParseDuration(cfg.DB.QueryTimeout)
	if err != nil || queryTimeout <= 0 {
		queryTimeout = 5 * time.Second
	}
	if err := db.Use(database.NewQueryTimeoutPlugin(queryTimeout)); err != nil {
		log.Warn().Err(err).Msg("query timeout plugin registration failed")
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get underlying sql.DB")
	}
	log.Info().Str("driver", cfg.DB.Driver).Msg("database connected")

	if cfg.DB.AutoMigrate {
		if cfg.App.Env == "production" {
			log.Fatal().Msg("DB_AUTO_MIGRATE is not allowed in production; run ./cmd/migrate instead")
		}
		log.Warn().Msg("DB_AUTO_MIGRATE is enabled: development only, use ./cmd/migrate for real schema changes")
		if err := db.AutoMigrate(&entity.User{}); err != nil {
			log.Fatal().Err(err).Msg("auto-migrate failed")
		}
		log.Info().Msg("auto-migrate complete")
	}

	// ── Redis (optional) ─────────────────────────────────────────────────────
	// Redis is an optimisation, not a requirement. Without REDIS_URL the service
	// uses in-process fallbacks; token revocation then only covers this process.
	var redisClient *redis.Client
	var cacher domainrepo.Cacher
	var blacklist auth.Blacklist
	redisEnabled := cfg.Redis.Enabled()
	if redisEnabled {
		redisClient = newRedisClient(&cfg.Redis)
		obsProvider.InstrumentRedis(redisClient)
		cacher = redisrepo.New(redisClient, "wapgo")
		blacklist = auth.NewRedisBlacklist(redisClient)
		log.Info().Msg("redis enabled: cache and token revocation are shared")
	} else {
		cacher = memoryrepo.New("wapgo")
		blacklist = auth.NewMemoryBlacklist()
		log.Warn().Msg("REDIS_URL is empty: using in-memory cache and token blacklist; " +
			"token revocation and refresh sessions are instance-local and lost on restart")
	}

	// ── Repositories ─────────────────────────────────────────────────────────
	userRepo := dbrepo.NewUserRepository(db)

	// ── Auth ─────────────────────────────────────────────────────────────────
	jwtExpiry, err := time.ParseDuration(cfg.JWT.Expiry)
	if err != nil {
		jwtExpiry = 15 * time.Minute
	}
	refreshExpiry, err := time.ParseDuration(cfg.JWT.RefreshExpiry)
	if err != nil {
		refreshExpiry = 7 * 24 * time.Hour
	}
	jwtCfg := &auth.Config{
		Secret:   cfg.JWT.Secret,
		Issuer:   cfg.JWT.Issuer,
		Audience: cfg.JWT.Audience,
		Expiry:   jwtExpiry,
	}

	bcryptCost := cfg.App.BcryptCost
	if bcryptCost < 10 {
		bcryptCost = 12
	}

	// ── Usecases ─────────────────────────────────────────────────────────────
	userUC := usecase.NewUserUseCase(userRepo)
	authUC := usecase.NewAuthUseCase(userRepo, cacher, jwtCfg, refreshExpiry, blacklist, bcryptCost)

	// ── Validators / Handlers ─────────────────────────────────────────────────
	val := validator.New()
	startTime := time.Now()
	userHandler := handler.NewUserHandler(userUC, val)
	authHandler := handler.NewAuthHandler(authUC, val, cfg.App.Env)
	probeTimeout, _ := time.ParseDuration(cfg.Health.ProbeTimeout)
	healthHandler := handler.NewHealthHandler(sqlDB, startTime, version, probeTimeout)
	if redisEnabled {
		healthHandler.AddChecker("redis", func(ctx context.Context) string {
			pingCtx, cancel := context.WithTimeout(ctx, probeTimeout)
			defer cancel()
			if err := redisClient.Ping(pingCtx).Err(); err != nil {
				return "down"
			}
			return "ok"
		})
	}

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

	// ── Rate limiter ─────────────────────────────────────────────────────────
	// Memory is the safe default: per-instance, no external dependency. Redis
	// shares one quota across replicas; without REDIS_URL we degrade to memory.
	rlWindow, err := time.ParseDuration(cfg.RateLimit.Window)
	if err != nil || rlWindow <= 0 {
		rlWindow = time.Minute
	}
	var globalLimiter ratelimit.Limiter
	if cfg.RateLimit.Backend == "redis" {
		if redisEnabled {
			globalLimiter = ratelimit.NewRedis(redisClient, cfg.RateLimit.Prefix, cfg.RateLimit.Max, rlWindow)
			log.Info().Str("backend", "redis").Int("max", cfg.RateLimit.Max).
				Dur("window", rlWindow).Msg("shared rate limiting enabled")
		} else {
			log.Warn().Msg("RATE_LIMIT_BACKEND=redis requires REDIS_URL; falling back to per-instance in-memory rate limiting")
		}
	}
	if globalLimiter == nil {
		globalLimiter = ratelimit.NewMemory(cfg.RateLimit.Max, rlWindow)
		log.Info().Str("backend", "memory").Int("max", cfg.RateLimit.Max).
			Dur("window", rlWindow).Msg("rate limiting is per-instance (not shared across replicas)")
	}

	// ── Fiber app ────────────────────────────────────────────────────────────
	trustedProxies := parseTrustedProxies(cfg.App.TrustedProxies)
	app := fiber.New(fiber.Config{
		AppName:               cfg.App.Name,
		BodyLimit:             4 * 1024 * 1024,
		ReadTimeout:           10 * time.Second,
		WriteTimeout:          10 * time.Second,
		IdleTimeout:           120 * time.Second,
		DisableStartupMessage: true,
		// Trusted proxies: use X-Forwarded-For only from known load balancers/ingress.
		TrustedProxies:          trustedProxies,
		EnableTrustedProxyCheck: len(trustedProxies) > 0,
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

	// Middleware stack (order matters)
	app.Use(mw.Recover())
	app.Use(mw.RequestID())
	app.Use(mw.SecurityHeaders())
	app.Use(mw.RateLimiterWith(globalLimiter))
	app.Use(mw.RequestLogger())
	app.Use(mw.CORS(cfg.App.CORSAllowedOrigins))
	app.Use(obsProvider.HTTPMiddleware())
	app.Use(observability.MetricsMiddleware())

	route.Setup(app, userHandler, authHandler, healthHandler, jwtCfg, blacklist, cfg.App.Env)

	// ── Start server ─────────────────────────────────────────────────────────
	go func() {
		addr := ":" + cfg.App.Port
		log.Info().Str("addr", addr).Msg("http server listening")
		if err := app.Listen(addr); err != nil {
			log.Fatal().Err(err).Msg("http server error")
		}
	}()

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutdown signal received")
	shutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(shutCtx); err != nil {
		log.Error().Err(err).Msg("http server forced shutdown")
	}
	log.Info().Msg("http server stopped")

	if err := obsProvider.Shutdown(shutCtx); err != nil {
		log.Error().Err(err).Msg("observability provider shutdown error")
	}
	if redisClient != nil {
		if err := redisClient.Close(); err != nil {
			log.Error().Err(err).Msg("redis close error")
		}
	}
	if err := sqlDB.Close(); err != nil {
		log.Error().Err(err).Msg("database close error")
	}
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

	if cfg.PoolSize > 0 {
		opts.PoolSize = cfg.PoolSize
	}
	if cfg.MinIdleConns > 0 {
		opts.MinIdleConns = cfg.MinIdleConns
	}
	if cfg.MaxRetries > 0 {
		opts.MaxRetries = cfg.MaxRetries
	}
	if d, err := time.ParseDuration(cfg.DialTimeout); err == nil {
		opts.DialTimeout = d
	}
	if d, err := time.ParseDuration(cfg.ReadTimeout); err == nil {
		opts.ReadTimeout = d
	}
	if d, err := time.ParseDuration(cfg.WriteTimeout); err == nil {
		opts.WriteTimeout = d
	}
	return redis.NewClient(opts)
}

// parseTrustedProxies splits a comma-separated string of proxy IPs/CIDRs.
func parseTrustedProxies(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
