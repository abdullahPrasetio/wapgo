// Example: JWT sign, verify, and RBAC middleware usage with wapgo/pkg/auth.
//
// Run:
//
//	cd examples/jwt && go run main.go
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/abdullahPrasetio/wapgo/pkg/auth"
)

func main() {
	cfg := &auth.Config{
		Secret:   "super-secret-key-at-least-32-bytes!",
		Issuer:   "wapgo",
		Audience: "wapgo-clients",
		Expiry:   15 * time.Minute,
	}

	// ── Sign ──────────────────────────────────────────────────────────────────
	token, err := auth.Sign("user-42", []string{"admin", "editor"}, cfg)
	if err != nil {
		log.Fatalf("sign: %v", err)
	}
	fmt.Println("Token:", token)

	// ── Verify ────────────────────────────────────────────────────────────────
	claims, err := auth.Verify(token, cfg)
	if err != nil {
		log.Fatalf("verify: %v", err)
	}
	fmt.Printf("Subject : %s\n", claims.Subject)
	fmt.Printf("Roles   : %v\n", claims.Roles)
	fmt.Printf("Expires : %s\n", claims.ExpiresAt.Time.Format(time.RFC3339)) //nolint:staticcheck

	// ── alg:none rejection demo ───────────────────────────────────────────────
	_, err = auth.Verify("eyJhbGciOiJub25lIn0.e30.", cfg)
	if err != nil {
		fmt.Println("alg:none rejected (expected):", err)
	}

	// ── Weak secret rejection demo ────────────────────────────────────────────
	weakCfg := &auth.Config{Secret: "short"}
	_, err = auth.Sign("user-1", nil, weakCfg)
	if err != nil {
		fmt.Println("Weak secret rejected (expected):", err)
	}
}
