package observability

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/migrator"
	"gorm.io/gorm/schema"
)

// internalMockDialector satisfies gorm.Dialector for internal test use.
type internalMockDialector struct{}

func (internalMockDialector) Name() string                                                          { return "mock" }
func (internalMockDialector) Initialize(*gorm.DB) error                                             { return nil }
func (internalMockDialector) Migrator(db *gorm.DB) gorm.Migrator                                   { return migrator.Migrator{Config: migrator.Config{DB: db}} }
func (internalMockDialector) DataTypeOf(*schema.Field) string                                       { return "text" }
func (internalMockDialector) DefaultValueOf(*schema.Field) clause.Expression                        { return clause.Expr{SQL: "NULL"} }
func (internalMockDialector) BindVarTo(w clause.Writer, _ *gorm.Statement, _ interface{})           { w.WriteByte('?') } //nolint:errcheck
func (internalMockDialector) QuoteTo(w clause.Writer, s string)                                     { w.WriteString(s) } //nolint:errcheck
func (internalMockDialector) Explain(sql string, _ ...interface{}) string                           { return sql }

// TestElasticGORMPlugin_Callbacks exercises the before/after closures registered
// by elasticGORMPlugin.Initialize via GORM DryRun mode.
func TestElasticGORMPlugin_Callbacks(t *testing.T) {
	db, err := gorm.Open(internalMockDialector{}, &gorm.Config{})
	assert.NoError(t, err)

	plugin := &elasticGORMPlugin{}
	assert.NoError(t, plugin.Initialize(db))

	// DryRun builds SQL and fires callbacks without hitting a real database.
	ctx := context.Background()
	session := db.Session(&gorm.Session{DryRun: true, Context: ctx})

	// Query callback — triggers before_query + after_query
	session.Find(&struct{}{})
	// Raw callback
	session.Raw("SELECT 1").Scan(&struct{}{})
}

// TestElasticGORMPlugin_Callbacks_AfterWithError exercises the error path in `after`.
func TestElasticGORMPlugin_Callbacks_AfterWithError(t *testing.T) {
	db, err := gorm.Open(internalMockDialector{}, &gorm.Config{})
	assert.NoError(t, err)

	plugin := &elasticGORMPlugin{}
	assert.NoError(t, plugin.Initialize(db))

	// Execute a query in DryRun — error is set to gorm.ErrRecordNotFound.
	ctx := context.Background()
	session := db.Session(&gorm.Session{DryRun: true, Context: ctx})
	session.First(&struct{}{}) // First sets ErrRecordNotFound when result is empty
}

// TestElasticRedisHook_ProcessHook exercises the ProcessHook and ProcessPipelineHook
// closures by running real commands against a miniredis server.
func TestElasticRedisHook_ProcessHook(t *testing.T) {
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer client.Close() //nolint:errcheck

	hook := &elasticRedisHook{}
	client.AddHook(hook)

	ctx := context.Background()

	// ProcessHook — normal command
	err = client.Set(ctx, "key", "val", 0).Err()
	assert.NoError(t, err)

	_, err = client.Get(ctx, "key").Result()
	assert.NoError(t, err)

	// ProcessHook — redis.Nil (not-found) should NOT be reported as error
	_, err = client.Get(ctx, "missing").Result()
	assert.ErrorIs(t, err, redis.Nil)
}

func TestElasticRedisHook_ProcessPipelineHook(t *testing.T) {
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer client.Close() //nolint:errcheck

	hook := &elasticRedisHook{}
	client.AddHook(hook)

	// ProcessPipelineHook
	ctx := context.Background()
	pipe := client.Pipeline()
	pipe.Set(ctx, "p1", "v1", 0)
	pipe.Get(ctx, "p1")
	_, err = pipe.Exec(ctx)
	assert.NoError(t, err)
}
