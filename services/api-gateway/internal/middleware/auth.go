package middleware

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	authpb "github.com/zapmarket/zapmarket/pkg/proto/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type authUserKey contextKey

const authUserCtxKey authUserKey = "auth_user"

type AuthUser struct {
	ID    string
	Email string
	Role  string
}

// AuthMiddleware validates Bearer tokens via auth-service gRPC ValidateToken.
type AuthMiddleware struct {
	client authpb.AuthServiceClient
	conn   *grpc.ClientConn
}

// NewAuthMiddleware dials auth-service. Uses TLS except when APP_ENV=development,
// where insecure credentials are used to support local docker-compose stacks.
func NewAuthMiddleware(authAddr string) (*AuthMiddleware, error) {
	var dialOpt grpc.DialOption
	if os.Getenv("APP_ENV") == "development" {
		dialOpt = grpc.WithTransportCredentials(insecure.NewCredentials())
	} else {
		dialOpt = grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12}))
	}
	conn, err := grpc.NewClient(authAddr, dialOpt)
	if err != nil {
		return nil, err
	}
	return &AuthMiddleware{client: authpb.NewAuthServiceClient(conn), conn: conn}, nil
}

// Close drains and closes the underlying gRPC connection. Call after the HTTP
// server has shut down so in-flight auth RPCs finish before the channel closes.
func (m *AuthMiddleware) Close() error {
	return m.conn.Close()
}

// Authenticate is chi middleware. Public routes must be registered before
// protected ones (the router skips this middleware for them).
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r)
		if !ok {
			jsonError(w, http.StatusUnauthorized, "MISSING_TOKEN", "authorization header is required")
			return
		}

		resp, err := m.client.ValidateToken(r.Context(), &authpb.ValidateTokenRequest{Token: token})
		if err != nil || !resp.GetValid() {
			msg := "invalid or expired token"
			if resp != nil && resp.ErrorMessage != "" {
				msg = resp.ErrorMessage
			}
			jsonError(w, http.StatusUnauthorized, "INVALID_TOKEN", msg)
			return
		}

		u := &AuthUser{ID: resp.User.GetId(), Email: resp.User.GetEmail(), Role: resp.User.GetRole()}
		ctx := context.WithValue(r.Context(), authUserCtxKey, u)
		// Forward user identity to downstream services.
		r.Header.Set("X-User-ID", u.ID)
		r.Header.Set("X-User-Email", u.Email)
		r.Header.Set("X-User-Role", u.Role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserFromContext(ctx context.Context) *AuthUser {
	u, _ := ctx.Value(authUserCtxKey).(*AuthUser)
	return u
}

func bearerToken(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	if h == "" {
		return "", false
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", false
	}
	t := strings.TrimSpace(parts[1])
	return t, t != ""
}

func jsonError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": false,
		"error":   map[string]string{"code": code, "message": message},
	})
}
