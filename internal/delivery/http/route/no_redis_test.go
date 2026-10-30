package route

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/abdullahPrasetio/wapgo/internal/delivery/http/handler"
	"github.com/abdullahPrasetio/wapgo/internal/domain/entity"
	domainrepo "github.com/abdullahPrasetio/wapgo/internal/domain/repository"
	memoryrepo "github.com/abdullahPrasetio/wapgo/internal/repository/memory"
	"github.com/abdullahPrasetio/wapgo/internal/usecase"
	"github.com/abdullahPrasetio/wapgo/pkg/auth"
	"github.com/abdullahPrasetio/wapgo/pkg/pagination"
	"github.com/abdullahPrasetio/wapgo/pkg/validator"
)

// inMemoryUserRepo is a minimal UserRepository for the no-Redis auth flow test.
type inMemoryUserRepo struct {
	users []*entity.User
}

func (r *inMemoryUserRepo) FindByID(_ context.Context, id uuid.UUID) (*entity.User, error) {
	for _, u := range r.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *inMemoryUserRepo) FindByEmail(_ context.Context, email string) (*entity.User, error) {
	for _, u := range r.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *inMemoryUserRepo) FindAll(_ context.Context) ([]*entity.User, error) { return r.users, nil }

func (r *inMemoryUserRepo) FindAllPaged(_ context.Context, _ *pagination.Request) ([]*entity.User, int, error) {
	return r.users, len(r.users), nil
}

func (r *inMemoryUserRepo) Create(_ context.Context, u *entity.User) error {
	r.users = append(r.users, u)
	return nil
}

func (r *inMemoryUserRepo) Update(_ context.Context, u *entity.User) error {
	for i, existing := range r.users {
		if existing.ID == u.ID {
			r.users[i] = u
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

func (r *inMemoryUserRepo) Delete(_ context.Context, id uuid.UUID) error {
	for i, u := range r.users {
		if u.ID == id {
			r.users = append(r.users[:i], r.users[i+1:]...)
			return nil
		}
	}
	return nil
}

func (r *inMemoryUserRepo) ExistsByEmail(_ context.Context, email string) (bool, error) {
	_, err := r.FindByEmail(context.Background(), email)
	return err == nil, nil
}

// TestAuthFlow_WithoutRedis_LogoutRevokesTokens is the 0.4 smoke test: with no
// Redis client, an in-memory cache + blacklist must still support the full
// login → logout → refresh-401 flow.
func TestAuthFlow_WithoutRedis_LogoutRevokesTokens(t *testing.T) {
	const (
		email    = "alice@example.com"
		password = "supersecret123"
	)

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	require.NoError(t, err)

	repo := &inMemoryUserRepo{users: []*entity.User{{
		ID:       uuid.New(),
		Name:     "Alice",
		Email:    email,
		Password: string(hash),
	}}}
	cacher := memoryrepo.New("test")
	blacklist := auth.NewMemoryBlacklist()
	jwtCfg := &auth.Config{
		Secret:   "test-secret-at-least-32-bytes-long!",
		Issuer:   "wapgo-test",
		Audience: "wapgo-test-api",
		Expiry:   15 * time.Minute,
	}
	authUC := usecase.NewAuthUseCase(repo, cacher, jwtCfg, time.Hour, blacklist, 10)
	authHandler := handler.NewAuthHandler(authUC, validator.New(), "development")

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	v1 := app.Group("/api/v1")
	RegisterAuthRoutes(v1, authHandler, jwtCfg, blacklist)
	v1.Get("/protected", auth.Middleware(jwtCfg, blacklist), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	// 1. Login.
	loginBody, _ := json.Marshal(map[string]string{"email": email, "password": password})
	loginReq := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(loginReq, -1)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	var login struct {
		Data struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&login))
	require.NotEmpty(t, login.Data.AccessToken)
	require.NotEmpty(t, login.Data.RefreshToken)

	// 2. Access token is valid before logout.
	req := httptest.NewRequest("GET", "/api/v1/protected", nil)
	req.Header.Set("Authorization", "Bearer "+login.Data.AccessToken)
	resp, err = app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	// 3. Logout with access + refresh token.
	logoutBody, _ := json.Marshal(map[string]string{"refresh_token": login.Data.RefreshToken})
	req = httptest.NewRequest("POST", "/api/v1/auth/logout", bytes.NewReader(logoutBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+login.Data.AccessToken)
	resp, err = app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	// 4. The revoked refresh token can no longer mint a new pair.
	refreshBody, _ := json.Marshal(map[string]string{"refresh_token": login.Data.RefreshToken})
	refreshReq := httptest.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewReader(refreshBody))
	refreshReq.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(refreshReq, -1)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode, "refresh after logout must be 401 with in-memory revocation")

	// 5. The revoked access token is rejected too.
	req = httptest.NewRequest("GET", "/api/v1/protected", nil)
	req.Header.Set("Authorization", "Bearer "+login.Data.AccessToken)
	resp, err = app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

// TestAuthFlow_WithoutRedis_RefreshRotation keeps coverage on the in-memory
// session store: a rotated refresh token is single-use.
func TestAuthFlow_WithoutRedis_RefreshRotation(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("supersecret123"), 10)
	require.NoError(t, err)

	repo := &inMemoryUserRepo{users: []*entity.User{{
		ID:       uuid.New(),
		Name:     "Bob",
		Email:    "bob@example.com",
		Password: string(hash),
	}}}
	cacher := memoryrepo.New("test")
	blacklist := auth.NewMemoryBlacklist()
	jwtCfg := &auth.Config{
		Secret:   "test-secret-at-least-32-bytes-long!",
		Issuer:   "wapgo-test",
		Audience: "wapgo-test-api",
		Expiry:   time.Minute,
	}
	authUC := usecase.NewAuthUseCase(repo, cacher, jwtCfg, time.Hour, blacklist, 10)

	login, err := authUC.Login(context.Background(), &usecase.LoginRequest{
		Email:    "bob@example.com",
		Password: "supersecret123",
	})
	require.NoError(t, err)

	rotated, err := authUC.Refresh(context.Background(), &usecase.RefreshRequest{RefreshToken: login.RefreshToken})
	require.NoError(t, err)
	require.NotEmpty(t, rotated.RefreshToken)

	_, err = authUC.Refresh(context.Background(), &usecase.RefreshRequest{RefreshToken: login.RefreshToken})
	assert.ErrorIs(t, err, usecase.ErrInvalidToken, "old refresh token must not be reusable")
}

// ensure the in-memory cacher satisfies the interface used by the usecase.
var _ domainrepo.Cacher = (*memoryrepo.Cacher)(nil)
