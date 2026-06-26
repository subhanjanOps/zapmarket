package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type contextKey string

const requestIDKey contextKey = "request_id"

// RequestID injects a unique X-Request-ID header into every request,
// preserving any ID the client already sent.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		// Only accept the client-supplied ID if it is a valid UUID; otherwise
		// generate a fresh one to prevent audit log poisoning.
		if id == "" || uuid.Validate(id) != nil {
			id = uuid.New().String()
		}
		r.Header.Set("X-Request-ID", id)
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// SanitizeIdentityHeaders strips client-supplied copies of the trusted identity
// headers from every inbound request before any auth check runs. Only the
// Authenticate middleware may set these after successful token validation.
func SanitizeIdentityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Del("X-User-ID")
		r.Header.Del("X-User-Email")
		r.Header.Del("X-User-Role")
		next.ServeHTTP(w, r)
	})
}

func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}
