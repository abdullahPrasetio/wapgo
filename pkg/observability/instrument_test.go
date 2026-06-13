package observability_test

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/migrator"
	"gorm.io/gorm/schema"

	"github.com/abdullahPrasetio/wapgo/config"
	"github.com/abdullahPrasetio/wapgo/pkg/observability"
)

// mockDialector satisfies gorm.Dialector without a real database.
type mockDialector struct{}

func (mockDialector) Name() string                                                    { return "mock" }
func (mockDialector) Initialize(*gorm.DB) error                                       { return nil }
func (mockDialector) Migrator(db *gorm.DB) gorm.Migrator                              { return migrator.Migrator{Config: migrator.Config{DB: db}} }
func (mockDialector) DataTypeOf(*schema.Field) string                                 { return "text" }
func (mockDialector) DefaultValueOf(*schema.Field) clause.Expression                  { return clause.Expr{SQL: "NULL"} }
func (mockDialector) BindVarTo(w clause.Writer, _ *gorm.Statement, _ interface{})     { w.WriteByte('?') } //nolint:errcheck
func (mockDialector) QuoteTo(w clause.Writer, s string)                               { w.WriteString(s) } //nolint:errcheck
func (mockDialector) Explain(sql string, _ ...interface{}) string                     { return sql }

// openMockDB returns a gorm.DB that is safe for callback/plugin registration tests.
func openMockDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(mockDialector{}, &gorm.Config{})
	require.NoError(t, err)
	return db
}

// newRedisClientForTest returns a redis.Client that never dials — safe for hook attachment.
func newRedisClientForTest() *redis.Client {
	return redis.NewClient(&redis.Options{Addr: "localhost:63790"}) // unused port; hooks attach synchronously
}

// ── OTel provider: InstrumentGORM + InstrumentRedis ───────────────────────────

func TestOTelProvider_InstrumentGORM(t *testing.T) {
	cfg := &config.ObservabilityConfig{Provider: "otel", TracingEnabled: false}
	p, err := observability.New(context.Background(), cfg, "svc", "0.0.0", "test")
	require.NoError(t, err)

	db := openMockDB(t)
	assert.NoError(t, p.InstrumentGORM(db))
}

func TestOTelProvider_InstrumentRedis(t *testing.T) {
	cfg := &config.ObservabilityConfig{Provider: "otel", TracingEnabled: false}
	p, err := observability.New(context.Background(), cfg, "svc", "0.0.0", "test")
	require.NoError(t, err)

	client := newRedisClientForTest()
	defer client.Close() //nolint:errcheck
	// Must not panic — hooks are attached synchronously.
	assert.NotPanics(t, func() { p.InstrumentRedis(client) })
}

// ── Elastic APM provider: InstrumentGORM + InstrumentRedis ────────────────────

func TestElasticProvider_InstrumentGORM(t *testing.T) {
	cfg := &config.ObservabilityConfig{Provider: "elastic_apm"}
	p, err := observability.New(context.Background(), cfg, "svc", "0.0.0", "test")
	require.NoError(t, err)

	db := openMockDB(t)
	assert.NoError(t, p.InstrumentGORM(db))
}

func TestElasticProvider_InstrumentRedis(t *testing.T) {
	cfg := &config.ObservabilityConfig{Provider: "elastic_apm"}
	p, err := observability.New(context.Background(), cfg, "svc", "0.0.0", "test")
	require.NoError(t, err)

	client := newRedisClientForTest()
	defer client.Close() //nolint:errcheck
	assert.NotPanics(t, func() { p.InstrumentRedis(client) })
}

// ── fiberHeaderCarrier (tracing.go internal) is exercised via HTTPMiddleware tests.
// Here we verify Set/Keys paths via the otel HTTPMiddleware span extraction.
// Those paths are hit when TracingMiddleware extracts a traceparent header.
// See tracing_test.go: TestOTelProvider_HTTPMiddleware_PropagatesW3CHeaders.
