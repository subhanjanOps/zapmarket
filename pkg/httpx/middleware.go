package httpx

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type requestIDCtxKey struct{}

// RequestIDFromContext returns the request ID stored by RequestID, if any.
func RequestIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(requestIDCtxKey{}).(string)
	return id, ok
}

// MaxBodyBytes is the default request body size limit (4 MiB).
const MaxBodyBytes int64 = 4 << 20

// LimitBody wraps each request body with http.MaxBytesReader to prevent
// memory exhaustion from oversized payloads. Handlers should check for
// *http.MaxBytesError when decoding to return 413.
func LimitBody(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

const RequestIDHeader = "X-Request-ID"

// RequestID propagates or generates an X-Request-ID header and stores it in the request context.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(RequestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}
		w.Header().Set(RequestIDHeader, id)
		ctx := context.WithValue(r.Context(), requestIDCtxKey{}, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Logger logs each request's method, path, status, duration, and request ID using slog.
func Logger(l *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rw, r)
			reqID, _ := RequestIDFromContext(r.Context())
			l.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.status,
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", reqID,
			)
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}
