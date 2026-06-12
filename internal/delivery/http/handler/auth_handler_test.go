package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/abdullahPrasetio/wapgo/internal/delivery/http/handler"
	"github.com/abdullahPrasetio/wapgo/internal/usecase"
	"github.com/abdullahPrasetio/wapgo/pkg/validator"
)

// --- mock AuthUseCase ---

type mockAuthUC struct {
	loginFn          func(ctx context.Context, req *usecase.LoginRequest) (*usecase.LoginResponse, error)
	refreshFn        func(ctx context.Context, req *usecase.RefreshRequest) (*usecase.LoginResponse, error)
	logoutFn         func(ctx context.Context, access, refresh string) error
	forgotPasswordFn func(ctx context.Context, req *usecase.ForgotPasswordRequest) (string, error)
	resetPasswordFn  func(ctx context.Context, req *usecase.ResetPasswordRequest) error
}

func (m *mockAuthUC) Login(ctx context.Context, req *usecase.LoginRequest) (*usecase.LoginResponse, error) {
	return m.loginFn(ctx, req)
}
func (m *mockAuthUC) Refresh(ctx context.Context, req *usecase.RefreshRequest) (*usecase.LoginResponse, error) {
	return m.refreshFn(ctx, req)
}
func (m *mockAuthUC) Logout(ctx context.Context, access, refresh string) error {
	return m.logoutFn(ctx, access, refresh)
}
func (m *mockAuthUC) ForgotPassword(ctx context.Context, req *usecase.ForgotPasswordRequest) (string, error) {
	return m.forgotPasswordFn(ctx, req)
}
func (m *mockAuthUC) ResetPassword(ctx context.Context, req *usecase.ResetPasswordRequest) error {
	return m.resetPasswordFn(ctx, req)
}

// --- helpers ---

func newAuthTestApp(uc usecase.AuthUseCase) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{"status": false})
		},
	})
	val := validator.New()
	h := handler.NewAuthHandler(uc, val, "development")
	app.Post("/auth/login", h.Login)
	app.Post("/auth/refresh", h.Refresh)
	app.Post("/auth/logout", h.Logout)
	app.Post("/auth/forgot-password", h.ForgotPassword)
	app.Post("/auth/reset-password", h.ResetPassword)
	return app
}

func authJSONBody(t *testing.T, v interface{}) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.NewBuffer(b)
}

// --- Login tests ---

