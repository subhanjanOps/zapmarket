// Package authctx carries the authenticated user (set by internal/middleware
// after a successful token validation) on the request context. It exists as
// its own package — rather than living on internal/middleware, which
// handlers would otherwise need to import — so that internal/handler/http
// can read the authenticated user without an import cycle
// (middleware already imports handler/http for its error-response helpers).
package authctx

import (
	"context"

	authpb "github.com/zapmarket/zapmarket/pkg/proto/auth"
)

type contextKey string

const userContextKey contextKey = "auth_user"

// WithUser returns a copy of ctx carrying the authenticated user.
func WithUser(ctx context.Context, user *authpb.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// UserFromContext retrieves the authenticated user from the request context.
// Returns nil if the context carries no user (unauthenticated request).
func UserFromContext(ctx context.Context) *authpb.User {
	user, _ := ctx.Value(userContextKey).(*authpb.User)
	return user
}
