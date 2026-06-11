package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	httputil "github.com/zapmarket/zapmarket/services/product-catalog-service/internal/handler/http"
	authpb "github.com/zapmarket/zapmarket/services/product-catalog-service/proto/authpb"
)

type contextKey string

const userContextKey contextKey = "auth_user"

// AuthMiddleware validates JWTs via the auth-service gRPC endpoint.
type AuthMiddleware struct {
	authClient authpb.AuthServiceClient
	logger     *slog.Logger
}

// NewAuthMiddleware dials the auth-service and returns a ready middleware.
func NewAuthMiddleware(addr string, logger *slog.Logger) (*AuthMiddleware, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &AuthMiddleware{
		authClient: authpb.NewAuthServiceClient(conn),
		logger:     logger,
	}, nil
}

// Authenticate validates the Bearer token on the request.
// On success it stores the authenticated user in the request context.
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := extractBearerToken(r)
		if !ok {
			httputil.ErrorResponse(w, http.StatusUnauthorized, "MISSING_TOKEN", "authorization header is required")
			return
		}

		resp, err := m.authClient.ValidateToken(r.Context(), &authpb.ValidateTokenRequest{Token: token})
		if err != nil {
			m.logger.Error("auth-service ValidateToken failed", "error", err)
			httputil.ErrorResponse(w, http.StatusUnauthorized, "AUTH_SERVICE_ERROR", "could not validate token")
			return
		}

		if !resp.Valid {
			httputil.ErrorResponse(w, http.StatusUnauthorized, "INVALID_TOKEN", resp.ErrorMessage)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, resp.User)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole returns middleware that allows only users whose role is in the provided list.
// Must be chained after Authenticate.
func (m *AuthMiddleware) RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := UserFromContext(r.Context())
			if user == nil {
				httputil.ErrorResponse(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
				return
			}

			if _, ok := allowed[user.Role]; !ok {
				httputil.ErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "you do not have permission to perform this action")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// UserFromContext retrieves the authenticated user from the request context.
// Returns nil if the context carries no user (unauthenticated request).
func UserFromContext(ctx context.Context) *authpb.User {
	user, _ := ctx.Value(userContextKey).(*authpb.User)
	return user
}

// extractBearerToken parses "Authorization: Bearer <token>" from the request header.
func extractBearerToken(r *http.Request) (string, bool) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return "", false
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", false
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", false
	}
	return token, true
}
