// Example: full HTTP auth flow — login → JWT → protected route → role gate.
//
// This example shows the complete pattern for adding authentication to a
// wapgo service: a public /login endpoint, JWT middleware on a group,
// and RBAC with RequireRole.
//
// Run:
//
//	cd examples/auth && go run main.go
//
// Then in another terminal:
//
//	# Login and get token
//	TOKEN=$(curl -s -X POST http://localhost:8081/login \
//	  -H 'Content-Type: application/json' \
//	  -d '{"email":"admin@example.com","password":"secret"}' | jq -r '.token')
//
//	# Call protected endpoint
//	curl -H "Authorization: Bearer $TOKEN" http://localhost:8081/api/profile
//
//	# Call admin-only endpoint
//	curl -H "Authorization: Bearer $TOKEN" http://localhost:8081/api/admin/dashboard
package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"

	"github.com/abdullahPrasetio/wapgo/pkg/auth"
	"github.com/abdullahPrasetio/wapgo/pkg/response"
)

// user simulates a database row.
type user struct {
	ID           string
	Email        string
	PasswordHash string
	Roles        []string
}

func main() {
	jwtCfg := &auth.Config{
		Secret:   "super-secret-key-at-least-32-bytes!",
		Issuer:   "wapgo-auth-example",
		Audience: "wapgo-api",
		Expiry:   0, // default 24h
	}

	// Seed one admin and one regular user.
	adminHash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	userHash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)

	db := map[string]*user{
		"admin@example.com": {ID: "u-001", Email: "admin@example.com", PasswordHash: string(adminHash), Roles: []string{"admin", "user"}},
		"user@example.com":  {ID: "u-002", Email: "user@example.com", PasswordHash: string(userHash), Roles: []string{"user"}},
	}

	app := fiber.New(fiber.Config{DisableStartupMessage: true})

	// ── Public: login ─────────────────────────────────────────────────────────
	app.Post("/login", func(c *fiber.Ctx) error {
		var body struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := c.BodyParser(&body); err != nil {
			return response.BadRequest(c, "invalid request body")
		}

		u, ok := db[body.Email]
		if !ok || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(body.Password)) != nil {
			return response.Error(c, fiber.StatusUnauthorized, response.ErrUnauthorized, "invalid credentials")
		}

		token, err := auth.Sign(u.ID, u.Roles, jwtCfg)
		if err != nil {
			return response.InternalError(c)
		}
		return c.JSON(fiber.Map{"token": token})
	})

	// ── Protected group: JWT required ─────────────────────────────────────────
	api := app.Group("/api", auth.Middleware(jwtCfg))

	// Any authenticated user
	api.Get("/profile", func(c *fiber.Ctx) error {
		claims := auth.GetClaims(c)
		return response.Success(c, "profile", fiber.Map{
			"id":    claims.Subject,
			"roles": claims.Roles,
		})
	})

	// Admin-only endpoint
	api.Get("/admin/dashboard",
		auth.RequireRole("admin"),
		func(c *fiber.Ctx) error {
			return response.Success(c, "admin dashboard", fiber.Map{"stats": "ok"})
		},
	)

	log.Println("auth example listening on :8081")
	log.Fatal(app.Listen(":8081"))
}
