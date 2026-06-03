package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abdullahPrasetio/wapgo/internal/delivery/http/handler"
	"github.com/abdullahPrasetio/wapgo/internal/domain/entity"
	"github.com/abdullahPrasetio/wapgo/internal/usecase"
	"github.com/abdullahPrasetio/wapgo/pkg/pagination"
	"github.com/abdullahPrasetio/wapgo/pkg/validator"
)

// ── mock usecase ─────────────────────────────────────────────────────────────

type mockUserUC struct {
	user    *entity.User
	users   []*entity.User
	err     error
}

func (m *mockUserUC) GetUser(_ context.Context, _ string) (*entity.User, error) {
	return m.user, m.err
}
func (m *mockUserUC) ListUsers(_ context.Context) ([]*entity.User, error) {
	return m.users, m.err
}
func (m *mockUserUC) CreateUser(_ context.Context, _ *usecase.CreateUserRequest) (*entity.User, error) {
	return m.user, m.err
}
func (m *mockUserUC) UpdateUser(_ context.Context, _ string, _ *usecase.UpdateUserRequest) (*entity.User, error) {
	return m.user, m.err
}
func (m *mockUserUC) DeleteUser(_ context.Context, _ string) error {
	return m.err
}
func (m *mockUserUC) ListUsersPaged(_ context.Context, _ *pagination.Request) ([]*entity.User, int, error) {
	return m.users, len(m.users), m.err
}

// ── helpers ───────────────────────────────────────────────────────────────────

func testApp(h *handler.UserHandler) *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/users/:id", h.GetUser)
	app.Get("/users", h.ListUsers)
	app.Post("/users", h.CreateUser)
	app.Put("/users/:id", h.UpdateUser)
	app.Delete("/users/:id", h.DeleteUser)
	return app
}

func fakeUser() *entity.User {
	return &entity.User{Name: "Alice", Email: "alice@example.com"}
}

func jsonBody(t *testing.T, v interface{}) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewBuffer(b)
}

// ── GetUser ───────────────────────────────────────────────────────────────────

