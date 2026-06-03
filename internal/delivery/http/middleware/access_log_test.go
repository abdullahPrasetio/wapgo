package middleware_test

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abdullahPrasetio/wapgo/internal/delivery/http/middleware"
	"github.com/abdullahPrasetio/wapgo/pkg/journal"
	applogger "github.com/abdullahPrasetio/wapgo/pkg/logger"
)

func TestAccessLog_WritesFullRecordWithThirdPartyAndTrace(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, applogger.SetupSinks(applogger.SinkConfig{Dir: dir}))

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(middleware.RequestID())
	app.Use(middleware.AccessLog(middleware.AccessLogOptions{BodyMaxBytes: 4096, CaptureBodies: true}))
	app.Post("/orders", func(c *fiber.Ctx) error {
		j := journal.FromContext(c.UserContext())
		j.AddThirdParty(journal.ThirdParty{Name: "billing", Method: "POST", URL: "https://billing/charge", Status: 200})
		j.AddTrace("risk.score", map[string]any{"score": 7})
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": "o1"})
	})

	req := httptest.NewRequest("POST", "/orders", strings.NewReader(`{"amount":100}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer top-secret")
	_, err := app.Test(req, -1)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, "api.log"))
	require.NoError(t, err)
	line := string(data)

	// Full request/response (bodies are JSON-escaped strings inside the record).
	assert.Contains(t, line, `"method":"POST"`)
	assert.Contains(t, line, "amount")        // request body captured
	assert.Contains(t, line, `"status":201`)  // response status
	assert.Contains(t, line, "o1")            // response body captured
	// Secrets redacted.
	assert.Contains(t, line, "[redacted]")
	assert.NotContains(t, line, "top-secret")
	// Embedded arrays.
	assert.Contains(t, line, `"name":"billing"`)
	assert.Contains(t, line, `"name":"risk.score"`)

	// Dual-write to their own files.
	tp, _ := os.ReadFile(filepath.Join(dir, "thirdparty.log"))
	assert.Contains(t, string(tp), `"name":"billing"`)
	tr, _ := os.ReadFile(filepath.Join(dir, "trace.log"))
	assert.Contains(t, string(tr), `"name":"risk.score"`)
}

func TestAccessLog_OmitsBodiesWhenDisabled(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, applogger.SetupSinks(applogger.SinkConfig{Dir: dir}))

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(middleware.RequestID())
	app.Use(middleware.AccessLog(middleware.AccessLogOptions{CaptureBodies: false}))
	app.Post("/x", func(c *fiber.Ctx) error { return c.SendString("hello-response") })

	req := httptest.NewRequest("POST", "/x", strings.NewReader("hello-request"))
	_, err := app.Test(req, -1)
	require.NoError(t, err)

	data, _ := os.ReadFile(filepath.Join(dir, "api.log"))
	line := string(data)
	assert.Contains(t, line, `"[omitted]"`)
	assert.NotContains(t, line, "hello-request")
	assert.NotContains(t, line, "hello-response")
}
