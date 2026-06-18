package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	authpb "github.com/zapmarket/zapmarket/pkg/proto/auth"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/authctx"
	httputil "github.com/zapmarket/zapmarket/services/product-catalog-service/internal/handler/http"
)

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

		ctx := authctx.WithUser(r.Context(), resp.User)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AuthenticateOptional is like Authenticate but does not reject unauthenticated
// requests. If a valid Bearer token is present, the user is stored in context
// so downstream handlers can apply role-based filtering (e.g. sellers see only
// their own products). Requests without a token, or with an invalid token, pass
// through with no user in context.
func (m *AuthMiddleware) AuthenticateOptional(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token, ok := extractBearerToken(r); ok {
			resp, err := m.authClient.ValidateToken(r.Context(), &authpb.ValidateTokenRequest{Token: token})
			if err == nil && resp.Valid {
				r = r.WithContext(authctx.WithUser(r.Context(), resp.User))
			}
		}
		next.ServeHTTP(w, r)
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
			user := authctx.UserFromContext(r.Context())
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
