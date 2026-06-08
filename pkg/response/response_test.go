package response_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abdullahPrasetio/wapgo/pkg/response"
)

func newApp(handler fiber.Handler) *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/test", handler)
	return app
}

func do(t *testing.T, app *fiber.App) (int, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()
	return resp.StatusCode, mustParseBody(t, resp)
}

func mustParseBody(t *testing.T, resp *http.Response) map[string]interface{} {
	t.Helper()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &m))
	return m
}

func TestSuccess(t *testing.T) {
	app := newApp(func(c *fiber.Ctx) error {
		return response.Success(c, "ok", fiber.Map{"key": "val"})
	})
	code, body := do(t, app)
	assert.Equal(t, fiber.StatusOK, code)
	assert.Equal(t, true, body["status"])
	assert.Equal(t, "ok", body["message"])
	assert.NotNil(t, body["data"])
}

func TestSuccess_NilData(t *testing.T) {
	app := newApp(func(c *fiber.Ctx) error {
		return response.Success(c, "deleted", nil)
	})
	code, body := do(t, app)
	assert.Equal(t, fiber.StatusOK, code)
	assert.Equal(t, true, body["status"])
	assert.Nil(t, body["data"])
}

func TestCreated(t *testing.T) {
	app := newApp(func(c *fiber.Ctx) error {
		return response.Created(c, "user created", fiber.Map{"id": "abc"})
	})
	code, body := do(t, app)
	assert.Equal(t, fiber.StatusCreated, code)
	assert.Equal(t, true, body["status"])
	assert.Equal(t, "user created", body["message"])
	assert.NotNil(t, body["data"])
}

func TestError(t *testing.T) {
	app := newApp(func(c *fiber.Ctx) error {
		return response.Error(c, fiber.StatusConflict, response.ErrConflict, "email conflict")
	})
	code, body := do(t, app)
	assert.Equal(t, fiber.StatusConflict, code)
	assert.Equal(t, false, body["status"])
	assert.Equal(t, "email conflict", body["message"])
	assert.Equal(t, string(response.ErrConflict), body["code"])
	assert.Nil(t, body["data"])
}

func TestBadRequest(t *testing.T) {
	app := newApp(func(c *fiber.Ctx) error {
		return response.BadRequest(c, "invalid body")
	})
	code, body := do(t, app)
	assert.Equal(t, fiber.StatusBadRequest, code)
	assert.Equal(t, false, body["status"])
	assert.Equal(t, "invalid body", body["message"])
}

func TestNotFound(t *testing.T) {
	app := newApp(func(c *fiber.Ctx) error {
		return response.NotFound(c, "user not found")
	})
	code, body := do(t, app)
	assert.Equal(t, fiber.StatusNotFound, code)
	assert.Equal(t, false, body["status"])
	assert.Equal(t, "user not found", body["message"])
}

func TestInternalError(t *testing.T) {
	app := newApp(func(c *fiber.Ctx) error {
		return response.InternalError(c)
	})
	code, body := do(t, app)
	assert.Equal(t, fiber.StatusInternalServerError, code)
	assert.Equal(t, false, body["status"])
	// Must never expose internal details — message is always generic
	assert.Equal(t, "internal server error", body["message"])
}

func TestUnauthorized(t *testing.T) {
	app := newApp(func(c *fiber.Ctx) error {
		return response.Unauthorized(c)
	})
	code, body := do(t, app)
	assert.Equal(t, fiber.StatusUnauthorized, code)
	assert.Equal(t, false, body["status"])
	assert.Equal(t, "unauthorized", body["message"])
}

func TestPaginated(t *testing.T) {
	app := newApp(func(c *fiber.Ctx) error {
		return response.Paginated(c, "ok", []string{"a", "b"}, 2, 10, 25)
	})
	code, body := do(t, app)
	assert.Equal(t, fiber.StatusOK, code)
	assert.Equal(t, true, body["status"])
	assert.Equal(t, "ok", body["message"])
	assert.NotNil(t, body["data"])

	pg, ok := body["pagination"].(map[string]interface{})
	require.True(t, ok, "pagination field should be an object")
	assert.Equal(t, float64(2), pg["page"])
	assert.Equal(t, float64(10), pg["per_page"])
	assert.Equal(t, float64(25), pg["total"])
	assert.Equal(t, float64(3), pg["total_pages"]) // ceil(25/10) = 3
}

func TestPaginated_FirstPage(t *testing.T) {
	app := newApp(func(c *fiber.Ctx) error {
		return response.Paginated(c, "ok", []string{"x"}, 1, 10, 5)
	})
	code, body := do(t, app)
	assert.Equal(t, fiber.StatusOK, code)

	pg := body["pagination"].(map[string]interface{})
	assert.Equal(t, float64(1), pg["page"])
	assert.Equal(t, float64(1), pg["total_pages"]) // ceil(5/10) = 1
}

func TestValidationError(t *testing.T) {
	app := newApp(func(c *fiber.Ctx) error {
		return response.ValidationError(c, "name is required")
	})
	code, body := do(t, app)
	assert.Equal(t, fiber.StatusUnprocessableEntity, code)
	assert.Equal(t, false, body["status"])
	assert.Equal(t, "name is required", body["message"])
	assert.Equal(t, string(response.ErrValidation), body["code"])
}

func TestConflict(t *testing.T) {
	app := newApp(func(c *fiber.Ctx) error {
		return response.Conflict(c, "email already exists")
	})
	code, body := do(t, app)
	assert.Equal(t, fiber.StatusConflict, code)
	assert.Equal(t, false, body["status"])
	assert.Equal(t, "email already exists", body["message"])
	assert.Equal(t, string(response.ErrConflict), body["code"])
}

func TestForbidden(t *testing.T) {
	app := newApp(func(c *fiber.Ctx) error {
		return response.Forbidden(c)
	})
	code, body := do(t, app)
	assert.Equal(t, fiber.StatusForbidden, code)
	assert.Equal(t, false, body["status"])
	assert.Equal(t, "forbidden", body["message"])
	assert.Equal(t, string(response.ErrForbidden), body["code"])
}

func TestPaginated_ZeroResults(t *testing.T) {
	app := newApp(func(c *fiber.Ctx) error {
		return response.Paginated(c, "empty", []string{}, 1, 10, 0)
	})
	code, body := do(t, app)
	assert.Equal(t, fiber.StatusOK, code)

	pg := body["pagination"].(map[string]interface{})
	assert.Equal(t, float64(0), pg["total"])
	assert.Equal(t, float64(0), pg["total_pages"])
}
