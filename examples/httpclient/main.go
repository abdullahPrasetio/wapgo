// Example: resilient inter-service HTTP client — retry, circuit breaker, SSRF guard.
//
// This example starts a local mock server, then shows retry on 503, circuit-breaker
// tripping after consecutive failures, and SSRF protection blocking internal hosts.
//
// Run:
//
//	cd examples/httpclient && go run main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"time"

	"github.com/abdullahPrasetio/wapgo/pkg/httpclient"
)

func main() {
	// ── Demo 1: Retry on 503 ─────────────────────────────────────────────────
	fmt.Println("=== Demo 1: Retry on 503 ===")
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		if n < 3 {
			fmt.Printf("  server: call %d → 503\n", n)
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		fmt.Printf("  server: call %d → 200 OK\n", n)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	}))
	defer server.Close()

	client := httpclient.New(httpclient.Options{
		Timeout:    2 * time.Second,
		MaxRetries: 3,
		// Allow the test server's loopback address explicitly via AllowedHosts.
		// In production omit AllowedHosts and rely on the default internal-block rules.
		AllowedHosts: []string{"127.0.0.1"},
	})

	ctx := context.Background()
	resp, body, err := client.Get(ctx, server.URL+"/ping")
	if err != nil {
		log.Fatalf("Get: %v", err)
	}
	fmt.Printf("  final status: %d, body: %s (after %d attempts)\n", resp.StatusCode, body, callCount.Load())

	// ── Demo 2: Circuit breaker ───────────────────────────────────────────────
	fmt.Println("\n=== Demo 2: Circuit breaker opens after 5 consecutive failures ===")
	alwaysFail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer alwaysFail.Close()

	cb := httpclient.New(httpclient.Options{
		Timeout:               500 * time.Millisecond,
		MaxRetries:            1,
		CBConsecutiveFailures: 3,
		CBTimeout:             5 * time.Second,
		AllowedHosts:          []string{"127.0.0.1"},
	})

	for i := range 6 {
		_, _, err := cb.Get(ctx, alwaysFail.URL+"/fail")
		fmt.Printf("  call %d: err=%v\n", i+1, err)
	}

	// ── Demo 3: SSRF protection ───────────────────────────────────────────────
	fmt.Println("\n=== Demo 3: SSRF protection blocks internal addresses ===")
	ssrfClient := httpclient.New(httpclient.Options{
		Timeout: 1 * time.Second,
		// No AllowedHosts → default policy blocks loopback/private
	})
	_, _, err = ssrfClient.Get(ctx, "http://169.254.169.254/metadata") // AWS metadata endpoint
	fmt.Printf("  link-local blocked: %v\n", err)

	_, _, err = ssrfClient.Get(ctx, "http://10.0.0.1/internal")
	fmt.Printf("  private range blocked: %v\n", err)
}
