package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/abdullahPrasetio/wapgo/internal/domain/entity"
	"github.com/abdullahPrasetio/wapgo/internal/usecase"
	"github.com/abdullahPrasetio/wapgo/pkg/pagination"
)

// ── mock repository ──────────────────────────────────────────────────────────

type mockUserRepo struct {
	users    map[uuid.UUID]*entity.User
	forceErr error
}

func newMockRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[uuid.UUID]*entity.User)}
}

func (m *mockUserRepo) FindByID(_ context.Context, id uuid.UUID) (*entity.User, error) {
	if m.forceErr != nil {
		return nil, m.forceErr
	}
	u, ok := m.users[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return u, nil
}

func (m *mockUserRepo) FindAll(_ context.Context) ([]*entity.User, error) {
	if m.forceErr != nil {
		return nil, m.forceErr
	}
	list := make([]*entity.User, 0, len(m.users))
	for _, u := range m.users {
		list = append(list, u)
	}
	return list, nil
}

func (m *mockUserRepo) Create(_ context.Context, user *entity.User) error {
	if m.forceErr != nil {
		return m.forceErr
	}
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepo) Update(_ context.Context, user *entity.User) error {
	if m.forceErr != nil {
		return m.forceErr
	}
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepo) Delete(_ context.Context, id uuid.UUID) error {
	if m.forceErr != nil {
		return m.forceErr
	}
	delete(m.users, id)
	return nil
}

func (m *mockUserRepo) ExistsByEmail(_ context.Context, email string) (bool, error) {
	if m.forceErr != nil {
		return false, m.forceErr
	}
	for _, u := range m.users {
		if u.Email == email {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockUserRepo) FindByEmail(_ context.Context, email string) (*entity.User, error) {
	if m.forceErr != nil {
		return nil, m.forceErr
	}
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockUserRepo) FindAllPaged(_ context.Context, _ *pagination.Request) ([]*entity.User, int, error) {
	if m.forceErr != nil {
		return nil, 0, m.forceErr
	}
	list := make([]*entity.User, 0, len(m.users))
	for _, u := range m.users {
		list = append(list, u)
	}
	return list, len(list), nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func seedUser(t *testing.T, uc usecase.UserUseCase) *entity.User {
	t.Helper()
	u, err := uc.CreateUser(context.Background(), &usecase.CreateUserRequest{
		Name:     "Alice",
		Email:    "alice@example.com",
		Password: "securePass1",
	})
	require.NoError(t, err)
	return u
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestCreateUser_Success(t *testing.T) {
	uc := usecase.NewUserUseCase(newMockRepo())
	u, err := uc.CreateUser(context.Background(), &usecase.CreateUserRequest{
		Name:     "Bob",
		Email:    "bob@example.com",
		Password: "password123",
	})
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, u.ID)
	assert.Equal(t, "Bob", u.Name)
	// Password field holds the bcrypt hash (not the plaintext); json:"-" hides it from API responses
	assert.NotEqual(t, "password123", u.Password, "plain-text password must not be stored")
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	uc := usecase.NewUserUseCase(newMockRepo())
	seedUser(t, uc)

	_, err := uc.CreateUser(context.Background(), &usecase.CreateUserRequest{
		Name:     "Alice2",
		Email:    "alice@example.com",
		Password: "anotherPass1",
	})
	assert.ErrorIs(t, err, usecase.ErrEmailConflict)
}

func TestCreateUser_RepoError(t *testing.T) {
	repo := newMockRepo()
	repo.forceErr = errors.New("db down")
	uc := usecase.NewUserUseCase(repo)

	_, err := uc.CreateUser(context.Background(), &usecase.CreateUserRequest{
		Name:     "X",
		Email:    "x@x.com",
		Password: "password123",
	})
	require.Error(t, err)
}

func TestGetUser_Success(t *testing.T) {
	uc := usecase.NewUserUseCase(newMockRepo())
	u := seedUser(t, uc)

	got, err := uc.GetUser(context.Background(), u.ID.String())
	require.NoError(t, err)
	assert.Equal(t, u.ID, got.ID)
}

func TestGetUser_NotFound(t *testing.T) {
	uc := usecase.NewUserUseCase(newMockRepo())
	_, err := uc.GetUser(context.Background(), uuid.New().String())
	assert.ErrorIs(t, err, usecase.ErrNotFound)
}

func TestGetUser_InvalidUUID(t *testing.T) {
	uc := usecase.NewUserUseCase(newMockRepo())
	_, err := uc.GetUser(context.Background(), "not-a-uuid")
	assert.ErrorIs(t, err, usecase.ErrInvalidUUID)
}

func TestGetUser_RepoError(t *testing.T) {
	repo := newMockRepo()
	uc := usecase.NewUserUseCase(repo)
	u := seedUser(t, uc)

	repo.forceErr = errors.New("db down")
	_, err := uc.GetUser(context.Background(), u.ID.String())
	require.Error(t, err)
	assert.NotErrorIs(t, err, usecase.ErrNotFound)
}

func TestListUsers(t *testing.T) {
	uc := usecase.NewUserUseCase(newMockRepo())
	seedUser(t, uc)

	list, err := uc.ListUsers(context.Background())
	require.NoError(t, err)
	assert.Len(t, list, 1)
}

func TestListUsers_RepoError(t *testing.T) {
	repo := newMockRepo()
	repo.forceErr = errors.New("db err")
	uc := usecase.NewUserUseCase(repo)

	_, err := uc.ListUsers(context.Background())
	require.Error(t, err)
}

func TestUpdateUser_Success(t *testing.T) {
	uc := usecase.NewUserUseCase(newMockRepo())
	u := seedUser(t, uc)

	updated, err := uc.UpdateUser(context.Background(), u.ID.String(), &usecase.UpdateUserRequest{
		Name: "Alice Updated",
	})
	require.NoError(t, err)
	assert.Equal(t, "Alice Updated", updated.Name)
}

func TestUpdateUser_EmailConflict(t *testing.T) {
	repo := newMockRepo()
	uc := usecase.NewUserUseCase(repo)
	u1 := seedUser(t, uc)

	u2, _ := uc.CreateUser(context.Background(), &usecase.CreateUserRequest{
		Name: "Bob", Email: "bob@example.com", Password: "password123",
	})
	_ = u1

	_, err := uc.UpdateUser(context.Background(), u2.ID.String(), &usecase.UpdateUserRequest{
		Email: "alice@example.com",
	})
	assert.ErrorIs(t, err, usecase.ErrEmailConflict)
}

func TestUpdateUser_NotFound(t *testing.T) {
	uc := usecase.NewUserUseCase(newMockRepo())
	_, err := uc.UpdateUser(context.Background(), uuid.New().String(), &usecase.UpdateUserRequest{Name: "X"})
	assert.ErrorIs(t, err, usecase.ErrNotFound)
}

func TestUpdateUser_InvalidUUID(t *testing.T) {
	uc := usecase.NewUserUseCase(newMockRepo())
	_, err := uc.UpdateUser(context.Background(), "bad-id", &usecase.UpdateUserRequest{})
	assert.ErrorIs(t, err, usecase.ErrInvalidUUID)
}

func TestDeleteUser_Success(t *testing.T) {
	uc := usecase.NewUserUseCase(newMockRepo())
	u := seedUser(t, uc)

	err := uc.DeleteUser(context.Background(), u.ID.String())
	require.NoError(t, err)

	_, err = uc.GetUser(context.Background(), u.ID.String())
	assert.ErrorIs(t, err, usecase.ErrNotFound)
}

func TestDeleteUser_NotFound(t *testing.T) {
	uc := usecase.NewUserUseCase(newMockRepo())
	err := uc.DeleteUser(context.Background(), uuid.New().String())
	assert.ErrorIs(t, err, usecase.ErrNotFound)
}

func TestDeleteUser_InvalidUUID(t *testing.T) {
	uc := usecase.NewUserUseCase(newMockRepo())
	err := uc.DeleteUser(context.Background(), "oops")
	assert.ErrorIs(t, err, usecase.ErrInvalidUUID)
}

func TestDeleteUser_RepoError(t *testing.T) {
	repo := newMockRepo()
	uc := usecase.NewUserUseCase(repo)
	u := seedUser(t, uc)

	repo.forceErr = errors.New("db down")
	err := uc.DeleteUser(context.Background(), u.ID.String())
	require.Error(t, err)
}
