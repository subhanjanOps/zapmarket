package http

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	authpb "github.com/zapmarket/zapmarket/services/currency-service/proto/authpb"
)

type userContextKey struct{}

// userInfo holds the minimal fields extracted from a validated JWT.
type userInfo struct {
	ID   string
	Role string
}

// AuthMiddleware validates JWTs via auth-service gRPC.
type AuthMiddleware struct {
	conn       *grpc.ClientConn
	authClient authpb.AuthServiceClient
	log        *slog.Logger
}

// NewAuthMiddleware dials auth-service and returns a middleware ready to use.
func NewAuthMiddleware(addr string, log *slog.Logger) (*AuthMiddleware, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &AuthMiddleware{conn: conn, authClient: authpb.NewAuthServiceClient(conn), log: log}, nil
}

// Close releases the gRPC connection.
func (m *AuthMiddleware) Close() error { return m.conn.Close() }

// RequireRole returns middleware that validates the JWT and enforces the given role.
func (m *AuthMiddleware) RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := extractBearer(r)
			if !ok {
				http.Error(w, `{"error":"authorization header is required"}`, http.StatusUnauthorized)
				return
			}

			resp, err := m.authClient.ValidateToken(r.Context(), &authpb.ValidateTokenRequest{Token: token})
			if err != nil {
				m.log.Error("auth-service ValidateToken failed", "error", err)
				http.Error(w, `{"error":"could not validate token"}`, http.StatusUnauthorized)
				return
			}
			if !resp.Valid {
				http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}
			if resp.User == nil || resp.User.GetRole() != role {
				http.Error(w, `{"error":"admin role required"}`, http.StatusForbidden)
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey{}, &userInfo{
				ID:   resp.User.GetId(),
				Role: resp.User.GetRole(),
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractBearer(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") || parts[1] == "" {
		return "", false
	}
	return strings.TrimSpace(parts[1]), true
}
