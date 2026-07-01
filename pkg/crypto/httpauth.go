package crypto

import (
	"context"
	"net/http"
	"strings"
)

type ctxKey string

const claimsCtxKey ctxKey = "auth_claims"

// ExtractBearerToken returns the token from the Authorization header, if present.
func ExtractBearerToken(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(h, prefix))
	if token == "" {
		return "", false
	}
	return token, true
}

// WithClaims stores validated claims on the context.
func WithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsCtxKey, claims)
}

// ClaimsFromContext retrieves claims stored by WithClaims/RequireAuth.
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsCtxKey).(*Claims)
	return claims, ok
}

// RequireAuth returns HTTP middleware that validates the Bearer access token
// using ValidateAccessToken and stores the resulting Claims in the request
// context. Requests without a valid token receive 401.
//
// This performs local JWT verification (shared secret), avoiding an
// inter-service gRPC round trip for every request. It requires the caller's
// service to hold the same JWT_SECRET_KEY as auth-service.
func RequireAuth(secretKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := ExtractBearerToken(r)
			if !ok {
				writeAuthError(w, http.StatusUnauthorized, "authorization header is required")
				return
			}
			claims, err := ValidateAccessToken(token, secretKey)
			if err != nil {
				writeAuthError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}
			next.ServeHTTP(w, r.WithContext(WithClaims(r.Context(), claims)))
		})
	}
}

// OptionalAuth validates a Bearer token if present and stores the resulting
// Claims in context, but never rejects the request — used for endpoints that
// accept both anonymous and authenticated callers (e.g. analytics events).
// An invalid/expired token is treated the same as no token: the request
// proceeds unauthenticated rather than failing.
func OptionalAuth(secretKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token, ok := ExtractBearerToken(r); ok {
				if claims, err := ValidateAccessToken(token, secretKey); err == nil {
					r = r.WithContext(WithClaims(r.Context(), claims))
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole returns middleware that only allows requests whose authenticated
// role (set by RequireAuth) is in the allowed list. Must be chained after RequireAuth.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				writeAuthError(w, http.StatusUnauthorized, "authentication required")
				return
			}
			if _, permitted := allowed[claims.Role]; !permitted {
				writeAuthError(w, http.StatusForbidden, "insufficient role")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeAuthError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"success":false,"message":"` + message + `"}`))
}
