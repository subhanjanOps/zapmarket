package proxy

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/sony/gobreaker/v2"
	"github.com/zapmarket/zapmarket/services/api-gateway/internal/middleware"
)

// Upstream holds a reverse proxy + circuit breaker for one downstream service.
type Upstream struct {
	name      string
	logger    *slog.Logger
	breaker   *gobreaker.CircuitBreaker[struct{}]
	OnRequest func(upstream string, status int, latency time.Duration)

	mu      sync.Mutex
	proxies map[string]*httputil.ReverseProxy
}

// New creates an Upstream for the named service.
func New(name string, logger *slog.Logger) *Upstream {
	cb := gobreaker.NewCircuitBreaker[struct{}](gobreaker.Settings{
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
	return &Upstream{
		name:    name,
		logger:  logger,
		breaker: cb,
		proxies: make(map[string]*httputil.ReverseProxy),
	}
}

// ServeHTTP proxies the request to targetAddr.
func (u *Upstream) ServeHTTP(w http.ResponseWriter, r *http.Request, targetAddr string) {
	start := time.Now()
	status := http.StatusOK

	_, err := u.breaker.Execute(func() (struct{}, error) {
		rp := u.getProxy(targetAddr)
		rec := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		rp.ServeHTTP(rec, r)
		status = rec.status
		if rec.status >= 500 {
			return struct{}{}, fmt.Errorf("upstream %s returned %d", u.name, rec.status)
		}
		return struct{}{}, nil
	})

	if err == gobreaker.ErrOpenState || err == gobreaker.ErrTooManyRequests {
		status = http.StatusServiceUnavailable
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"success":false,"error":{"code":"CIRCUIT_OPEN","message":"service temporarily unavailable"}}`))
	}

	if u.OnRequest != nil {
		u.OnRequest(u.name, status, time.Since(start))
	}
}

// CircuitState returns the current circuit breaker state as a string.
func (u *Upstream) CircuitState() string {
	return u.breaker.State().String()
}

func (u *Upstream) getProxy(addr string) *httputil.ReverseProxy {
	u.mu.Lock()
	defer u.mu.Unlock()

	if rp, ok := u.proxies[addr]; ok {
		return rp
	}

	target, err := url.Parse(addr)
	if err != nil {
		u.logger.Error("invalid upstream addr", "addr", addr, "error", err)
		target, _ = url.Parse("http://localhost:1")
	}

	rp := httputil.NewSingleHostReverseProxy(target)
	rp.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		u.logger.Error("proxy error", "upstream", u.name, "addr", addr, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"success":false,"error":{"code":"UPSTREAM_ERROR","message":"upstream service unavailable"}}`))
	}

	origDirector := rp.Director
	rp.Director = func(req *http.Request) {
		origDirector(req)
		req.Host = target.Host
		if id := middleware.GetRequestID(req.Context()); id != "" {
			req.Header.Set("X-Request-ID", id)
		}
	}

	u.proxies[addr] = rp
	return rp
}

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