func TestAuthHandler_Login_Success(t *testing.T) {
	uc := &mockAuthUC{
		loginFn: func(_ context.Context, _ *usecase.LoginRequest) (*usecase.LoginResponse, error) {
			return &usecase.LoginResponse{AccessToken: "acc", RefreshToken: "ref", ExpiresIn: 900}, nil
		},
	}
	req := httptest.NewRequest("POST", "/auth/login", authJSONBody(t, map[string]string{
		"email": "a@b.com", "password": "password123",
	}))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := newAuthTestApp(uc).Test(req)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	uc := &mockAuthUC{
		loginFn: func(_ context.Context, _ *usecase.LoginRequest) (*usecase.LoginResponse, error) {
			return nil, usecase.ErrInvalidCredentials
		},
	}
	req := httptest.NewRequest("POST", "/auth/login", authJSONBody(t, map[string]string{
		"email": "a@b.com", "password": "wrong",
	}))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := newAuthTestApp(uc).Test(req)
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestAuthHandler_Login_BadBody(t *testing.T) {
	req := httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := newAuthTestApp(&mockAuthUC{}).Test(req)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestAuthHandler_Login_ValidationError(t *testing.T) {
	req := httptest.NewRequest("POST", "/auth/login", authJSONBody(t, map[string]string{"email": "not-an-email"}))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := newAuthTestApp(&mockAuthUC{}).Test(req)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// --- Refresh tests ---

func TestAuthHandler_Refresh_Success(t *testing.T) {
	uc := &mockAuthUC{
		refreshFn: func(_ context.Context, _ *usecase.RefreshRequest) (*usecase.LoginResponse, error) {
			return &usecase.LoginResponse{AccessToken: "new_acc", RefreshToken: "new_ref"}, nil
		},
	}
	req := httptest.NewRequest("POST", "/auth/refresh", authJSONBody(t, map[string]string{"refresh_token": "tok"}))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := newAuthTestApp(uc).Test(req)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestAuthHandler_Refresh_InvalidToken(t *testing.T) {
	uc := &mockAuthUC{
		refreshFn: func(_ context.Context, _ *usecase.RefreshRequest) (*usecase.LoginResponse, error) {
			return nil, usecase.ErrInvalidToken
		},
	}
	req := httptest.NewRequest("POST", "/auth/refresh", authJSONBody(t, map[string]string{"refresh_token": "bad"}))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := newAuthTestApp(uc).Test(req)
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

// --- Logout tests ---

func TestAuthHandler_Logout_Success(t *testing.T) {
	uc := &mockAuthUC{logoutFn: func(_ context.Context, _, _ string) error { return nil }}
	req := httptest.NewRequest("POST", "/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer sometoken")
	resp, _ := newAuthTestApp(uc).Test(req)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestAuthHandler_Logout_MissingBearer(t *testing.T) {
	req := httptest.NewRequest("POST", "/auth/logout", nil)
	resp, _ := newAuthTestApp(&mockAuthUC{}).Test(req)
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestAuthHandler_Logout_InvalidToken(t *testing.T) {
	uc := &mockAuthUC{logoutFn: func(_ context.Context, _, _ string) error { return usecase.ErrInvalidToken }}
	req := httptest.NewRequest("POST", "/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer badtoken")
	resp, _ := newAuthTestApp(uc).Test(req)
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

// --- ForgotPassword tests ---

func TestAuthHandler_ForgotPassword_Success(t *testing.T) {
	uc := &mockAuthUC{
		forgotPasswordFn: func(_ context.Context, _ *usecase.ForgotPasswordRequest) (string, error) {
			return "reset-token-123", nil
		},
	}
	req := httptest.NewRequest("POST", "/auth/forgot-password", authJSONBody(t, map[string]string{"email": "user@example.com"}))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := newAuthTestApp(uc).Test(req)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestAuthHandler_ForgotPassword_ValidationError(t *testing.T) {
	req := httptest.NewRequest("POST", "/auth/forgot-password", authJSONBody(t, map[string]string{"email": "bad"}))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := newAuthTestApp(&mockAuthUC{}).Test(req)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestAuthHandler_ForgotPassword_InternalError(t *testing.T) {
	uc := &mockAuthUC{
		forgotPasswordFn: func(_ context.Context, _ *usecase.ForgotPasswordRequest) (string, error) {
			return "", errors.New("db down")
		},
	}
	req := httptest.NewRequest("POST", "/auth/forgot-password", authJSONBody(t, map[string]string{"email": "user@example.com"}))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := newAuthTestApp(uc).Test(req)
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
}

// --- ResetPassword tests ---

func TestAuthHandler_ResetPassword_Success(t *testing.T) {
	uc := &mockAuthUC{
		resetPasswordFn: func(_ context.Context, _ *usecase.ResetPasswordRequest) error { return nil },
	}
	req := httptest.NewRequest("POST", "/auth/reset-password", authJSONBody(t, map[string]string{
		"token": "tok123", "password": "newpassword1",
	}))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := newAuthTestApp(uc).Test(req)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestAuthHandler_ResetPassword_InvalidToken(t *testing.T) {
	uc := &mockAuthUC{
		resetPasswordFn: func(_ context.Context, _ *usecase.ResetPasswordRequest) error {
			return usecase.ErrInvalidToken
		},
	}
	req := httptest.NewRequest("POST", "/auth/reset-password", authJSONBody(t, map[string]string{
		"token": "expired", "password": "newpassword1",
	}))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := newAuthTestApp(uc).Test(req)
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}
