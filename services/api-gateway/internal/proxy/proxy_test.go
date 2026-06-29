package proxy_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zapmarket/zapmarket/services/api-gateway/internal/proxy"
)

// newFailingServer returns a test server that always responds with the given status code.
func newFailingServer(statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
	}))
}

// newSuccessServer returns a test server that always responds 200 with a JSON body.
func newSuccessServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
}

// serveOnce is a helper to call upstream.ServeHTTP and capture the response.
func serveOnce(upstream *proxy.Upstream, targetAddr string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	upstream.ServeHTTP(rec, req, targetAddr)
	return rec
}

// TestUpstream_OpensAfterFiveConsecutiveFailures trips the circuit breaker by
// sending 5 requests that produce upstream 5xx responses, then asserts that the
// 6th request is short-circuited with status 503 and a CIRCUIT_OPEN error code
// without reaching the upstream at all.
func TestUpstream_OpensAfterFiveConsecutiveFailures(t *testing.T) {
	srv := newFailingServer(http.StatusInternalServerError)
	defer srv.Close()

	upstream := proxy.New("test-upstream", slog.Default())

	// Trip the breaker: 5 consecutive 5xx responses.
	for i := 0; i < 5; i++ {
		rec := serveOnce(upstream, srv.URL)
		// Each of the first 5 calls reaches the upstream and gets proxied through.
		// The upstream returns 500 which is recorded internally; the breaker counts failures.
		if rec.Code == http.StatusServiceUnavailable {
			t.Fatalf("call %d: circuit opened prematurely", i+1)
		}
	}

	// 6th call — breaker should be open now.
	rec := serveOnce(upstream, srv.URL)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 from open circuit, got %d", rec.Code)
	}

	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}

	var payload struct {
		Success bool `json:"success"`
		Error   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("body is not valid JSON: %v — body: %s", err, body)
	}
	if payload.Success {
		t.Error("expected success=false in CIRCUIT_OPEN response")
	}
	if payload.Error.Code != "CIRCUIT_OPEN" {
		t.Errorf("expected error.code=CIRCUIT_OPEN, got %q", payload.Error.Code)
	}

	// Confirm state via CircuitState.
	state := upstream.CircuitState()
	if state != "open" {
		t.Errorf("expected circuit state 'open', got %q", state)
	}
}

// TestUpstream_PassesThroughSuccessfulRequests verifies that when the upstream
// returns 200, the proxy forwards the response correctly and the circuit breaker
// remains closed.
func TestUpstream_PassesThroughSuccessfulRequests(t *testing.T) {
	srv := newSuccessServer()
	defer srv.Close()

	upstream := proxy.New("test-upstream-ok", slog.Default())

	for i := 0; i < 3; i++ {
		rec := serveOnce(upstream, srv.URL)
		if rec.Code != http.StatusOK {
			t.Fatalf("call %d: expected 200, got %d", i+1, rec.Code)
		}
	}

	state := upstream.CircuitState()
	if state != "closed" {
		t.Errorf("expected circuit to remain 'closed' after successful requests, got %q", state)
	}
}
