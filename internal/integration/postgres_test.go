//go:build integration

// Integration tests for the Postgres user repository.
// Requires Docker. Run with: go test -tags=integration ./internal/integration/...
package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/abdullahPrasetio/wapgo/internal/domain/entity"
	dbrepo "github.com/abdullahPrasetio/wapgo/internal/repository/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresUserRepository(t *testing.T) {
	ctx := context.Background()

	// ── Spin up Postgres container ────────────────────────────────────────────
	pgContainer, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("testuser"),
		tcpostgres.WithPassword("testpass"),
		tcpostgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)
	t.Cleanup(func() { pgContainer.Terminate(ctx) }) //nolint:errcheck

	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// ── Open GORM connection ──────────────────────────────────────────────────
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Auto-migrate the users table
	require.NoError(t, db.AutoMigrate(&entity.User{}))

	repo := dbrepo.NewUserRepository(db)

	// ── Create ────────────────────────────────────────────────────────────────
	t.Run("create and find by id", func(t *testing.T) {
		user := &entity.User{
			Name:     "Alice",
			Email:    fmt.Sprintf("alice+%d@example.com", time.Now().UnixNano()),
			Password: "hashed",
		}
		require.NoError(t, repo.Create(ctx, user))
		assert.NotEmpty(t, user.ID)

		found, err := repo.FindByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, user.Name, found.Name)
		assert.Equal(t, user.Email, found.Email)
	})

	// ── FindAll ───────────────────────────────────────────────────────────────
	t.Run("find all", func(t *testing.T) {
		users, err := repo.FindAll(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, users)
	})

	// ── ExistsByEmail ─────────────────────────────────────────────────────────
	t.Run("exists by email", func(t *testing.T) {
		email := fmt.Sprintf("bob+%d@example.com", time.Now().UnixNano())
		user := &entity.User{Name: "Bob", Email: email, Password: "h"}
		require.NoError(t, repo.Create(ctx, user))

		exists, err := repo.ExistsByEmail(ctx, email)
		require.NoError(t, err)
		assert.True(t, exists)

		notExists, err := repo.ExistsByEmail(ctx, "nobody@example.com")
		require.NoError(t, err)
		assert.False(t, notExists)
	})

	// ── Update ────────────────────────────────────────────────────────────────
	t.Run("update", func(t *testing.T) {
		user := &entity.User{
			Name:     "Carol",
			Email:    fmt.Sprintf("carol+%d@example.com", time.Now().UnixNano()),
			Password: "h",
		}
		require.NoError(t, repo.Create(ctx, user))

		user.Name = "Carol Updated"
		require.NoError(t, repo.Update(ctx, user))

		found, err := repo.FindByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, "Carol Updated", found.Name)
	})

	// ── Delete ────────────────────────────────────────────────────────────────
	t.Run("delete", func(t *testing.T) {
		user := &entity.User{
			Name:     "Dave",
			Email:    fmt.Sprintf("dave+%d@example.com", time.Now().UnixNano()),
			Password: "h",
		}
		require.NoError(t, repo.Create(ctx, user))
		require.NoError(t, repo.Delete(ctx, user.ID))

		_, err := repo.FindByID(ctx, user.ID)
		assert.Error(t, err) // soft-deleted; expect not found
	})
}
