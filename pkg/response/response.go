package response

import "github.com/gofiber/fiber/v2"

// Response is the standard API response envelope.
type Response struct {
	Status  bool        `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ErrorCode is a machine-readable error identifier for frontend/API consumers.
type ErrorCode string

const (
	ErrValidation   ErrorCode = "ERR_VALIDATION"
	ErrNotFound     ErrorCode = "ERR_NOT_FOUND"
	ErrConflict     ErrorCode = "ERR_CONFLICT"
	ErrUnauthorized ErrorCode = "ERR_UNAUTHORIZED"
	ErrForbidden    ErrorCode = "ERR_FORBIDDEN"
	ErrInternal     ErrorCode = "ERR_INTERNAL"
	ErrBadRequest   ErrorCode = "ERR_BAD_REQUEST"
)

// ErrorResponse is returned on errors — deliberately excludes internal details.
type ErrorResponse struct {
	Status  bool      `json:"status"`
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

func Success(c *fiber.Ctx, message string, data interface{}) error {
	return c.Status(fiber.StatusOK).JSON(Response{
		Status:  true,
		Message: message,
		Data:    data,
	})
}

func Created(c *fiber.Ctx, message string, data interface{}) error {
	return c.Status(fiber.StatusCreated).JSON(Response{
		Status:  true,
		Message: message,
		Data:    data,
	})
}

// Error sends an error response with a structured error code.
func Error(c *fiber.Ctx, httpStatus int, errCode ErrorCode, message string) error {
	return c.Status(httpStatus).JSON(ErrorResponse{
		Status:  false,
		Code:    errCode,
		Message: message,
	})
}

func BadRequest(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusBadRequest, ErrBadRequest, message)
}

func ValidationError(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusUnprocessableEntity, ErrValidation, message)
}

func NotFound(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusNotFound, ErrNotFound, message)
}

func Conflict(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusConflict, ErrConflict, message)
}

func InternalError(c *fiber.Ctx) error {
	return Error(c, fiber.StatusInternalServerError, ErrInternal, "internal server error")
}

func Unauthorized(c *fiber.Ctx) error {
	return Error(c, fiber.StatusUnauthorized, ErrUnauthorized, "unauthorized")
}

func Forbidden(c *fiber.Ctx) error {
	return Error(c, fiber.StatusForbidden, ErrForbidden, "forbidden")
}

// PageMeta holds pagination metadata included in list responses.
type PageMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// PaginatedResponse is the standard envelope for paginated list endpoints.
type PaginatedResponse struct {
	Status     bool        `json:"status"`
	Message    string      `json:"message"`
	Data       interface{} `json:"data,omitempty"`
	Pagination PageMeta    `json:"pagination"`
}

// Paginated sends a 200 OK response with data and pagination metadata.
// total is the total number of records (across all pages).
func Paginated(c *fiber.Ctx, message string, data interface{}, page, perPage, total int) error {
	totalPages := 0
	if perPage > 0 {
		totalPages = (total + perPage - 1) / perPage
	}
	return c.Status(fiber.StatusOK).JSON(PaginatedResponse{
		Status:  true,
		Message: message,
		Data:    data,
		Pagination: PageMeta{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}
