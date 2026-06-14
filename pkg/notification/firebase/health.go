package firebase

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// HealthCheck returns a probe that verifies Firebase credentials are parseable
// and that the Google OAuth2 token endpoint is reachable via TCP.
// It does NOT perform a real token exchange — that would be too expensive for a
// liveness probe.
//
// Wire it into your health endpoint:
//
//	health.Register("firebase", firebase.HealthCheck(os.Getenv("FIREBASE_CREDENTIALS_JSON")))
func HealthCheck(credentialsJSON string) func(ctx context.Context) string {
	return func(ctx context.Context) string {
		if credentialsJSON == "" {
			return "error: FIREBASE_CREDENTIALS_JSON not set"
		}

		var sa serviceAccount
		if err := json.Unmarshal([]byte(credentialsJSON), &sa); err != nil {
			return "error: invalid credentials JSON: " + err.Error()
		}
		if sa.ProjectID == "" || sa.ClientEmail == "" {
			return "error: credentials missing project_id or client_email"
		}

		// TCP reachability check against Google's token endpoint.
		timeout := 5 * time.Second
		if dl, ok := ctx.Deadline(); ok {
			timeout = time.Until(dl)
		}
		conn, err := net.DialTimeout("tcp", "oauth2.googleapis.com:443", timeout)
		if err != nil {
			return fmt.Sprintf("error: cannot reach oauth2.googleapis.com: %s", err)
		}
		conn.Close() //nolint:errcheck
		return "ok"
	}
}
