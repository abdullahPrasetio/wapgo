package db_test

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/abdullahPrasetio/wapgo/internal/domain/entity"
	dbrepo "github.com/abdullahPrasetio/wapgo/internal/repository/db"
)

func newMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { sqlDB.Close() })

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	return db, mock
}

func userColumns() []string {
	return []string{"id", "name", "email", "password", "created_at", "updated_at", "deleted_at"}
}

func userRow(u *entity.User) *sqlmock.Rows {
	return sqlmock.NewRows(userColumns()).
		AddRow(u.ID, u.Name, u.Email, u.Password, u.CreatedAt, u.UpdatedAt, nil)
}

// ── FindByID ──────────────────────────────────────────────────────────────────

func TestFindByID_Found(t *testing.T) {
	db, mock := newMockDB(t)
	repo := dbrepo.NewUserRepository(db)

	id := uuid.New()
	expected := &entity.User{ID: id, Name: "Alice", Email: "alice@example.com", Password: "h"}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
		WithArgs(id, 1).
		WillReturnRows(userRow(expected))

	user, err := repo.FindByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, expected.Name, user.Name)
	assert.Equal(t, expected.Email, user.Email)
}

func TestFindByID_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	repo := dbrepo.NewUserRepository(db)

	id := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
		WithArgs(id, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := repo.FindByID(context.Background(), id)
	assert.Error(t, err)
}

func TestFindByID_DBError(t *testing.T) {
	db, mock := newMockDB(t)
	repo := dbrepo.NewUserRepository(db)

	id := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
		WithArgs(id, 1).
		WillReturnError(sql.ErrConnDone)

	_, err := repo.FindByID(context.Background(), id)
	assert.Error(t, err)
}

// ── FindAll ───────────────────────────────────────────────────────────────────

func TestFindAll_ReturnsRows(t *testing.T) {
	db, mock := newMockDB(t)
	repo := dbrepo.NewUserRepository(db)

	rows := sqlmock.NewRows(userColumns()).
		AddRow(uuid.New(), "Alice", "a@x.com", "h", time.Now(), time.Now(), nil).
		AddRow(uuid.New(), "Bob", "b@x.com", "h", time.Now(), time.Now(), nil)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
		WillReturnRows(rows)

	users, err := repo.FindAll(context.Background())
	require.NoError(t, err)
	assert.Len(t, users, 2)
}

func TestFindAll_Empty(t *testing.T) {
	db, mock := newMockDB(t)
	repo := dbrepo.NewUserRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
		WillReturnRows(sqlmock.NewRows(userColumns()))

	users, err := repo.FindAll(context.Background())
	require.NoError(t, err)
	assert.Empty(t, users)
}

// ── Create ────────────────────────────────────────────────────────────────────

func TestCreate_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := dbrepo.NewUserRepository(db)

	// BeforeCreate sets UUID, so GORM uses Exec (no RETURNING needed)
	user := &entity.User{Name: "Carol", Email: "carol@example.com", Password: "h"}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "users"`)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.Create(context.Background(), user)
	require.NoError(t, err)
}

func TestCreate_DBError(t *testing.T) {
	db, mock := newMockDB(t)
	repo := dbrepo.NewUserRepository(db)

	user := &entity.User{Name: "Dave", Email: "dave@example.com", Password: "h"}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "users"`)).
		WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	err := repo.Create(context.Background(), user)
	assert.Error(t, err)
}

// ── Update ────────────────────────────────────────────────────────────────────

func TestUpdate_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := dbrepo.NewUserRepository(db)

	user := &entity.User{ID: uuid.New(), Name: "Eve Updated", Email: "eve@example.com", Password: "h"}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "users"`)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.Update(context.Background(), user)
	require.NoError(t, err)
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestDelete_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := dbrepo.NewUserRepository(db)

	id := uuid.New()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "users"`)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.Delete(context.Background(), id)
	require.NoError(t, err)
}

// ── ExistsByEmail ─────────────────────────────────────────────────────────────

func TestExistsByEmail_Exists(t *testing.T) {
	db, mock := newMockDB(t)
	repo := dbrepo.NewUserRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "users"`)).
		WithArgs("alice@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	exists, err := repo.ExistsByEmail(context.Background(), "alice@example.com")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestExistsByEmail_NotExists(t *testing.T) {
	db, mock := newMockDB(t)
	repo := dbrepo.NewUserRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "users"`)).
		WithArgs("nobody@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	exists, err := repo.ExistsByEmail(context.Background(), "nobody@example.com")
	require.NoError(t, err)
	assert.False(t, exists)
}
