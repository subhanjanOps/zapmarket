// Package authctx carries the authenticated user set by internal/middleware
// on the request context. Kept in its own package to avoid an import cycle
// between middleware (which imports handler/http for error helpers) and
// handler/http (which needs to read the authenticated user).
package authctx

import (
	"context"

	authpb "github.com/zapmarket/zapmarket/pkg/proto/auth"
)

type contextKey string

const userContextKey contextKey = "auth_user"

func WithUser(ctx context.Context, user *authpb.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func UserFromContext(ctx context.Context) *authpb.User {
	user, _ := ctx.Value(userContextKey).(*authpb.User)
	return user
}
