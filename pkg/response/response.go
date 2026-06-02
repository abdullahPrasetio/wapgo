package response

import "github.com/gofiber/fiber/v2"

// Response is the standard API response envelope.
type Response struct {
	Status  bool        `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ErrorResponse is returned on errors — deliberately excludes internal details.
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
	// Never expose internal details
	return Error(c, fiber.StatusInternalServerError, "internal server error")
}

func Unauthorized(c *fiber.Ctx) error {
	return Error(c, fiber.StatusUnauthorized, "unauthorized")
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
