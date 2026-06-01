package observability

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestFiberHeaderCarrier_SetAndKeys(t *testing.T) {
	var keys []string

	app := fiber.New()
	app.Get("/hc", func(c *fiber.Ctx) error {
		carrier := &fiberHeaderCarrier{c: c}
		// Set writes a response header — must be called while ctx is live.
		carrier.Set("X-Out", "val")
		// Keys lists request headers.
		keys = carrier.Keys()
		return c.SendStatus(200)
	})

	req := httptest.NewRequest("GET", "/hc", nil)
	req.Header.Set("X-Foo", "bar")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Contains(t, keys, "X-Foo")
}

func TestNoopProvider_AllMethods(t *testing.T) {
	p := noopProvider{}

	// HTTPMiddleware must return a passthrough handler.
	app := fiber.New()
	app.Use(p.HTTPMiddleware())
	app.Get("/ping", func(c *fiber.Ctx) error { return c.SendStatus(200) })
	req := httptest.NewRequest("GET", "/ping", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// InstrumentGORM with nil db — noopProvider ignores it.
	assert.NoError(t, p.InstrumentGORM(nil))

	// WrapTransport returns the inner transport unchanged.
	assert.Nil(t, p.WrapTransport(nil))

	// Shutdown is a no-op.
	assert.NoError(t, p.Shutdown(context.Background()))
}
