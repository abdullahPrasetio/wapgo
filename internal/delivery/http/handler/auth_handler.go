package handler

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/abdullahPrasetio/wapgo/internal/usecase"
	"github.com/abdullahPrasetio/wapgo/pkg/response"
	"github.com/abdullahPrasetio/wapgo/pkg/validator"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	uc  usecase.AuthUseCase
	val *validator.Validator
	env string // "development" | "staging" | "production"
}

// NewAuthHandler creates an AuthHandler.
// env is used to gate the reset_token field in ForgotPassword responses — the
// token is only exposed in non-production environments.
func NewAuthHandler(uc usecase.AuthUseCase, val *validator.Validator, env string) *AuthHandler {
	return &AuthHandler{uc: uc, val: val, env: env}
}

// Login godoc
// @Summary      Login
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      usecase.LoginRequest   true  "Credentials"
// @Success      200   {object}  response.Response{data=usecase.LoginResponse}
// @Failure      400   {object}  response.ErrorResponse
// @Failure      401   {object}  response.ErrorResponse
// @Failure      500   {object}  response.ErrorResponse
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req usecase.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	if err := h.val.Validate(&req); err != nil {
		return response.BadRequest(c, err.Error())
	}

	resp, err := h.uc.Login(c.UserContext(), &req)
	if err != nil {
		return h.mapError(c, err)
	}
	return response.Success(c, "login successful", resp)
}

// Refresh godoc
// @Summary      Refresh access token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      usecase.RefreshRequest  true  "Refresh token"
// @Success      200   {object}  response.Response{data=usecase.LoginResponse}
// @Failure      400   {object}  response.ErrorResponse
// @Failure      401   {object}  response.ErrorResponse
// @Router       /auth/refresh [post]
func (h *AuthHandler) Refresh(c *fiber.Ctx) error {
	var req usecase.RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	if err := h.val.Validate(&req); err != nil {
		return response.BadRequest(c, err.Error())
	}

	resp, err := h.uc.Refresh(c.UserContext(), &req)
	if err != nil {
		return h.mapError(c, err)
	}
	return response.Success(c, "token refreshed", resp)
}

// Logout godoc
// @Summary      Logout (revoke access token)
// @Tags         auth
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body  body      usecase.LogoutRequest  false  "Optional refresh token to revoke"
// @Success      200   {object}  response.Response
// @Failure      401   {object}  response.ErrorResponse
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	raw := c.Get("Authorization")
	if !strings.HasPrefix(raw, "Bearer ") {
		return response.Unauthorized(c)
	}
	accessToken := strings.TrimPrefix(raw, "Bearer ")

	var req usecase.LogoutRequest
	// Body is optional — ignore parse error.
	_ = c.BodyParser(&req)

	if err := h.uc.Logout(c.UserContext(), accessToken, req.RefreshToken); err != nil {
		return h.mapError(c, err)
	}
	return response.Success(c, "logged out", nil)
}

// ForgotPassword godoc
// @Summary      Request password reset token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      usecase.ForgotPasswordRequest  true  "Email"
// @Success      200   {object}  response.Response
// @Failure      400   {object}  response.ErrorResponse
// @Router       /auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *fiber.Ctx) error {
	var req usecase.ForgotPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	if err := h.val.Validate(&req); err != nil {
		return response.BadRequest(c, err.Error())
	}

	token, err := h.uc.ForgotPassword(c.UserContext(), &req)
	if err != nil {
		return response.InternalError(c)
	}

	data := fiber.Map{"message": "if the email is registered you will receive a reset token"}
	if token != "" && h.env != "production" {
		data["reset_token"] = token
	}
	return response.Success(c, "password reset requested", data)
}

// ResetPassword godoc
// @Summary      Reset password using token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      usecase.ResetPasswordRequest  true  "Reset token + new password"
// @Success      200   {object}  response.Response
// @Failure      400   {object}  response.ErrorResponse
// @Failure      401   {object}  response.ErrorResponse
// @Router       /auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *fiber.Ctx) error {
	var req usecase.ResetPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	if err := h.val.Validate(&req); err != nil {
		return response.BadRequest(c, err.Error())
	}

	if err := h.uc.ResetPassword(c.UserContext(), &req); err != nil {
		return h.mapError(c, err)
	}
	return response.Success(c, "password reset successful", nil)
}

// mapError translates auth domain errors to HTTP responses.
func (h *AuthHandler) mapError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, usecase.ErrInvalidCredentials):
		return response.Unauthorized(c)
	case errors.Is(err, usecase.ErrInvalidToken):
		return response.Unauthorized(c)
	default:
		return response.InternalError(c)
	}
}
