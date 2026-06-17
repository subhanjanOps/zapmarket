package proxy

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/sony/gobreaker/v2"
	"github.com/zapmarket/zapmarket/services/api-gateway/internal/middleware"
)

// Upstream holds a reverse proxy + circuit breaker for one downstream service.
type Upstream struct {
	name    string
	proxy   *httputil.ReverseProxy
	breaker *gobreaker.CircuitBreaker[*http.Response]
}

// New creates an Upstream that proxies to targetURL.
func New(name, targetURL string, logger *slog.Logger) (*Upstream, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}

	rp := httputil.NewSingleHostReverseProxy(target)
	rp.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logger.Error("proxy error", "upstream", name, "error", err)
		http.Error(w, `{"success":false,"error":{"code":"UPSTREAM_ERROR","message":"upstream service unavailable"}}`, http.StatusBadGateway)
	}

	// Strip the /api/v1 gateway prefix before forwarding.
	origDirector := rp.Director
	rp.Director = func(req *http.Request) {
		origDirector(req)
		req.Host = target.Host
		// Forward the request ID so downstream logs correlate.
		if id := middleware.GetRequestID(req.Context()); id != "" {
			req.Header.Set("X-Request-ID", id)
		}
	}

	cb := gobreaker.NewCircuitBreaker[*http.Response](gobreaker.Settings{
		Name:        name,
		MaxRequests: 5,
		Interval:    30 * time.Second,
		Timeout:     10 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 5
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			logger.Warn("circuit breaker state change", "upstream", name, "from", from, "to", to)
		},
	})

	return &Upstream{name: name, proxy: rp, breaker: cb}, nil
}

// ServeHTTP implements http.Handler — routes the request through the circuit breaker.
func (u *Upstream) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, err := u.breaker.Execute(func() (*http.Response, error) {
		rec := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		u.proxy.ServeHTTP(rec, r)
		if rec.status >= 500 {
			return nil, fmt.Errorf("upstream %s returned %d", u.name, rec.status)
		}
		return nil, nil
	})

	if err != nil {
		if err == gobreaker.ErrOpenState || err == gobreaker.ErrTooManyRequests {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"success":false,"error":{"code":"CIRCUIT_OPEN","message":"service temporarily unavailable"}}`))
			return
		}
	}
}

// responseRecorder captures the status code so the circuit breaker can count failures.
type responseRecorder struct {
	http.ResponseWriter
	status  int
	written bool
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.written = true
	r.ResponseWriter.WriteHeader(status)
}
