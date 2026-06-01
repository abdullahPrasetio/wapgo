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

func parseBody(t *testing.T, resp *http.Response) map[string]interface{} {
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
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	body := parseBody(t, resp)
	assert.Equal(t, true, body["status"])
	assert.Equal(t, "ok", body["message"])
}

func TestCreated(t *testing.T) {
	app := newApp(func(c *fiber.Ctx) error {
		return response.Created(c, "created", nil)
	})
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)
}

func TestError(t *testing.T) {
	app := newApp(func(c *fiber.Ctx) error {
		return response.Error(c, 400, "bad input")
	})
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
	body := parseBody(t, resp)
	assert.Equal(t, false, body["status"])
}

func TestBadRequest(t *testing.T) {
	app := newApp(func(c *fiber.Ctx) error {
		return response.BadRequest(c, "invalid")
	})
	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestNotFound(t *testing.T) {
	app := newApp(func(c *fiber.Ctx) error {
		return response.NotFound(c, "gone")
	})
	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, 404, resp.StatusCode)
}

func TestInternalError(t *testing.T) {
	app := newApp(func(c *fiber.Ctx) error {
		return response.InternalError(c)
	})
	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, 500, resp.StatusCode)
}

func TestUnauthorized(t *testing.T) {
	app := newApp(func(c *fiber.Ctx) error {
		return response.Unauthorized(c)
	})
	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, 401, resp.StatusCode)
}
