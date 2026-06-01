package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	App          AppConfig          `mapstructure:"app"`
	DB           DBConfig           `mapstructure:"db"`
	Redis        RedisConfig        `mapstructure:"redis"`
	Kafka        KafkaConfig        `mapstructure:"kafka"`
	RabbitMQ     RabbitMQConfig     `mapstructure:"rabbitmq"`
	Log          LogConfig          `mapstructure:"log"`
	Services     ServiceURLs        `mapstructure:"services"`
	JWT          JWTConfig          `mapstructure:"jwt"`
	Observability ObservabilityConfig `mapstructure:"observability"`
}

// JWTConfig holds parameters for HS256 token signing and verification.
type JWTConfig struct {
	Secret   string `mapstructure:"secret"`   // JWT_SECRET — must be ≥32 bytes in production
	Issuer   string `mapstructure:"issuer"`   // JWT_ISSUER
	Audience string `mapstructure:"audience"` // JWT_AUDIENCE
	Expiry   string `mapstructure:"expiry"`   // JWT_EXPIRY — Go duration string, e.g. "24h"
}

// ObservabilityConfig controls which observability backend is used.
//
// Provider chooses the backend:
//   - "otel"        (default) — OpenTelemetry SDK, OTLP export
//   - "elastic_apm" — Elastic APM Go agent; agent ENV vars (ELASTIC_APM_*)
//     are read automatically by the agent and do not need to appear here.
type ObservabilityConfig struct {
	Provider       string `mapstructure:"provider"`         // OBSERVABILITY_PROVIDER
	TracingEnabled bool   `mapstructure:"tracing_enabled"`  // OTEL_TRACING_ENABLED (otel only)
	OTLPEndpoint   string `mapstructure:"otlp_endpoint"`    // OTEL_EXPORTER_OTLP_ENDPOINT (otel only)
}

type AppConfig struct {
	Env                string `mapstructure:"env"`
	Port               string `mapstructure:"port"`
	Name               string `mapstructure:"name"`
	CORSAllowedOrigins string `mapstructure:"cors_allowed_origins"`
}

type DBConfig struct {
	Driver       string `mapstructure:"driver"`
	Host         string `mapstructure:"host"`
	Port         string `mapstructure:"port"`
	Name         string `mapstructure:"name"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	ConnMaxLife  string `mapstructure:"conn_max_life"` // e.g. "5m"
	AutoMigrate  bool   `mapstructure:"auto_migrate"`
	SSLMode      string `mapstructure:"sslmode"`
}

type RedisConfig struct {
	URL      string `mapstructure:"url"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type KafkaConfig struct {
	Brokers string `mapstructure:"brokers"`
	GroupID string `mapstructure:"group_id"`
}

type RabbitMQConfig struct {
	DSN      string `mapstructure:"dsn"`
	Exchange string `mapstructure:"exchange"`
}

type LogConfig struct {
	Level    string `mapstructure:"level"`
	ToFile   bool   `mapstructure:"to_file"`
	FilePath string `mapstructure:"file_path"`
}

func Load() (*Config, error) {
	v := viper.New()

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./config")
	v.AddConfigPath(".")

	// ENV vars override yaml — highest priority
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Explicit bindings ensure ENV names like DB_MAX_OPEN_CONNS map correctly
	bindings := map[string]string{
		"app.env":                  "APP_ENV",
		"app.port":                 "APP_PORT",
		"app.name":                 "APP_NAME",
		"app.cors_allowed_origins": "APP_CORS_ALLOWED_ORIGINS",
		"db.driver":                "DB_DRIVER",
		"db.host":                  "DB_HOST",
		"db.port":                  "DB_PORT",
		"db.name":                  "DB_NAME",
		"db.user":                  "DB_USER",
		"db.password":              "DB_PASSWORD",
		"db.max_open_conns":        "DB_MAX_OPEN_CONNS",
		"db.max_idle_conns":        "DB_MAX_IDLE_CONNS",
		"db.conn_max_life":         "DB_CONN_MAX_LIFE",
		"db.auto_migrate":          "DB_AUTO_MIGRATE",
		"db.sslmode":               "DB_SSL_MODE",
		"redis.url":                "REDIS_URL",
		"redis.password":           "REDIS_PASSWORD",
		"redis.db":                 "REDIS_DB",
		"kafka.brokers":            "KAFKA_BROKERS",
		"kafka.group_id":           "KAFKA_GROUP_ID",
		"rabbitmq.dsn":             "RABBITMQ_DSN",
		"rabbitmq.exchange":        "RABBITMQ_EXCHANGE",
		"log.level":                "LOG_LEVEL",
		"log.to_file":              "LOG_TO_FILE",
		"log.file_path":            "LOG_FILE_PATH",
		"services.user_url":                "USER_SERVICE_URL",
		"services.order_url":               "ORDER_SERVICE_URL",
		"jwt.secret":                        "JWT_SECRET",
		"jwt.issuer":                        "JWT_ISSUER",
		"jwt.audience":                      "JWT_AUDIENCE",
		"jwt.expiry":                        "JWT_EXPIRY",
		"observability.provider":             "OBSERVABILITY_PROVIDER",
		"observability.tracing_enabled":      "OTEL_TRACING_ENABLED",
		"observability.otlp_endpoint":        "OTEL_EXPORTER_OTLP_ENDPOINT",
	}
	for key, env := range bindings {
		if err := v.BindEnv(key, env); err != nil {
			return nil, fmt.Errorf("binding env %s: %w", env, err)
		}
	}

	// Defaults
	v.SetDefault("app.env", "development")
	v.SetDefault("app.port", "8080")
	v.SetDefault("app.name", "wapgo-service")
	v.SetDefault("app.cors_allowed_origins", "http://localhost:3000")
	v.SetDefault("db.driver", "postgres")
	v.SetDefault("db.host", "localhost")
	v.SetDefault("db.port", "5432")
	v.SetDefault("db.max_open_conns", 25)
	v.SetDefault("db.max_idle_conns", 5)
	v.SetDefault("db.conn_max_life", "5m")
	v.SetDefault("db.auto_migrate", false)
	v.SetDefault("db.sslmode", "disable")
	v.SetDefault("redis.url", "redis://localhost:6379")
	v.SetDefault("redis.db", 0)
	v.SetDefault("log.level", "info")
	v.SetDefault("log.to_file", false)
	v.SetDefault("log.file_path", "logs/app.log")
	v.SetDefault("jwt.issuer", "wapgo-service")
	v.SetDefault("jwt.audience", "wapgo-api")
	v.SetDefault("jwt.expiry", "24h")
	v.SetDefault("observability.provider", "otel")
	v.SetDefault("observability.tracing_enabled", false)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshalling config: %w", err)
	}

	return &cfg, nil
}
