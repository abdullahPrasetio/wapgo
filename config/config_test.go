package config_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abdullahPrasetio/wapgo/config"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear any previously set env vars that would bleed in
	os.Unsetenv("APP_ENV")
	os.Unsetenv("APP_PORT")

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "development", cfg.App.Env)
	assert.Equal(t, "8080", cfg.App.Port)
	assert.Equal(t, "wapgo-service", cfg.App.Name)
}

func TestLoad_ENVOverride(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_PORT", "9090")
	t.Setenv("APP_NAME", "my-service")

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "production", cfg.App.Env)
	assert.Equal(t, "9090", cfg.App.Port)
	assert.Equal(t, "my-service", cfg.App.Name)
}

func TestLoad_DBDefaults(t *testing.T) {
	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "postgres", cfg.DB.Driver)
	assert.Equal(t, 25, cfg.DB.MaxOpenConns)
	assert.Equal(t, "5m", cfg.DB.ConnMaxLife)
}

func TestLoad_DBEnvOverride(t *testing.T) {
	t.Setenv("DB_DRIVER", "mysql")
	t.Setenv("DB_MAX_OPEN_CONNS", "50")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "mysql", cfg.DB.Driver)
	assert.Equal(t, 50, cfg.DB.MaxOpenConns)
}

func TestLoad_LogDefaults(t *testing.T) {
	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "info", cfg.Log.Level)
	assert.False(t, cfg.Log.ToFile)
}

func TestLoad_ServiceURLs(t *testing.T) {
	t.Setenv("USER_SERVICE_URL", "http://my-user-svc:8080")
	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "http://my-user-svc:8080", cfg.Services.UserURL)
}
