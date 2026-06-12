package auth

import "github.com/golang-jwt/jwt/v5"

// Claims is the JWT payload propagated through every authenticated request.
// Subject holds the user-ID; Roles carries optional RBAC role strings.
// TokenType distinguishes "access" from "refresh" tokens — middleware rejects any token
// that is not an access token, preventing refresh tokens from being used as Bearer auth.
type Claims struct {
	jwt.RegisteredClaims
	Roles     []string `json:"roles,omitempty"`
	TokenType string   `json:"token_type,omitempty"` // "access" | "refresh"
}
