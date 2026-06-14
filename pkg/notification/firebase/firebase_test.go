package firebase

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateRSAPEM returns a PKCS8 PEM-encoded 2048-bit RSA private key.
func generateRSAPEM(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
}

func nopLog() zerolog.Logger { return zerolog.Nop() }

// ── maskToken ─────────────────────────────────────────────────────────────────

func TestMaskToken_ShortToken(t *testing.T) {
	assert.Equal(t, "***", maskToken("abc"))
	assert.Equal(t, "***", maskToken(""))
	assert.Equal(t, "***", maskToken("12345678"))
}

func TestMaskToken_LongToken(t *testing.T) {
	result := maskToken("abcdefghij")
	assert.Equal(t, "abcd***ghij", result)
}

// ── buildPayload ──────────────────────────────────────────────────────────────

func TestBuildPayload_WithToken(t *testing.T) {
	msg := Message{Token: "tok123", Title: "Hello", Body: "World"}
	raw, err := buildPayload(msg)
	require.NoError(t, err)

	var out map[string]any
	require.NoError(t, json.Unmarshal(raw, &out))
	inner := out["message"].(map[string]any)
	assert.Equal(t, "tok123", inner["token"])
	notif := inner["notification"].(map[string]any)
	assert.Equal(t, "Hello", notif["title"])
	assert.Equal(t, "World", notif["body"])
}

func TestBuildPayload_WithTopic(t *testing.T) {
	msg := Message{Topic: "news", Title: "Breaking", Body: "Story"}
	raw, err := buildPayload(msg)
	require.NoError(t, err)

	var out map[string]any
	require.NoError(t, json.Unmarshal(raw, &out))
	inner := out["message"].(map[string]any)
	assert.Equal(t, "news", inner["topic"])
	assert.Nil(t, inner["token"])
}

func TestBuildPayload_WithImageURL(t *testing.T) {
	msg := Message{Token: "t", Title: "T", Body: "B", ImageURL: "https://img.example.com/x.png"}
	raw, err := buildPayload(msg)
	require.NoError(t, err)

	var out map[string]any
	require.NoError(t, json.Unmarshal(raw, &out))
	inner := out["message"].(map[string]any)
	notif := inner["notification"].(map[string]any)
	assert.Equal(t, "https://img.example.com/x.png", notif["image"])
}

func TestBuildPayload_WithData(t *testing.T) {
	msg := Message{Token: "t", Title: "T", Body: "B", Data: map[string]string{"order": "99"}}
	raw, err := buildPayload(msg)
	require.NoError(t, err)

	var out map[string]any
	require.NoError(t, json.Unmarshal(raw, &out))
	inner := out["message"].(map[string]any)
	data := inner["data"].(map[string]any)
	assert.Equal(t, "99", data["order"])
}

func TestBuildPayload_NoData_OmitsDataKey(t *testing.T) {
	msg := Message{Token: "t", Title: "T", Body: "B"}
	raw, err := buildPayload(msg)
	require.NoError(t, err)

	var out map[string]any
	require.NoError(t, json.Unmarshal(raw, &out))
	inner := out["message"].(map[string]any)
	_, hasData := inner["data"]
	assert.False(t, hasData)
}

// ── parseRSAPrivateKey ────────────────────────────────────────────────────────

func TestParseRSAPrivateKey_InvalidPEM(t *testing.T) {
	_, err := parseRSAPrivateKey("not-a-pem")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid PEM")
}

func TestParseRSAPrivateKey_ValidKey(t *testing.T) {
	pemStr := generateRSAPEM(t)
	key, err := parseRSAPrivateKey(pemStr)
	require.NoError(t, err)
	assert.NotNil(t, key)
}

// ── NewFCMClient ──────────────────────────────────────────────────────────────

func TestNewFCMClient_InvalidJSON(t *testing.T) {
	_, err := NewFCMClient("not-json", nopLog())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse credentials")
}

func TestNewFCMClient_MissingFields(t *testing.T) {
	creds := `{"project_id":"","private_key":"","client_email":""}`
	_, err := NewFCMClient(creds, nopLog())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing")
}

func TestNewFCMClient_MissingProjectID(t *testing.T) {
	creds := `{"project_id":"","private_key":"key","client_email":"svc@proj.iam.gserviceaccount.com"}`
	_, err := NewFCMClient(creds, nopLog())
	require.Error(t, err)
}

func TestNewFCMClient_Valid(t *testing.T) {
	creds := `{"project_id":"my-proj","private_key":"key","client_email":"svc@proj.iam.gserviceaccount.com"}`
	client, err := NewFCMClient(creds, nopLog())
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNewFCMClient_DefaultsTokenURI(t *testing.T) {
	creds := `{"project_id":"p","private_key":"k","client_email":"e@x.com"}`
	client, err := NewFCMClient(creds, nopLog())
	require.NoError(t, err)
	impl := client.(*FCMClient)
	assert.Equal(t, tokenURL, impl.sa.TokenURI)
}

// ── HealthCheck ───────────────────────────────────────────────────────────────

func TestFirebaseHealthCheck_EmptyCredentials(t *testing.T) {
	probe := HealthCheck("")
	result := probe(context.Background())
	assert.Contains(t, result, "error:")
	assert.Contains(t, result, "FIREBASE_CREDENTIALS_JSON")
}

func TestFirebaseHealthCheck_InvalidJSON(t *testing.T) {
	probe := HealthCheck("not-json")
	result := probe(context.Background())
	assert.Contains(t, result, "error:")
	assert.Contains(t, result, "invalid credentials JSON")
}

func TestFirebaseHealthCheck_MissingFields(t *testing.T) {
	probe := HealthCheck(`{"project_id":"","client_email":""}`)
	result := probe(context.Background())
	assert.Contains(t, result, "error:")
	assert.Contains(t, result, "missing")
}

// ── cachedAccessToken ─────────────────────────────────────────────────────────

func TestCachedAccessToken_ReturnsCachedToken(t *testing.T) {
	c := &FCMClient{}
	c.accessToken = "cached-token"
	c.tokenExpiry = time.Now().Add(time.Hour)

	tok, err := c.cachedAccessToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "cached-token", tok)
}
