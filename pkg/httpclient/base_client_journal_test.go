package httpclient

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abdullahPrasetio/wapgo/pkg/journal"
	applogger "github.com/abdullahPrasetio/wapgo/pkg/logger"
)

func TestClient_Do_RecordsThirdPartyInJournal(t *testing.T) {
	require.NoError(t, applogger.SetupSinks(applogger.SinkConfig{Dir: t.TempDir()}))

	c := testClient(mockTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	}))

	ctx, j := journal.Start(context.Background(), journal.KindAPI)
	j.SetRequestID("req-9")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.example.com/x", nil)
	require.NoError(t, err)

	resp, err := c.Do(ctx, req)
	require.NoError(t, err)

	// The caller can still read the (buffered) response body.
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "ok")

	tps := j.ThirdParties()
	require.Len(t, tps, 1)
	assert.Equal(t, http.MethodGet, tps[0].Method)
	assert.Equal(t, 200, tps[0].Status)
	assert.Equal(t, "api.example.com", tps[0].Host)
	assert.Contains(t, tps[0].ResponseBody, "ok")
}

func TestClient_Do_NoJournal_Unchanged(t *testing.T) {
	c := testClient(mockTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("hi"))}, nil
	}))

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://api.example.com/x", nil)
	require.NoError(t, err)

	resp, err := c.Do(context.Background(), req)
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "hi", string(body))
}
