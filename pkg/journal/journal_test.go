package journal_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abdullahPrasetio/wapgo/pkg/journal"
	"github.com/abdullahPrasetio/wapgo/pkg/logger"
)

func TestStartAndFromContext(t *testing.T) {
	ctx, j := journal.Start(context.Background(), journal.KindAPI)
	require.NotNil(t, j)
	assert.Same(t, j, journal.FromContext(ctx))
}

func TestFromContext_NilWhenAbsent(t *testing.T) {
	assert.Nil(t, journal.FromContext(context.Background()))
}

func TestNilJournal_IsNoop(t *testing.T) {
	var j *journal.Journal // nil
	assert.NotPanics(t, func() {
		j.AddTrace("x", nil)
		j.AddThirdParty(journal.ThirdParty{})
		j.SetRequestID("r")
		assert.Nil(t, j.Traces())
		assert.Nil(t, j.ThirdParties())
	})
}

func TestAddThirdPartyAndTrace_AggregateAndDualWrite(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, logger.SetupSinks(logger.SinkConfig{Dir: dir, Rotation: "size"}))

	_, j := journal.Start(context.Background(), journal.KindAPI)
	j.SetRequestID("req-1")
	j.SetTraceID("trace-1")

	j.AddThirdParty(journal.ThirdParty{Name: "user-svc", Method: "GET", URL: "https://x/users/1", Status: 200})
	j.AddThirdParty(journal.ThirdParty{Name: "billing", Method: "POST", URL: "https://y/charge", Status: 201})
	j.AddTrace("risk-score", map[string]any{"score": 42})

	// Aggregated into the parent record.
	assert.Len(t, j.ThirdParties(), 2)
	assert.Len(t, j.Traces(), 1)

	// Dual-written to their own files.
	tp, err := os.ReadFile(filepath.Join(dir, "thirdparty.log"))
	require.NoError(t, err)
	assert.Equal(t, 2, strings.Count(string(tp), "\n"), "two thirdparty lines")
	assert.Contains(t, string(tp), `"request_id":"req-1"`)
	assert.Contains(t, string(tp), `"name":"billing"`)

	tr, err := os.ReadFile(filepath.Join(dir, "trace.log"))
	require.NoError(t, err)
	assert.Contains(t, string(tr), `"name":"risk-score"`)
	assert.Contains(t, string(tr), `"score":42`)
}

func TestRedactHeaders(t *testing.T) {
	in := map[string]string{
		"Authorization": "Bearer secret",
		"Content-Type":  "application/json",
		"Cookie":        "session=abc",
	}
	out := journal.RedactHeaders(in)
	assert.Equal(t, "[redacted]", out["Authorization"])
	assert.Equal(t, "[redacted]", out["Cookie"])
	assert.Equal(t, "application/json", out["Content-Type"])
}

func TestCapBody(t *testing.T) {
	assert.Equal(t, "[omitted]", journal.CapBody([]byte("x"), "application/json", 0))
	assert.Equal(t, "[binary omitted]", journal.CapBody([]byte("x"), "image/png", 100))
	assert.Equal(t, "hello", journal.CapBody([]byte("hello"), "application/json", 100))

	long := strings.Repeat("a", 50)
	got := journal.CapBody([]byte(long), "text/plain", 10)
	assert.True(t, strings.HasSuffix(got, "...[truncated]"))
	assert.Contains(t, got, strings.Repeat("a", 10))
}
