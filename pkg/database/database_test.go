package database

import (
	"database/sql"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abdullahPrasetio/wapgo/config"
)

// ── buildDialector ────────────────────────────────────────────────────────────

func TestBuildDialector_MySQL(t *testing.T) {
	cfg := &config.DBConfig{Driver: "mysql", Host: "localhost", Port: "3306", User: "root", Password: "pass", Name: "db"}
	d, err := buildDialector(cfg)
	require.NoError(t, err)
	assert.NotNil(t, d)
	assert.Equal(t, "mysql", d.Name())
}

func TestBuildDialector_MySQLWithTLS(t *testing.T) {
	cfg := &config.DBConfig{Driver: "mysql", Host: "localhost", Port: "3306", User: "root", Password: "pass", Name: "db", SSLMode: "require"}
	d, err := buildDialector(cfg)
	require.NoError(t, err)
	assert.NotNil(t, d)
}

func TestBuildDialector_Postgres(t *testing.T) {
	cfg := &config.DBConfig{Driver: "postgres", Host: "localhost", Port: "5432", User: "pg", Password: "pass", Name: "db"}
	d, err := buildDialector(cfg)
	require.NoError(t, err)
	assert.NotNil(t, d)
	assert.Equal(t, "postgres", d.Name())
}

func TestBuildDialector_EmptyDriverDefaultsToPostgres(t *testing.T) {
	cfg := &config.DBConfig{Driver: "", Host: "localhost", Port: "5432", User: "pg", Password: "pass", Name: "db"}
	d, err := buildDialector(cfg)
	require.NoError(t, err)
	assert.Equal(t, "postgres", d.Name())
}

func TestBuildDialector_UnsupportedDriver(t *testing.T) {
	cfg := &config.DBConfig{Driver: "sqlite"}
	_, err := buildDialector(cfg)
	assert.ErrorContains(t, err, "unsupported DB driver")
}

// ── configurePool ─────────────────────────────────────────────────────────────

func newSQLMockDB(t *testing.T) *sql.DB {
	t.Helper()
	sqlDB, _, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { sqlDB.Close() })
	return sqlDB
}

func TestConfigurePool_AllFields(t *testing.T) {
	cfg := &config.DBConfig{MaxOpenConns: 10, MaxIdleConns: 5, ConnMaxLife: "5m"}
	err := configurePool(newSQLMockDB(t), cfg)
	require.NoError(t, err)
}

func TestConfigurePool_Defaults(t *testing.T) {
	err := configurePool(newSQLMockDB(t), &config.DBConfig{})
	require.NoError(t, err)
}

func TestConfigurePool_InvalidDuration(t *testing.T) {
	cfg := &config.DBConfig{ConnMaxLife: "not-a-duration"}
	err := configurePool(newSQLMockDB(t), cfg)
	assert.ErrorContains(t, err, "invalid conn_max_life")
}

// ── NewConnection ─────────────────────────────────────────────────────────────

func TestNewConnection_UnsupportedDriver(t *testing.T) {
	cfg := &config.DBConfig{Driver: "oracle"}
	_, err := NewConnection(cfg)
	assert.ErrorContains(t, err, "unsupported DB driver")
}
