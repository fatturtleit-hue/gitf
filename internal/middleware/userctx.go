package middleware

import (
	"context"

	"gitf/internal/models"
)

type userCtxKey struct{}

// WithUser stores the authenticated user in the request context.
// Lives in middleware (not handlers) because the auth middleware is the
// producer; handlers and other middleware are consumers. Keeping the producer
// owner of the key avoids an import cycle between handlers and middleware.
func WithUser(ctx context.Context, user *models.User) context.Context {
	return context.WithValue(ctx, userCtxKey{}, user)
}

// UserFromContext retrieves the authenticated user from context.
func UserFromContext(ctx context.Context) (*models.User, bool) {
	user, ok := ctx.Value(userCtxKey{}).(*models.User)
	if !ok || user == nil {
		return nil, false
	}
	return user, true
}
