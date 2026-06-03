package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	_ "github.com/abdullahPrasetio/wapgo/internal/domain/entity" // swag type resolution
	"github.com/abdullahPrasetio/wapgo/internal/usecase"
	"github.com/abdullahPrasetio/wapgo/pkg/pagination"
	"github.com/abdullahPrasetio/wapgo/pkg/response"
	"github.com/abdullahPrasetio/wapgo/pkg/validator"
)

// UserHandler handles HTTP requests for the user domain.
type UserHandler struct {
	uc  usecase.UserUseCase
	val *validator.Validator
}

// NewUserHandler creates a UserHandler with the given usecase and validator.
func NewUserHandler(uc usecase.UserUseCase, val *validator.Validator) *UserHandler {
	return &UserHandler{uc: uc, val: val}
}

// GetUser godoc
// @Summary      Get user by ID
// @Tags         users
// @Produce      json
// @Param        id   path      string  true  "User UUID"
// @Success      200  {object}  response.Response{data=entity.User}
// @Failure      400  {object}  response.ErrorResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Router       /users/{id} [get]
func (h *UserHandler) GetUser(c *fiber.Ctx) error {
	id := c.Params("id")
	user, err := h.uc.GetUser(c.UserContext(), id)
	if err != nil {
		return h.mapError(c, err)
	}
	return response.Success(c, "user retrieved", user)
}

// ListUsers godoc
// @Summary      List users (paginated)
// @Tags         users
// @Produce      json
// @Param        page   query     int     false  "Page number (default 1)"
// @Param        size   query     int     false  "Page size (default 20, max 100)"
// @Param        sort   query     string  false  "Sort column (default created_at)"
// @Param        order  query     string  false  "Sort order: asc or desc (default desc)"
// @Success      200    {object}  response.PaginatedResponse{data=[]entity.User}
// @Failure      500    {object}  response.ErrorResponse
// @Router       /users [get]
func (h *UserHandler) ListUsers(c *fiber.Ctx) error {
	req := pagination.FromQuery(c)
	users, total, err := h.uc.ListUsersPaged(c.UserContext(), req)
	if err != nil {
		return response.InternalError(c)
	}
	return response.Paginated(c, "users retrieved", users, req.PageNum(), req.PageSize(), total)
}

// CreateUser godoc
// @Summary      Create a new user
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body      usecase.CreateUserRequest  true  "User payload"
// @Success      201   {object}  response.Response{data=entity.User}
// @Failure      400   {object}  response.ErrorResponse
// @Failure      409   {object}  response.ErrorResponse
// @Failure      500   {object}  response.ErrorResponse
// @Router       /users [post]
func (h *UserHandler) CreateUser(c *fiber.Ctx) error {
	var req usecase.CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	if err := h.val.Validate(&req); err != nil {
		return response.BadRequest(c, err.Error())
	}

	user, err := h.uc.CreateUser(c.UserContext(), &req)
	if err != nil {
		return h.mapError(c, err)
	}
	return response.Created(c, "user created", user)
}

// UpdateUser godoc
// @Summary      Update user by ID
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id    path      string                     true  "User UUID"
// @Param        body  body      usecase.UpdateUserRequest  true  "Update payload"
// @Success      200   {object}  response.Response{data=entity.User}
// @Failure      400   {object}  response.ErrorResponse
// @Failure      404   {object}  response.ErrorResponse
// @Failure      409   {object}  response.ErrorResponse
// @Failure      500   {object}  response.ErrorResponse
// @Router       /users/{id} [put]
func (h *UserHandler) UpdateUser(c *fiber.Ctx) error {
	id := c.Params("id")

	var req usecase.UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	if err := h.val.Validate(&req); err != nil {
		return response.BadRequest(c, err.Error())
	}

	user, err := h.uc.UpdateUser(c.UserContext(), id, &req)
	if err != nil {
		return h.mapError(c, err)
	}
	return response.Success(c, "user updated", user)
}

// DeleteUser godoc
// @Summary      Delete user by ID
// @Tags         users
// @Produce      json
// @Param        id   path      string  true  "User UUID"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.ErrorResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Router       /users/{id} [delete]
func (h *UserHandler) DeleteUser(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.uc.DeleteUser(c.UserContext(), id); err != nil {
		return h.mapError(c, err)
	}
	return response.Success(c, "user deleted", nil)
}

// mapError translates domain errors to appropriate HTTP responses.
// It deliberately avoids leaking internal error details.
func (h *UserHandler) mapError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, usecase.ErrNotFound):
		return response.NotFound(c, "user not found")
	case errors.Is(err, usecase.ErrEmailConflict):
		return response.Conflict(c, "email already in use")
	case errors.Is(err, usecase.ErrInvalidUUID):
		return response.BadRequest(c, "invalid id format")
	default:
		return response.InternalError(c)
	}
}
