// Package firebase provides an optional Firebase Cloud Messaging (FCM) push
// notification add-on for wapgo services.
//
// No Firebase Admin SDK dependency — auth is done directly via the FCM v1
// HTTP API using a Google service account JWT (RS256), matching the pattern
// used by the Google Identity Platform. The only extra packages used are
// golang-jwt/jwt/v5 (already in go.mod) and stdlib crypto/rsa + net/http.
//
// Usage:
//
//	pusher, err := firebase.NewFCMClient(os.Getenv("FIREBASE_CREDENTIALS_JSON"), logger)
//
//	err = pusher.Send(ctx, firebase.Message{
//	    Token: "device-registration-token",
//	    Title: "Order shipped",
//	    Body:  "Your order is on the way!",
//	    Data:  map[string]string{"order_id": "123"},
//	})
//
// Each Send records an OTel span ("notification.firebase.send") and adds a
// ThirdParty entry to the request journal. Access tokens are cached for ~1 hour.
package firebase

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"

	"github.com/abdullahPrasetio/wapgo/pkg/journal"
)

const (
	fcmScope    = "https://www.googleapis.com/auth/firebase.messaging"
	tokenURL    = "https://oauth2.googleapis.com/token"
	fcmEndpoint = "https://fcm.googleapis.com/v1/projects/%s/messages:send"
)

// Pusher is the interface for sending Firebase push notifications.
// Depend on this interface rather than FCMClient so it can be mocked in tests.
type Pusher interface {
	Send(ctx context.Context, msg Message) error
}

// Message is the FCM push notification to be sent.
// Set Token to send to a single device, or Topic to send to a subscribed topic.
type Message struct {
	Token    string            // device registration token (mutually exclusive with Topic)
	Topic    string            // FCM topic (mutually exclusive with Token)
	Title    string
	Body     string
	Data     map[string]string // extra key-value payload forwarded to the app
	ImageURL string            // optional large image URL
}

// serviceAccount mirrors the fields we need from a Firebase service account JSON file.
type serviceAccount struct {
	ProjectID   string `json:"project_id"`
	PrivateKey  string `json:"private_key"`
	ClientEmail string `json:"client_email"`
	TokenURI    string `json:"token_uri"`
}

// FCMClient sends push notifications via the FCM v1 HTTP API.
// Access tokens are cached until ~30 seconds before expiry, so concurrent
// callers share the same token without racing.
type FCMClient struct {
	sa          serviceAccount
	httpClient  *http.Client
	mu          sync.Mutex
	accessToken string
	tokenExpiry time.Time
	log         zerolog.Logger
}

// NewFCMClient creates an FCMClient from the JSON content of a Firebase service
// account key file (set FIREBASE_CREDENTIALS_JSON to its content).
func NewFCMClient(credentialsJSON string, log zerolog.Logger) (Pusher, error) {
	var sa serviceAccount
	if err := json.Unmarshal([]byte(credentialsJSON), &sa); err != nil {
		return nil, fmt.Errorf("firebase: parse credentials: %w", err)
	}
	if sa.ProjectID == "" || sa.PrivateKey == "" || sa.ClientEmail == "" {
		return nil, fmt.Errorf("firebase: credentials missing project_id, private_key, or client_email")
	}
	if sa.TokenURI == "" {
		sa.TokenURI = tokenURL
	}
	return &FCMClient{
		sa:         sa,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		log:        log,
	}, nil
}

