// Package pagination provides query helpers for paginated GORM queries
// and Fiber request parsing.
package pagination

import (
	"math"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

const (
	defaultPage    = 1
	defaultSize    = 20
	maxSize        = 100
	defaultSortCol = "created_at"
	defaultOrder   = "desc"
)

// Request holds parsed pagination parameters from an HTTP query string.
type Request struct {
	Page  int    `query:"page"`
	Size  int    `query:"size"`
	Sort  string `query:"sort"`
	Order string `query:"order"`
}

// Page returns the 1-based page number (minimum 1).
func (r *Request) PageNum() int {
	if r.Page < 1 {
		return defaultPage
	}
	return r.Page
}

// PageSize returns the bounded page size.
func (r *Request) PageSize() int {
	if r.Size < 1 {
		return defaultSize
	}
	if r.Size > maxSize {
		return maxSize
	}
	return r.Size
}

// SortColumn returns the sort column (defaults to created_at).
func (r *Request) SortColumn() string {
	if r.Sort == "" {
		return defaultSortCol
	}
	return r.Sort
}

// SortOrder returns "asc" or "desc" (defaults to desc).
func (r *Request) SortOrder() string {
	if r.Order == "asc" {
		return "asc"
	}
	return defaultOrder
}

// Offset returns the SQL OFFSET value.
func (r *Request) Offset() int {
	return (r.PageNum() - 1) * r.PageSize()
}

// Result wraps a paginated data slice with metadata.
type Result[T any] struct {
	Data       []T `json:"data"`
	Page       int `json:"page"`
	Size       int `json:"size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// NewResult constructs a Result from fetched data and total count.
func NewResult[T any](data []T, total int, req *Request) Result[T] {
	totalPages := int(math.Ceil(float64(total) / float64(req.PageSize())))
	if totalPages < 1 {
		totalPages = 0
	}
	return Result[T]{
		Data:       data,
		Page:       req.PageNum(),
		Size:       req.PageSize(),
		Total:      total,
		TotalPages: totalPages,
	}
}

// FromQuery parses pagination parameters from a Fiber request context.
func FromQuery(c *fiber.Ctx) *Request {
	req := &Request{}
	_ = c.QueryParser(req)
	return req
}

// Scope returns a GORM scope that applies LIMIT, OFFSET and ORDER BY.
// Use it as: db.Scopes(pagination.Scope(req)).Find(&records)
func Scope(req *Request) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.
			Order(req.SortColumn() + " " + req.SortOrder()).
			Limit(req.PageSize()).
			Offset(req.Offset())
	}
}
