//go:build ignore

package response

import "github.com/gofiber/fiber/v2"

type Response struct {
	Status  bool        `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type ErrorResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
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

func Error(c *fiber.Ctx, code int, message string) error {
	return c.Status(code).JSON(ErrorResponse{
		Status:  false,
		Message: message,
	})
}

func BadRequest(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusBadRequest, message)
}

func NotFound(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusNotFound, message)
}

func InternalError(c *fiber.Ctx) error {
	return Error(c, fiber.StatusInternalServerError, "internal server error")
}

func Unauthorized(c *fiber.Ctx) error {
	return Error(c, fiber.StatusUnauthorized, "unauthorized")
}

type PageMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type PaginatedResponse struct {
	Status     bool        `json:"status"`
	Message    string      `json:"message"`
	Data       interface{} `json:"data,omitempty"`
	Pagination PageMeta    `json:"pagination"`
}

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
