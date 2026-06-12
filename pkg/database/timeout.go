package database

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type cancelKey struct{}

// QueryTimeoutPlugin enforces a per-statement deadline on every GORM operation.
// Register once after opening the connection:
//
//	db.Use(database.NewQueryTimeoutPlugin(5 * time.Second))
//
// Statements that already carry a shorter deadline are not affected.
type QueryTimeoutPlugin struct {
	timeout time.Duration
}

// NewQueryTimeoutPlugin creates the plugin. Pass 0 to use the default 5 s.
func NewQueryTimeoutPlugin(timeout time.Duration) *QueryTimeoutPlugin {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &QueryTimeoutPlugin{timeout: timeout}
}

func (p *QueryTimeoutPlugin) Name() string { return "QueryTimeoutPlugin" }

func (p *QueryTimeoutPlugin) Initialize(db *gorm.DB) error {
	// Silently ignore duplicate-registration errors (idempotent).
	_ = db.Callback().Query().Before("gorm:query").Register("qt:before:query", p.before)
	_ = db.Callback().Query().After("gorm:query").Register("qt:after:query", p.after)
	_ = db.Callback().Create().Before("gorm:create").Register("qt:before:create", p.before)
	_ = db.Callback().Create().After("gorm:create").Register("qt:after:create", p.after)
	_ = db.Callback().Update().Before("gorm:update").Register("qt:before:update", p.before)
	_ = db.Callback().Update().After("gorm:update").Register("qt:after:update", p.after)
	_ = db.Callback().Delete().Before("gorm:delete").Register("qt:before:delete", p.before)
	_ = db.Callback().Delete().After("gorm:delete").Register("qt:after:delete", p.after)
	_ = db.Callback().Row().Before("gorm:row").Register("qt:before:row", p.before)
	_ = db.Callback().Row().After("gorm:row").Register("qt:after:row", p.after)
	_ = db.Callback().Raw().Before("gorm:raw").Register("qt:before:raw", p.before)
	_ = db.Callback().Raw().After("gorm:raw").Register("qt:after:raw", p.after)
	return nil
}

func (p *QueryTimeoutPlugin) before(db *gorm.DB) {
	if db.Statement == nil {
		return
	}
	base := db.Statement.Context
	if base == nil {
		base = context.Background()
	}
	if _, hasDeadline := base.Deadline(); hasDeadline {
		return // caller already controls the deadline
	}
	ctx, cancel := context.WithTimeout(base, p.timeout)
	db.Statement.Context = context.WithValue(ctx, cancelKey{}, cancel)
}

func (p *QueryTimeoutPlugin) after(db *gorm.DB) {
	if db.Statement == nil {
		return
	}
	if cancel, ok := db.Statement.Context.Value(cancelKey{}).(context.CancelFunc); ok {
		cancel()
	}
}