func TestGetUser_200(t *testing.T) {
	h := handler.NewUserHandler(&mockUserUC{user: fakeUser()}, validator.New())
	req := httptest.NewRequest("GET", "/users/some-id", nil)
	resp, err := testApp(h).Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestGetUser_NotFound(t *testing.T) {
	h := handler.NewUserHandler(&mockUserUC{err: usecase.ErrNotFound}, validator.New())
	req := httptest.NewRequest("GET", "/users/some-id", nil)
	resp, _ := testApp(h).Test(req)
	assert.Equal(t, 404, resp.StatusCode)
}

func TestGetUser_InvalidUUID(t *testing.T) {
	h := handler.NewUserHandler(&mockUserUC{err: usecase.ErrInvalidUUID}, validator.New())
	req := httptest.NewRequest("GET", "/users/bad", nil)
	resp, _ := testApp(h).Test(req)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestGetUser_InternalError(t *testing.T) {
	h := handler.NewUserHandler(&mockUserUC{err: errors.New("db down")}, validator.New())
	req := httptest.NewRequest("GET", "/users/some-id", nil)
	resp, _ := testApp(h).Test(req)
	assert.Equal(t, 500, resp.StatusCode)
}

// ── ListUsers ─────────────────────────────────────────────────────────────────

func TestListUsers_200(t *testing.T) {
	h := handler.NewUserHandler(&mockUserUC{users: []*entity.User{fakeUser()}}, validator.New())
	req := httptest.NewRequest("GET", "/users", nil)
	resp, _ := testApp(h).Test(req)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestListUsers_InternalError(t *testing.T) {
	h := handler.NewUserHandler(&mockUserUC{err: errors.New("err")}, validator.New())
	req := httptest.NewRequest("GET", "/users", nil)
	resp, _ := testApp(h).Test(req)
	assert.Equal(t, 500, resp.StatusCode)
}

func TestListUsers_PaginationMeta(t *testing.T) {
	users := []*entity.User{fakeUser(), fakeUser()}
	h := handler.NewUserHandler(&mockUserUC{users: users}, validator.New())
	req := httptest.NewRequest("GET", "/users?page=1&size=10", nil)
	resp, _ := testApp(h).Test(req)
	assert.Equal(t, 200, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	pag, ok := body["pagination"].(map[string]any)
	require.True(t, ok, "pagination key must be present")
	assert.Equal(t, float64(1), pag["page"])
	assert.Equal(t, float64(10), pag["per_page"])
	assert.Equal(t, float64(2), pag["total"])
}

// ── CreateUser ────────────────────────────────────────────────────────────────

func TestCreateUser_201(t *testing.T) {
	h := handler.NewUserHandler(&mockUserUC{user: fakeUser()}, validator.New())
	body := jsonBody(t, map[string]string{
		"name": "Alice", "email": "alice@example.com", "password": "secret123",
	})
	req := httptest.NewRequest("POST", "/users", body)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := testApp(h).Test(req)
	assert.Equal(t, 201, resp.StatusCode)
}

func TestCreateUser_InvalidBody(t *testing.T) {
	h := handler.NewUserHandler(&mockUserUC{}, validator.New())
	req := httptest.NewRequest("POST", "/users", bytes.NewBufferString("not json{{{"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := testApp(h).Test(req)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestCreateUser_ValidationFail(t *testing.T) {
	h := handler.NewUserHandler(&mockUserUC{}, validator.New())
	body := jsonBody(t, map[string]string{"name": "", "email": "bad"})
	req := httptest.NewRequest("POST", "/users", body)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := testApp(h).Test(req)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestCreateUser_EmailConflict(t *testing.T) {
	h := handler.NewUserHandler(&mockUserUC{err: usecase.ErrEmailConflict}, validator.New())
	body := jsonBody(t, map[string]string{
		"name": "Alice", "email": "alice@example.com", "password": "password1",
	})
	req := httptest.NewRequest("POST", "/users", body)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := testApp(h).Test(req)
	assert.Equal(t, 409, resp.StatusCode)
}

// ── UpdateUser ────────────────────────────────────────────────────────────────

func TestUpdateUser_200(t *testing.T) {
	h := handler.NewUserHandler(&mockUserUC{user: fakeUser()}, validator.New())
	body := jsonBody(t, map[string]string{"name": "Alice Updated"})
	req := httptest.NewRequest("PUT", "/users/some-id", body)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := testApp(h).Test(req)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestUpdateUser_ValidationFail(t *testing.T) {
	h := handler.NewUserHandler(&mockUserUC{}, validator.New())
	body := jsonBody(t, map[string]string{"email": "bad-email"})
	req := httptest.NewRequest("PUT", "/users/some-id", body)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := testApp(h).Test(req)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestUpdateUser_NotFound(t *testing.T) {
	h := handler.NewUserHandler(&mockUserUC{err: usecase.ErrNotFound}, validator.New())
	body := jsonBody(t, map[string]string{})
	req := httptest.NewRequest("PUT", "/users/some-id", body)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := testApp(h).Test(req)
	assert.Equal(t, 404, resp.StatusCode)
}

// ── DeleteUser ────────────────────────────────────────────────────────────────

func TestDeleteUser_200(t *testing.T) {
	h := handler.NewUserHandler(&mockUserUC{}, validator.New())
	req := httptest.NewRequest("DELETE", "/users/some-id", nil)
	resp, _ := testApp(h).Test(req)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestDeleteUser_NotFound(t *testing.T) {
	h := handler.NewUserHandler(&mockUserUC{err: usecase.ErrNotFound}, validator.New())
	req := httptest.NewRequest("DELETE", "/users/some-id", nil)
	resp, _ := testApp(h).Test(req)
	assert.Equal(t, 404, resp.StatusCode)
}

func TestDeleteUser_InvalidUUID(t *testing.T) {
	h := handler.NewUserHandler(&mockUserUC{err: usecase.ErrInvalidUUID}, validator.New())
	req := httptest.NewRequest("DELETE", "/users/bad-id", nil)
	resp, _ := testApp(h).Test(req)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestDeleteUser_InternalError(t *testing.T) {
	h := handler.NewUserHandler(&mockUserUC{err: errors.New("db down")}, validator.New())
	req := httptest.NewRequest("DELETE", "/users/some-id", nil)
	resp, _ := testApp(h).Test(req)
	assert.Equal(t, 500, resp.StatusCode)
}

func TestUpdateUser_InvalidUUID(t *testing.T) {
	h := handler.NewUserHandler(&mockUserUC{err: usecase.ErrInvalidUUID}, validator.New())
	body := jsonBody(t, map[string]string{"name": "NewName"})
	req := httptest.NewRequest("PUT", "/users/bad-id", body)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := testApp(h).Test(req)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestUpdateUser_InternalError(t *testing.T) {
	h := handler.NewUserHandler(&mockUserUC{err: errors.New("db down")}, validator.New())
	body := jsonBody(t, map[string]string{"name": "NewName"})
	req := httptest.NewRequest("PUT", "/users/some-id", body)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := testApp(h).Test(req)
	assert.Equal(t, 500, resp.StatusCode)
}

func TestUpdateUser_BadBody(t *testing.T) {
	h := handler.NewUserHandler(&mockUserUC{user: fakeUser()}, validator.New())
	req := httptest.NewRequest("PUT", "/users/some-id", bytes.NewBufferString("{bad"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := testApp(h).Test(req)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestCreateUser_InternalError(t *testing.T) {
	h := handler.NewUserHandler(&mockUserUC{err: errors.New("db down")}, validator.New())
	body := jsonBody(t, map[string]string{
		"name": "Alice", "email": "alice@example.com", "password": "password1",
	})
	req := httptest.NewRequest("POST", "/users", body)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := testApp(h).Test(req)
	assert.Equal(t, 500, resp.StatusCode)
}
