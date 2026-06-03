package pagination_test

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"

	"github.com/abdullahPrasetio/wapgo/pkg/pagination"
)

func TestRequest_defaults(t *testing.T) {
	req := &pagination.Request{}
	assert.Equal(t, 1, req.PageNum())
	assert.Equal(t, 20, req.PageSize())
	assert.Equal(t, "created_at", req.SortColumn())
	assert.Equal(t, "desc", req.SortOrder())
	assert.Equal(t, 0, req.Offset())
}

func TestRequest_clampsMaxSize(t *testing.T) {
	req := &pagination.Request{Size: 999}
	assert.Equal(t, 100, req.PageSize())
}

func TestRequest_offset(t *testing.T) {
	req := &pagination.Request{Page: 3, Size: 10}
	assert.Equal(t, 20, req.Offset())
}

func TestRequest_sortOrder(t *testing.T) {
	req := &pagination.Request{Order: "asc"}
	assert.Equal(t, "asc", req.SortOrder())

	req2 := &pagination.Request{Order: "invalid"}
	assert.Equal(t, "desc", req2.SortOrder())
}

func TestNewResult(t *testing.T) {
	req := &pagination.Request{Page: 2, Size: 10}
	data := []string{"a", "b"}
	result := pagination.NewResult(data, 25, req)

	assert.Equal(t, 2, result.Page)
	assert.Equal(t, 10, result.Size)
	assert.Equal(t, 25, result.Total)
	assert.Equal(t, 3, result.TotalPages)
	assert.Equal(t, data, result.Data)
}

func TestNewResult_zeroTotal(t *testing.T) {
	req := &pagination.Request{Page: 1, Size: 20}
	result := pagination.NewResult([]string{}, 0, req)
	assert.Equal(t, 0, result.TotalPages)
}

func TestFromQuery(t *testing.T) {
	app := fiber.New()
	var parsed *pagination.Request

	app.Get("/", func(c *fiber.Ctx) error {
		parsed = pagination.FromQuery(c)
		return c.SendStatus(200)
	})

	req := httptest.NewRequest("GET", "/?page=2&size=15&sort=name&order=asc", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, 2, parsed.PageNum())
	assert.Equal(t, 15, parsed.PageSize())
	assert.Equal(t, "name", parsed.SortColumn())
	assert.Equal(t, "asc", parsed.SortOrder())
}