// Send delivers msg as a Firebase push notification, records an OTel span, and
// appends a ThirdParty entry to the journal stored in ctx.
func (c *FCMClient) Send(ctx context.Context, msg Message) error {
	ctx, span := otel.Tracer("wapgo").Start(ctx, "notification.firebase.send")
	defer span.End()

	tok, err := c.cachedAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("firebase: get access token: %w", err)
	}

	endpoint := fmt.Sprintf(fcmEndpoint, c.sa.ProjectID)
	payload, err := buildPayload(msg)
	if err != nil {
		return err
	}

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("firebase: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")

	resp, doErr := c.httpClient.Do(req)
	latency := time.Since(start).Milliseconds()

	status := 0
	errStr := ""
	if doErr != nil {
		errStr = doErr.Error()
		err = doErr
		span.RecordError(doErr)
		span.SetStatus(codes.Error, doErr.Error())
	} else {
		status = resp.StatusCode
		io.Copy(io.Discard, io.LimitReader(resp.Body, 4*1024)) //nolint:errcheck
		resp.Body.Close()              //nolint:errcheck
		if resp.StatusCode != http.StatusOK {
			errStr = fmt.Sprintf("FCM HTTP %d", resp.StatusCode)
			span.SetStatus(codes.Error, errStr)
			err = fmt.Errorf("firebase: %s", errStr)
		}
	}

	journal.FromContext(ctx).AddThirdParty(journal.ThirdParty{
		Name:      "firebase-fcm",
		Method:    http.MethodPost,
		URL:       endpoint,
		Host:      "fcm.googleapis.com",
		Status:    status,
		LatencyMS: latency,
		Error:     errStr,
		StartedAt: start,
	})

	c.log.Debug().
		Str("token", maskToken(msg.Token)).
		Str("topic", msg.Topic).
		Str("title", msg.Title).
		Int64("latency_ms", latency).
		Err(err).
		Msg("notification.firebase.send")

	return err
}

// ── token management ────────────────────────────────────────────────────────

func (c *FCMClient) cachedAccessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.accessToken, nil
	}
	tok, expiry, err := c.fetchAccessToken(ctx)
	if err != nil {
		return "", err
	}
	c.accessToken = tok
	c.tokenExpiry = expiry
	return tok, nil
}

func (c *FCMClient) fetchAccessToken(ctx context.Context) (string, time.Time, error) {
	pk, err := parseRSAPrivateKey(c.sa.PrivateKey)
	if err != nil {
		return "", time.Time{}, err
	}

	now := time.Now()
	signed, err := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss":   c.sa.ClientEmail,
		"sub":   c.sa.ClientEmail,
		"scope": fcmScope,
		"aud":   c.sa.TokenURI,
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	}).SignedString(pk)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("firebase: sign jwt: %w", err)
	}

	body := url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  {signed},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.sa.TokenURI,
		strings.NewReader(body.Encode()))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("firebase: build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("firebase: token request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4*1024)).Decode(&result); err != nil {
		return "", time.Time{}, fmt.Errorf("firebase: decode token response: %w", err)
	}
	if result.AccessToken == "" {
		return "", time.Time{}, fmt.Errorf("firebase: empty access token from %s", c.sa.TokenURI)
	}

	// Subtract 30 s so we refresh before Google rejects the old token.
	expiry := now.Add(time.Duration(result.ExpiresIn)*time.Second - 30*time.Second)
	return result.AccessToken, expiry, nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

func buildPayload(msg Message) ([]byte, error) {
	notif := map[string]any{
		"title": msg.Title,
		"body":  msg.Body,
	}
	if msg.ImageURL != "" {
		notif["image"] = msg.ImageURL
	}

	fcmMsg := map[string]any{
		"notification": notif,
	}
	if msg.Token != "" {
		fcmMsg["token"] = msg.Token
	} else if msg.Topic != "" {
		fcmMsg["topic"] = msg.Topic
	}
	if len(msg.Data) > 0 {
		fcmMsg["data"] = msg.Data
	}

	return json.Marshal(map[string]any{"message": fcmMsg})
}

func parseRSAPrivateKey(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("firebase: invalid PEM private key")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("firebase: parse private key: %w", err)
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("firebase: private key is not RSA")
	}
	return rsaKey, nil
}

// maskToken shows only the first and last 4 chars to keep logs safe.
func maskToken(t string) string {
	if len(t) <= 8 {
		return "***"
	}
	return t[:4] + "***" + t[len(t)-4:]
}
