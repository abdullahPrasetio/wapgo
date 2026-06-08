package database

import (
	"database/sql"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gormpostgres "gorm.io/driver/postgres"

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

func TestBuildDialector_MissingHost(t *testing.T) {
	cfg := &config.DBConfig{Driver: "postgres", Host: "", Name: "db", User: "pg"}
	_, err := buildDialector(cfg)
	assert.ErrorContains(t, err, "DB_HOST is required")
}

func TestBuildDialector_MissingName(t *testing.T) {
	cfg := &config.DBConfig{Driver: "postgres", Host: "localhost", Name: "", User: "pg"}
	_, err := buildDialector(cfg)
	assert.ErrorContains(t, err, "DB_NAME is required")
}

func TestBuildDialector_MissingUser(t *testing.T) {
	cfg := &config.DBConfig{Driver: "postgres", Host: "localhost", Name: "db", User: ""}
	_, err := buildDialector(cfg)
	assert.ErrorContains(t, err, "DB_USER is required")
}

func TestBuildDialector_MySQLDefaultPort(t *testing.T) {
	cfg := &config.DBConfig{Driver: "mysql", Host: "localhost", Port: "", User: "root", Password: "pass", Name: "db"}
	d, err := buildDialector(cfg)
	require.NoError(t, err)
	assert.Equal(t, "mysql", d.Name())
}

func TestBuildDialector_PostgresDefaultPort(t *testing.T) {
	cfg := &config.DBConfig{Driver: "postgres", Host: "localhost", Port: "", User: "pg", Password: "pass", Name: "db"}
	d, err := buildDialector(cfg)
	require.NoError(t, err)
	assert.Equal(t, "postgres", d.Name())
}

func TestBuildDialector_PostgresSSLMode(t *testing.T) {
	cfg := &config.DBConfig{Driver: "postgres", Host: "localhost", Port: "5432", User: "pg", Password: "pass", Name: "db", SSLMode: "require"}
	d, err := buildDialector(cfg)
	require.NoError(t, err)
	assert.Equal(t, "postgres", d.Name())
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

func TestNewConnection_BuildDialectorError(t *testing.T) {
	cfg := &config.DBConfig{Driver: "postgres", Host: "", Name: "db", User: "pg"}
	_, err := NewConnection(cfg)
	assert.ErrorContains(t, err, "DB_HOST is required")
}

// ── openWithDialector ─────────────────────────────────────────────────────────

func TestOpenWithDialector_Success(t *testing.T) {
	sqlDB, _, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { sqlDB.Close() })

	dialector := gormpostgres.New(gormpostgres.Config{Conn: sqlDB})
	cfg := &config.DBConfig{MaxOpenConns: 5, MaxIdleConns: 2, ConnMaxLife: "10m"}
	db, err := openWithDialector(dialector, cfg)
	require.NoError(t, err)
	assert.NotNil(t, db)
}

func TestOpenWithDialector_PoolError(t *testing.T) {
	sqlDB, _, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { sqlDB.Close() })

	dialector := gormpostgres.New(gormpostgres.Config{Conn: sqlDB})
	cfg := &config.DBConfig{ConnMaxLife: "bad-duration"}
	_, err = openWithDialector(dialector, cfg)
	assert.ErrorContains(t, err, "invalid conn_max_life")
}

